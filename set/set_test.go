package set

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConcurrentSet_Contains(t *testing.T) {
	s := NewConcurrentSet[int]()

	s.content[1] = struct{}{}

	assert.True(t, s.Contains(1))
}

func TestConcurrentSet_Put(t *testing.T) {
	s := NewConcurrentSet[int]()

	s.Put(1)

	assert.True(t, s.Contains(1))
}

func TestConcurrentSet_ForEach(t *testing.T) {
	exists := false
	s := NewConcurrentSet[int]()
	s.Put(1)

	s.ForEach(func(k int) {
		if k == 1 {
			exists = true
		}
	})

	assert.True(t, exists)
}