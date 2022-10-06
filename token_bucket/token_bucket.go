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
	"github.com/fedragon/rate-limiter/logging"

	"go.uber.org/zap"
)

type (
	Path   string
	UserID string

	Config struct {
		Limit  common.Rate
		Refill common.Rate
	}

	// RateLimiterBuilder builds a rate limiter.
	RateLimiterBuilder struct {
		paths *concurrent.Map[Path, Config]
		users *concurrent.Set[UserID]
	}

	// RateLimiter acts as an HTTP middleware that rate-limits traffic according to the `token bucket` algorithm, which
	// means that users can only issue requests at a given rate (configurable by endpoint) and further requests are
	// dropped until their quota is refilled.
	// Since it uses goroutines to manage the quota refilling logic, it needs to be explicitly stopped by invoking the
	// Stop() method during the HTTP server shutdown process.
	RateLimiter struct {
		paths      *concurrent.Map[Path, Config]
		userQuotas *concurrent.Map[UserID, *concurrent.Map[Path, int]]
		cancel     context.CancelFunc
		once       sync.Once
	}
)

// NewRateLimiterBuilder instantiates a rate limiter builder.
func NewRateLimiterBuilder() *RateLimiterBuilder {
	return &RateLimiterBuilder{
		paths: concurrent.NewMap[Path, Config](),
		users: concurrent.NewSet[UserID](),
	}
}

// SetLimit sets a limit on a path. The path needs to be absolute and start with a leading '/'.
func (b *RateLimiterBuilder) SetLimit(path string, cfg Config) *RateLimiterBuilder {
	b.paths.Put(Path(path), cfg)
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
	if b.paths.Size() == 0 {
		return nil, errors.New("no rate limit configured")
	}

	rl := RateLimiter{
		paths:      b.paths,
		userQuotas: concurrent.NewMap[UserID, *concurrent.Map[Path, int]](),
	}

	for u := range b.users.Iterate() {
		for t := range b.paths.Iterate() {
			path := t.Key
			limit := t.Value
			uqs, ok := rl.userQuotas.Get(u)

			if !ok {
				uqs = concurrent.NewMap[Path, int]()
				rl.userQuotas.Put(u, uqs)
			}

			uqs.Put(path, limit.Limit.Value)
		}
	}

	return &rl, nil
}

// Stop stops the rate limiter, cleaning up all used resources.
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
	limit, ok := rl.paths.Get(path)

	if !ok {
		return nil, fmt.Errorf("unknown path: %v", path)
	}

	return &limit.Refill, nil
}

func (rl *RateLimiter) refill(ctx context.Context, path Path, limit Config) {
	log := logging.Logger()

	ticker := time.NewTicker(limit.Refill.Interval)
	defer ticker.Stop()

	log.Debug("starting refiller", zap.String("path", string(path)), zap.Duration("interval", limit.Refill.Interval))

	for {
		select {
		case <-ctx.Done():
			log.Debug("stopping refiller", zap.String("path", string(path)))
			return
		case <-ticker.C:
			for t := range rl.userQuotas.Iterate() {
				quotas := t.Value
				current, _ := quotas.Get(path)
				next := current + limit.Refill.Value
				if next > limit.Limit.Value {
					next = limit.Limit.Value
				}

				quotas.Put(path, next)
			}
		default:
		}
	}
}

// Handle returns an HTTP middleware that applies preconfigured rate-limiting rules to all received requests.
func (rl *RateLimiter) Handle(next http.Handler) http.Handler {
	rl.once.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		rl.cancel = cancel

		for t := range rl.paths.Iterate() {
			go rl.refill(ctx, t.Key, t.Value)
		}
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
