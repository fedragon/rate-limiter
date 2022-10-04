package concurrent

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

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
