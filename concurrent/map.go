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

func (s *Map[K, V]) Get(key K) (V, bool) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	value, exists := s.content[key]
	return value, exists
}

func (s *Map[K, V]) Put(key K, value V) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.content[key] = value
}

func (s *Map[K, V]) ForEach(fn func(key K, value V)) {
	for k, v := range s.content {
		fn(k, v)
	}
}
