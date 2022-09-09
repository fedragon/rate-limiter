package rate_limiter

import (
	"context"
	"fmt"
	"net/http"
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
		pathLimits map[Path]Limit
		users      map[UserID]struct{}
	}

	RateLimiter struct {
		pathLimits map[Path]Limit
		userQuotas map[UserID]map[Path]int
		cancel     context.CancelFunc
	}
)

func (b *RateLimiterBuilder) SetLimit(path string, limit Limit) *RateLimiterBuilder {
	if b.pathLimits == nil {
		b.pathLimits = make(map[Path]Limit)
	}
	b.pathLimits[Path(path)] = limit

	return b
}

func (b *RateLimiterBuilder) RegisterUser(ID string) *RateLimiterBuilder {
	if b.users == nil {
		b.users = make(map[UserID]struct{})
	}

	b.users[UserID(ID)] = struct{}{}

	return b
}

func (b *RateLimiterBuilder) Build() *RateLimiter {
	ctx, cancel := context.WithCancel(context.Background())
	rl := RateLimiter{
		pathLimits: b.pathLimits,
		userQuotas: make(map[UserID]map[Path]int),
		cancel:     cancel,
	}

	for u, _ := range b.users {
		for p, v := range b.pathLimits {
			uqs := rl.userQuotas[u]

			if uqs == nil {
				uqs = make(map[Path]int)
				rl.userQuotas[u] = uqs
			}

			uqs[p] = v.Limit.Value
		}
	}

	for p, v := range b.pathLimits {
		go rl.refill(ctx, p, v)
	}

	return &rl
}

func (rl *RateLimiter) Stop() {
	rl.cancel()
}

func (rl *RateLimiter) GetQuota(userID UserID, path Path) (int, bool) {
	if user, exists := rl.userQuotas[userID]; exists {
		if quota, exists := user[path]; exists {
			return quota, true
		}
	}

	return 0, false
}

func (rl *RateLimiter) DecrQuota(userID UserID, path Path) {
	user := rl.userQuotas[userID]

	if user[path] > 0 {
		user[path] -= 1
	}
}

func (rl *RateLimiter) refillQuotas(path Path, value int, max int) {
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
		quota, exists := rl.GetQuota(userID, path)
		if !exists {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if quota == 0 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		rl.DecrQuota(userID, path)

		next.ServeHTTP(w, r)
	})
}
