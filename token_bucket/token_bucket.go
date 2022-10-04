package token_bucket

import (
	"context"
	"errors"
	"fmt"
	"github.com/fedragon/rate-limiter/concurrent"
	"net/http"
	"strconv"
	"sync"
	"time"
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
		userQuotas map[UserID]map[Path]int
		mux        sync.RWMutex
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
		userQuotas: make(map[UserID]map[Path]int),
		cancel:     cancel,
	}

	if b.users == nil {
		b.users = concurrent.NewSet[UserID]()
	}

	b.users.ForEach(func(u UserID) {
		b.pathLimits.ForEach(func(path Path, limit Limit) {
			uqs := rl.userQuotas[u]

			if uqs == nil {
				uqs = make(map[Path]int)
				rl.userQuotas[u] = uqs
			}

			uqs[path] = limit.Limit.Value
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
	rl.mux.RLock()
	defer rl.mux.RUnlock()
	if user, exists := rl.userQuotas[userID]; exists {
		if quota, exists := user[path]; exists {
			return quota, true
		}
	}

	return 0, false
}

func (rl *RateLimiter) decrQuota(userID UserID, path Path) {
	rl.mux.Lock()
	defer rl.mux.Unlock()
	user := rl.userQuotas[userID]

	if user[path] > 0 {
		user[path] -= 1
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
	rl.mux.Lock()
	defer rl.mux.Unlock()
	for _, quotas := range rl.userQuotas {
		current := quotas[path]
		next := current + value
		if next > max {
			next = max
		}

		quotas[path] = max
	}
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