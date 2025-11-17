package safe

import "sync"

// Slice is a thread-safe slice implementation
type Slice struct {
	mu   sync.RWMutex
data []interface{}
}

// NewSlice creates a new thread-safe slice
func NewSlice() *Slice {
	return &Slice{
		data: make([]interface{}, 0),
	}
}

// Append adds an element to the end of the slice
func (s *Slice) Append(value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = append(s.data, value)
}

// Get returns the element at the given index
func (s *Slice) Get(index int) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if index < 0 || index >= len(s.data) {
		return nil, false
	}
	return s.data[index], true
}

// Set sets the element at the given index
func (s *Slice) Set(index int, value interface{}) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.data) {
		return false
	}
	s.data[index] = value
	return true
}

// Len returns the number of elements in the slice
func (s *Slice) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

// Range calls f sequentially for each element in the slice.
// If f returns false, range stops the iteration.
func (s *Slice) Range(f func(index int, value interface{}) bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i, v := range s.data {
		if !f(i, v) {
			break
		}
	}
}

// Clear removes all elements from the slice
func (s *Slice) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make([]interface{}, 0)
}

// Remove removes the element at the given index
func (s *Slice) Remove(index int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.data) {
		return false
	}
	s.data = append(s.data[:index], s.data[index+1:]...)
	return true
}

// ToSlice returns a copy of the underlying slice
// Note: This is a snapshot and may be out of date immediately after returning
func (s *Slice) ToSlice() []interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]interface{}, len(s.data))
	copy(result, s.data)
	return result
}
