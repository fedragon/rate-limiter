package leaking_bucket

import (
	"context"
	"github.com/fedragon/rate-limiter/common"
	q "github.com/fedragon/rate-limiter/queue"
	"net/http"
	"strconv"
	"sync"
)

// RateLimiter acts as an HTTP middleware that rate-limits traffic according to the `leaking bucket` algorithm, which
// means that requests are processed at a fixed rate using a bounded queue which is regularly refilled: once the queue
// is full, further requests are dropped.
// Since it uses goroutines to manage the queue refilling logic, it needs to be explicitly stopped by invoking the
// Stop() method during the HTTP server shutdown process.
type RateLimiter struct {
	queue  *q.Queue
	cancel context.CancelFunc
	once   sync.Once
}

// NewRateLimiter returns a new rate limiter that is refilled at the provided rate.
func NewRateLimiter(rate *common.Rate) *RateLimiter {
	ctx, cancel := context.WithCancel(context.Background())
	return &RateLimiter{
		queue:  q.NewQueue(ctx, rate),
		cancel: cancel,
	}
}

// Stop stops the rate limiter's goroutines, cleaning up all used resources.
func (rl *RateLimiter) Stop() {
	rl.cancel()
}

func (rl *RateLimiter) Handle(next http.Handler) http.Handler {
	rl.once.Do(func() {
		go rl.queue.Start()
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.queue.Pop() {
			rate := rl.queue.Rate()

			w.WriteHeader(http.StatusTooManyRequests)
			w.Header().Add("X-Ratelimit-Retry-Limit", strconv.Itoa(rate.Value))
			w.Header().Add("X-Ratelimit-Retry-After", strconv.Itoa(int(rate.Interval.Seconds())))
			return
		}

		next.ServeHTTP(w, r)
	})
}
