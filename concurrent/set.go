package concurrent

import "sync"

// Set represents a mathematical set that is safe for concurrent use.
// Values in the set are guaranteed to be unique.
type Set[V comparable] struct {
	content map[V]struct{}
	mux     sync.RWMutex
}

// NewSet creates and returns an instance of Set parametrized in V.
func NewSet[V comparable]() *Set[V] {
	return &Set[V]{
		content: make(map[V]struct{}),
	}
}

// Contains returns true if provided value belongs to the set, false otherwise.
func (s *Set[V]) Contains(value V) bool {
	s.mux.RLock()
	defer s.mux.RUnlock()

	_, exists := s.content[value]
	return exists
}

// Put stores value in the set.
func (s *Set[V]) Put(value V) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.content[value] = struct{}{}
}

// Size returns the current set size.
func (s *Set[V]) Size() int {
	s.mux.RLock()
	defer s.mux.RUnlock()

	return len(s.content)
}

// Iterate returns a channel prepopulated with all the values contained in this set.
func (s *Set[V]) Iterate() <-chan V {
	s.mux.RLock()
	defer s.mux.RUnlock()

	values := make(chan V, s.Size())
	defer close(values)
	for v := range s.content {
		values <- v
	}

	return values
}
