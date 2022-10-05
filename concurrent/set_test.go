package concurrent

import (
	"context"
	"github.com/fedragon/rate-limiter/test"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSet_Contains(t *testing.T) {
	s := NewSet[int]()

	s.content[1] = struct{}{}

	assert.True(t, s.Contains(1))
}

func TestSet_Put(t *testing.T) {
	s := NewSet[int]()

	s.Put(1)

	assert.True(t, s.Contains(1))
}

func TestSet_ConcurrentPut(t *testing.T) {
	s := NewSet[int]()
	key := 1
	producer := func(wg *sync.WaitGroup, s *Set[int]) {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			time.Sleep(test.RandomDuration())
			s.Put(key)
		}
	}

	var wg sync.WaitGroup
	wg.Add(3)
	go producer(&wg, s)
	go producer(&wg, s)
	go producer(&wg, s)
	wg.Wait()

	assert.True(t, s.Contains(key))
}

func TestSet_ConcurrentContains(t *testing.T) {
	s := NewSet[int]()
	key := 1
	consumer := func(wg *sync.WaitGroup, s *Set[int]) {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			time.Sleep(test.RandomDuration())
			s.Contains(key)
		}
	}

	s.Put(key)

	var wg sync.WaitGroup
	wg.Add(3)
	go consumer(&wg, s)
	go consumer(&wg, s)
	go consumer(&wg, s)
	wg.Wait()
}

func TestSet_Size(t *testing.T) {
	m := NewSet[int]()

	assert.Zero(t, m.Size())

	m.Put(1)
	m.Put(2)

	assert.Equal(t, 2, m.Size())
}

func TestSet_ConcurrentSize(t *testing.T) {
	m := NewSet[int]()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	client := func(ctx context.Context, m *Set[int]) {
		for {
			time.Sleep(test.RandomDuration())
			m.Size()
		}
	}

	go client(ctx, m)
	go client(ctx, m)
	go client(ctx, m)
}

func TestSet_ConcurrentIterate(t *testing.T) {
	m := NewSet[int]()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	client := func(ctx context.Context, m *Set[int]) {
		for {
			time.Sleep(test.RandomDuration())
			m.Iterate()
		}
	}

	go client(ctx, m)
	go client(ctx, m)
	go client(ctx, m)
}
