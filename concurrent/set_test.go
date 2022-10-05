package concurrent

import (
	"math/rand"
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

func TestSet_ForEach(t *testing.T) {
	exists := false
	s := NewSet[int]()
	s.Put(1)

	s.ForEach(func(k int) {
		if k == 1 {
			exists = true
		}
	})

	assert.True(t, exists)
}

func TestSet_ConcurrentPut(t *testing.T) {
	s := NewSet[int]()
	key := 1
	producer := func(wg *sync.WaitGroup, s *Set[int]) {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			time.Sleep(time.Duration(rand.Int31n(50)) * time.Millisecond)
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
			time.Sleep(time.Duration(rand.Int31n(50)) * time.Millisecond)
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
