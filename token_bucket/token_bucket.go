package token_bucket

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/fedragon/rate-limiter/concurrent"
)

type (
	Path   string
	UserID string

	Rate struct {
		Value    int
		Interval time.Duration
	}

	Limit struct {
		Limit  Rate
		Refill Rate
	}

	RateLimiterBuilder struct {
		pathLimits *concurrent.Map[Path, Limit]
		users      *concurrent.Set[UserID]
	}

	RateLimiter struct {
		pathLimits *concurrent.Map[Path, Limit]
		userQuotas *concurrent.Map[UserID, *concurrent.Map[Path, int]]
		cancel     context.CancelFunc
	}
)

func (b *RateLimiterBuilder) SetLimit(path string, limit Limit) *RateLimiterBuilder {
	if b.pathLimits == nil {
		b.pathLimits = concurrent.NewMap[Path, Limit]()
	}
	b.pathLimits.Put(Path(path), limit)

	return b
}

func (b *RateLimiterBuilder) RegisterUser(ID string) *RateLimiterBuilder {
	if b.users == nil {
		b.users = concurrent.NewSet[UserID]()
	}

	b.users.Put(UserID(ID))

	return b
}

func (b *RateLimiterBuilder) Build() (*RateLimiter, error) {
	ctx, cancel := context.WithCancel(context.Background())

	if b.pathLimits == nil {
		cancel()
		return nil, errors.New("no rate limits configured")
	}

	rl := RateLimiter{
		pathLimits: b.pathLimits,
		userQuotas: concurrent.NewMap[UserID, *concurrent.Map[Path, int]](),
		cancel:     cancel,
	}

	if b.users == nil {
		b.users = concurrent.NewSet[UserID]()
	}

	b.users.ForEach(func(u UserID) {
		b.pathLimits.ForEach(func(path Path, limit Limit) {
			uqs, ok := rl.userQuotas.Get(u)

			if !ok {
				uqs = concurrent.NewMap[Path, int]()
				rl.userQuotas.Put(u, uqs)
			}

			uqs.Put(path, limit.Limit.Value)
		})
	})

	b.pathLimits.ForEach(func(p Path, v Limit) {
		go rl.refill(ctx, p, v)
	})

	return &rl, nil
}

func (rl *RateLimiter) Stop() {
	rl.cancel()
}

func (rl *RateLimiter) getQuota(userID UserID, path Path) (int, bool) {
	if user, exists := rl.userQuotas.Get(userID); exists {
		return user.Get(path)
	}

	return 0, false
}

func (rl *RateLimiter) decrQuota(userID UserID, path Path) {
	user, ok := rl.userQuotas.Get(userID)
	if !ok {
		return
	}

	if quota, ok := user.Get(path); ok && quota > 0 {
		user.Put(path, quota-1)
	}
}

func (rl *RateLimiter) getRefillInterval(path Path) time.Duration {
	limit, ok := rl.pathLimits.Get(path)

	if !ok {
		return 0
	}
	return limit.Refill.Interval
}

func (rl *RateLimiter) refillQuotas(path Path, value int, max int) {
	rl.userQuotas.ForEach(func(k UserID, quotas *concurrent.Map[Path, int]) {
		current, _ := quotas.Get(path)
		next := current + value
		if next > max {
			next = max
		}

		quotas.Put(path, next)
	})
}

func (rl *RateLimiter) refill(ctx context.Context, path Path, limit Limit) {
	ticker := time.NewTicker(limit.Limit.Interval)
	defer ticker.Stop()

	fmt.Printf("starting refiller for path '%v'\n", path)

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("stopping refiller for path '%v'\n", path)
			return
		case <-ticker.C:
			rl.refillQuotas(path, limit.Refill.Value, limit.Limit.Value)
		default:
		}
	}
}

func (rl *RateLimiter) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := UserID(r.Header.Get("X-User-ID"))
		path := Path(r.URL.Path)
		quota, exists := rl.getQuota(userID, path)
		if !exists {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		retryAfter := rl.getRefillInterval(path)
		w.Header().Add("X-Ratelimit-Remaining", strconv.Itoa(quota))
		w.Header().Add("X-Ratelimit-Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
		if quota == 0 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		rl.decrQuota(userID, path)

		next.ServeHTTP(w, r)
	})
}
