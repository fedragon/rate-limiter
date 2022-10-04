package set

import "sync"

type ConcurrentSet[K comparable] struct {
	content map[K]struct{}
	mux     sync.RWMutex
}

func NewConcurrentSet[K comparable]() *ConcurrentSet[K] {
	return &ConcurrentSet[K]{
		content: make(map[K]struct{}),
	}
}

func (s *ConcurrentSet[K]) Contains(key K) bool {
	s.mux.RLock()
	defer s.mux.RUnlock()

	_, exists := s.content[key]
	return exists
}

func (s *ConcurrentSet[K]) Put(key K) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.content[key] = struct{}{}
}

func (s *ConcurrentSet[K]) ForEach(fn func(key K)) {
	for k := range s.content {
		fn(k)
	}
}
