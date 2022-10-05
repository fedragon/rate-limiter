package concurrent

import (
	"context"
	"github.com/fedragon/rate-limiter/test"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type msg struct {
	value string
	ts    time.Time
}

func TestMap_Get(t *testing.T) {
	m := NewMap[int, string]()
	key := 1
	expected := "a"
	m.content[key] = expected

	got, ok := m.Get(key)

	assert.True(t, ok)
	assert.Equal(t, expected, got)
}

func TestMap_Put(t *testing.T) {
	m := NewMap[int, string]()
	key := 1
	expected := "a"

	m.Put(key, expected)

	got, ok := m.Get(key)
	assert.True(t, ok)
	assert.Equal(t, expected, got)
}

func TestMap_ConcurrentPut(t *testing.T) {
	m := NewMap[int, string]()
	key := 1
	outputs := make(chan msg, 30)
	producer := func(wg *sync.WaitGroup, m *Map[int, string], out chan<- msg, value string) {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			time.Sleep(test.RandomDuration())
			m.Put(key, value)
			out <- msg{value, time.Now()}
		}
	}

	var wg sync.WaitGroup
	wg.Add(3)
	go producer(&wg, m, outputs, "A")
	go producer(&wg, m, outputs, "M")
	go producer(&wg, m, outputs, "X")
	wg.Wait()
	close(outputs)

	got, ok := m.Get(key)
	assert.True(t, ok)
	assert.Equal(t, got, mostRecent(outputs))
}

func TestMap_ConcurrentGet(t *testing.T) {
	m := NewMap[int, string]()
	key := 1
	expected := "A"
	outputs := make(chan msg, 30)
	consumer := func(wg *sync.WaitGroup, m *Map[int, string], out chan<- msg) {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			time.Sleep(test.RandomDuration())
			value, _ := m.Get(key)
			out <- msg{value, time.Now()}
		}
	}

	m.Put(key, expected)

	var wg sync.WaitGroup
	wg.Add(3)
	go consumer(&wg, m, outputs)
	go consumer(&wg, m, outputs)
	go consumer(&wg, m, outputs)
	wg.Wait()
	close(outputs)

	assert.Equal(t, expected, mostRecent(outputs))
}

func TestMap_Size(t *testing.T) {
	m := NewMap[int, string]()

	assert.Zero(t, m.Size())

	m.Put(1, "a")
	m.Put(2, "b")

	assert.Equal(t, 2, m.Size())
}

func TestMap_ConcurrentSize(t *testing.T) {
	m := NewMap[int, string]()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	client := func(ctx context.Context, m *Map[int, string]) {
		for {
			time.Sleep(test.RandomDuration())
			m.Size()
		}
	}

	go client(ctx, m)
	go client(ctx, m)
	go client(ctx, m)
}

func TestMap_ConcurrentIterate(t *testing.T) {
	m := NewMap[int, string]()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	client := func(ctx context.Context, m *Map[int, string]) {
		for {
			time.Sleep(test.RandomDuration())
			m.Iterate()
		}
	}

	go client(ctx, m)
	go client(ctx, m)
	go client(ctx, m)
}

func mostRecent(msgs <-chan msg) string {
	var value string
	var ts time.Time
	for v := range msgs {
		if v.ts.After(ts) {
			value = v.value
		}
	}

	return value
}
