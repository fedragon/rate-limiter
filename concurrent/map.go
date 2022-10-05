package concurrent

import "sync"

// Map represents a generic map that is safe for concurrent use.
type Map[K comparable, V any] struct {
	content map[K]V
	mux     sync.RWMutex
}

type Tuple[K comparable, V any] struct {
	Key   K
	Value V
}

// NewMap creates and returns an instance of Map.
func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{
		content: make(map[K]V),
	}
}

// Get returns the value associated to key, if it exists, or its type's zero value otherwise.
// The returned boolean indicates whether a value has been found or not.
func (m *Map[K, V]) Get(key K) (V, bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	value, exists := m.content[key]
	return value, exists
}

// Put associates value to key, overwriting any existing value.
func (m *Map[K, V]) Put(key K, value V) {
	m.mux.Lock()
	defer m.mux.Unlock()

	m.content[key] = value
}

// Size returns the current map size.
func (m *Map[K, V]) Size() int {
	m.mux.RLock()
	defer m.mux.RUnlock()

	return len(m.content)
}

// Iterate returns a channel prepopulated with all the tuples contained in this map.
func (m *Map[K, V]) Iterate() <-chan Tuple[K, V] {
	m.mux.RLock()
	defer m.mux.RUnlock()

	tuples := make(chan Tuple[K, V], m.Size())
	defer close(tuples)
	for k, v := range m.content {
		tuples <- Tuple[K, V]{k, v}
	}

	return tuples
}
