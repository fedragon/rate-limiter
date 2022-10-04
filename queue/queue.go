package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/fedragon/rate-limiter/common"
)

// Queue represents a bounded queue that is refilled at regular intervals.
// When its Pop() method is invoked, it will return true if the queue contains a buffered value, false otherwise.
// It needs to be explicitly started, using its Start() method, and later stopped, using its Stop() method,  to clean-up
// all used resources.
type Queue struct {
	ctx     context.Context
	cancel  context.CancelFunc
	content chan struct{}
	rate    *common.Rate
}

// NewQueue returns a new queue.
func NewQueue(ctx context.Context, rate *common.Rate) *Queue {
	ctx, cancel := context.WithCancel(ctx)
	content := make(chan struct{}, rate.Value)
	for v := 0; v < rate.Value; v++ {
		content <- struct{}{}
	}

	return &Queue{
		ctx:     ctx,
		cancel:  cancel,
		rate:    rate,
		content: content,
	}
}

// Start starts the queue refilling logic.
func (q *Queue) Start() {
	fmt.Println("starting queue")
	t := time.NewTicker(q.rate.Interval)
	defer t.Stop()

	for {
		select {
		case <-q.ctx.Done():
			fmt.Println("stopping queue")
			close(q.content)
			return
		case <-t.C:
			for i := 0; i < q.rate.Value; i++ {
				select {
				case q.content <- struct{}{}:
					fmt.Println("refilling queue")
				default:
					// buffer is full
				}
			}
		}
	}
}

// Stop stops the queue.
func (q *Queue) Stop() {
	q.cancel()
}

// Pop returns true if there is an available value in the queue, false otherwise.
func (q *Queue) Pop() bool {
	select {
	case <-q.content:
		return true
	default:
		return false
	}
}
