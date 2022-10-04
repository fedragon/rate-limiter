package queue

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fedragon/rate-limiter/common"
	"github.com/stretchr/testify/assert"
)

func TestQueue_Start_RefillsAtExpectedRate(t *testing.T) {
	var total atomic.Int64
	var expected int64 = 6 // 3 prebuffered at creation time + 3 from the refiller execution
	ctx, cancel := context.WithTimeout(context.Background(), 260*time.Millisecond)
	defer cancel()
	q := NewQueue(ctx, &common.Rate{Value: 3, Interval: 250 * time.Millisecond})

	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case <-q.content:
				total.Add(1)
			default:
			}
		}
	}(ctx)
	q.Start()

	assert.Equal(t, expected, total.Load())
}

func TestQueue_Stop_ClosesUnderlyingChannel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	q := NewQueue(ctx, &common.Rate{Value: 0, Interval: time.Second})

	q.Start()
	q.Stop()

	select {
	case _, more := <-q.content:
		assert.False(t, more)
	default:
	}
}
