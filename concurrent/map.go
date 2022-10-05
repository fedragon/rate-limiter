package concurrent

import "sync"

type Map[K comparable, V any] struct {
	content map[K]V
	mux     sync.RWMutex
}

func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{
		content: make(map[K]V),
	}
}

func (m *Map[K, V]) Get(key K) (V, bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	value, exists := m.content[key]
	return value, exists
}

func (m *Map[K, V]) Put(key K, value V) {
	m.mux.Lock()
	defer m.mux.Unlock()

	m.content[key] = value
}

func (m *Map[K, V]) ForEach(fn func(key K, value V)) {
	m.mux.Lock()
	defer m.mux.Unlock()

	for k, v := range m.content {
		fn(k, v)
	}
}

func (m *Map[K, V]) Size() int {
	m.mux.RLock()
	defer m.mux.RUnlock()

	return len(m.content)
}
