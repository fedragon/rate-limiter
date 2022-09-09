package rate_limiter

import (
	"net/http"
)

type RateLimiter struct {
	pathLimits map[string]int
	userQuotas map[string]map[string]int
}

func (rl *RateLimiter) SetLimit(path string, limit int) *RateLimiter {
	if rl.pathLimits == nil {
		rl.pathLimits = make(map[string]int)
	}
	rl.pathLimits[path] = limit

	return rl
}

func (rl *RateLimiter) RegisterUser(ID string) *RateLimiter {
	if rl.userQuotas == nil {
		rl.userQuotas = make(map[string]map[string]int)
	}

	for path, limit := range rl.pathLimits {
		quotas := rl.userQuotas[ID]

		if quotas == nil {
			quotas = make(map[string]int)
			rl.userQuotas[ID] = quotas
		}

		quotas[path] = limit
	}

	return rl
}

func (rl *RateLimiter) GetQuota(userID string, path string) (int, bool) {
	if user, exists := rl.userQuotas[userID]; exists {
		if quota, exists := user[path]; exists {
			return quota, true
		}
	}

	return 0, false
}

func (rl *RateLimiter) DecrQuota(userID string, path string) {
	user := rl.userQuotas[userID]

	if user[path] > 0 {
		user[path] -= 1
	}
}

func (rl *RateLimiter) RateLimit(next http.Handler) http.Handler {
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
