package rate_limiter

import (
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

	RateLimiter struct {
		pathLimits map[Path]Limit
		userQuotas map[UserID]map[Path]int
	}
)

func (rl *RateLimiter) SetLimit(path string, limit Limit) *RateLimiter {
	if rl.pathLimits == nil {
		rl.pathLimits = make(map[Path]Limit)
	}
	rl.pathLimits[Path(path)] = limit

	return rl
}

func (rl *RateLimiter) RegisterUser(ID string) *RateLimiter {
	if rl.userQuotas == nil {
		rl.userQuotas = make(map[UserID]map[Path]int)
	}

	for path, limit := range rl.pathLimits {
		quotas := rl.userQuotas[UserID(ID)]

		if quotas == nil {
			quotas = make(map[Path]int)
			rl.userQuotas[UserID(ID)] = quotas
		}

		quotas[path] = limit.Limit.Value
	}

	return rl
}

func (rl *RateLimiter) GetQuota(userID string, path string) (int, bool) {
	if user, exists := rl.userQuotas[UserID(userID)]; exists {
		if quota, exists := user[Path(path)]; exists {
			return quota, true
		}
	}

	return 0, false
}

func (rl *RateLimiter) DecrQuota(userID string, path string) {
	user := rl.userQuotas[UserID(userID)]

	if user[Path(path)] > 0 {
		user[Path(path)] -= 1
	}
}

func (rl *RateLimiter) RefillQuotas() {
	for _, quotas := range rl.userQuotas {
		for path, current := range quotas {
			limit := rl.pathLimits[path]
			if current+limit.Refill.Value > limit.Limit.Value {
				quotas[path] = limit.Limit.Value
			} else {
				quotas[path] = current + limit.Refill.Value
			}
		}
	}
}

func (rl *RateLimiter) Refill(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.RefillQuotas()
		default:
		}
	}
}

func (rl *RateLimiter) RateLimit(refillRate time.Duration, next http.Handler) http.Handler {
	go rl.Refill(refillRate)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-ID")
		quota, exists := rl.GetQuota(userID, r.URL.Path)
		if !exists {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if quota == 0 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		rl.DecrQuota(userID, r.URL.Path)

		next.ServeHTTP(w, r)
	})
}
