package rate_limiter

import (
	"net/http"
)

func RateLimit(route string, maxRequests int, next http.Handler) http.Handler {
	bucket := map[string]int{
		"123": maxRequests,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == route {
			userID := r.Header.Get("X-User-ID")

			if bucket[userID] > 0 {
				bucket[userID] -= 1
				w.WriteHeader(http.StatusOK)
				return
			}

			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
