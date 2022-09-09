package rate_limiter

import (
	"fmt"
	"net/http"
	"time"
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

func (rl *RateLimiter) ResetQuotas() {
	fmt.Println("Resetting quotas")
	for _, quotas := range rl.userQuotas {
		for path, _ := range quotas {
			quotas[path] = rl.pathLimits[path]
		}
	}
}

func (rl *RateLimiter) Refill(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.ResetQuotas()
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
