package concurrent

import (
	"math/rand"
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
			time.Sleep(time.Duration(rand.Int31n(50)) * time.Millisecond)
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
			time.Sleep(time.Duration(rand.Int31n(50)) * time.Millisecond)
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

func TestMap_ForEach(t *testing.T) {
	var got string
	m := NewMap[int, string]()
	key := 1
	expected := "a"
	m.Put(key, expected)

	m.ForEach(func(k int, v string) {
		if k == 1 {
			got = v
		}
	})

	assert.Equal(t, expected, got)
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
