package token_bucket

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/fedragon/rate-limiter/common"
	"github.com/fedragon/rate-limiter/concurrent"
)

type (
	Path   string
	UserID string

	Limit struct {
		Limit  common.Rate
		Refill common.Rate
	}

	// RateLimiterBuilder builds a rate limiter.
	RateLimiterBuilder struct {
		pathLimits *concurrent.Map[Path, Limit]
		users      *concurrent.Set[UserID]
	}

	// RateLimiter acts as an HTTP middleware that rate-limits traffic according to the `token bucket` algorithm, which
	// means that users can only issue requests at a given rate (configurable by endpoint) and further requests are
	// dropped until their quota is refilled.
	// Since it uses goroutines to manage the quota refilling logic, it needs to be explicitly stopped by invoking the
	// Stop() method during the HTTP server shutdown process.
	RateLimiter struct {
		pathLimits *concurrent.Map[Path, Limit]
		userQuotas *concurrent.Map[UserID, *concurrent.Map[Path, int]]
		cancel     context.CancelFunc
		once       sync.Once
	}
)

// NewRateLimiterBuilder instantiates a rate limiter builder.
func NewRateLimiterBuilder() *RateLimiterBuilder {
	return &RateLimiterBuilder{
		pathLimits: concurrent.NewMap[Path, Limit](),
		users:      concurrent.NewSet[UserID](),
	}
}

// SetLimit sets a limit on a path. The path needs to be absolute and should start with a leading '/'.
func (b *RateLimiterBuilder) SetLimit(path string, limit Limit) *RateLimiterBuilder {
	b.pathLimits.Put(Path(path), limit)
	return b
}

// RegisterUser registers a user.
func (b *RateLimiterBuilder) RegisterUser(ID string) *RateLimiterBuilder {
	b.users.Put(UserID(ID))
	return b
}

// Build builds a rate limiter, setting quotas for each configured user and path.
// It returns an error if no limits have been configured.
func (b *RateLimiterBuilder) Build() (*RateLimiter, error) {
	if b.pathLimits.Size() == 0 {
		return nil, errors.New("no rate limits configured")
	}

	rl := RateLimiter{
		pathLimits: b.pathLimits,
		userQuotas: concurrent.NewMap[UserID, *concurrent.Map[Path, int]](),
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

	return &rl, nil
}

// Stop stops the rate limiter's goroutines, cleaning up all used resources.
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

func (rl *RateLimiter) getRefillRate(path Path) (*common.Rate, error) {
	limit, ok := rl.pathLimits.Get(path)

	if !ok {
		return nil, fmt.Errorf("unknown path: %v", path)
	}

	return &limit.Refill, nil
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

// Handle returns an HTTP middleware that applies preconfigured rate-limiting rules to all received requests.
func (rl *RateLimiter) Handle(next http.Handler) http.Handler {
	rl.once.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		rl.cancel = cancel

		rl.pathLimits.ForEach(func(p Path, v Limit) {
			go rl.refill(ctx, p, v)
		})
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := UserID(r.Header.Get("X-User-ID"))
		path := Path(r.URL.Path)
		quota, exists := rl.getQuota(userID, path)
		if !exists {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if quota == 0 {
			rate, err := rl.getRefillRate(path)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

			w.WriteHeader(http.StatusTooManyRequests)
			w.Header().Add("X-Ratelimit-Limit", strconv.Itoa(rate.Value))
			w.Header().Add("X-Ratelimit-Retry-After", strconv.Itoa(int(rate.Interval.Seconds())))
			return
		}

		rl.decrQuota(userID, path)

		next.ServeHTTP(w, r)
	})
}
