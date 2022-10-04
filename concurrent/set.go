package concurrent

import "sync"

type Set[K comparable] struct {
	content map[K]struct{}
	mux     sync.RWMutex
}

func NewSet[K comparable]() *Set[K] {
	return &Set[K]{
		content: make(map[K]struct{}),
	}
}

func (s *Set[K]) Contains(key K) bool {
	s.mux.RLock()
	defer s.mux.RUnlock()

	_, exists := s.content[key]
	return exists
}

func (s *Set[K]) Put(key K) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.content[key] = struct{}{}
}

func (s *Set[K]) ForEach(fn func(key K)) {
	for k := range s.content {
		fn(k)
	}
}
