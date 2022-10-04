package leaking_bucket

import (
	"context"
	"github.com/fedragon/rate-limiter/common"
	q "github.com/fedragon/rate-limiter/queue"
	"net/http"
	"sync"
)

type RateLimiter struct {
	queue  *q.Queue
	cancel context.CancelFunc
	once   sync.Once
}

func NewRateLimiter(rate *common.Rate) *RateLimiter {
	ctx, cancel := context.WithCancel(context.Background())
	return &RateLimiter{
		queue:  q.NewQueue(ctx, rate),
		cancel: cancel,
	}
}

func (rl *RateLimiter) Stop() {
	rl.cancel()
}

func (rl *RateLimiter) Handle(next http.Handler) http.Handler {
	rl.once.Do(func() {
		go rl.queue.Start()
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.queue.Pop() {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
