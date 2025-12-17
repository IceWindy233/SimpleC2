package safe

import "sync"

// Map is a thread-safe map implementation using RWMutex
// It supports any comparable type as key

type Map struct {
	mu sync.RWMutex
	m  map[interface{}]interface{}
}

// NewMap creates a new thread-safe map
func NewMap() *Map {
	return &Map{
		m: make(map[interface{}]interface{}),
	}
}

// Store sets the value for a key
func (m *Map) Store(key, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.m[key] = value
}

// Load returns the value stored in the map for a key, or nil if no value is present.
// The ok result indicates whether value was found in the map
func (m *Map) Load(key interface{}) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.m[key]
	return val, ok
}

// LoadOrStore returns the existing value for the key if present.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false if stored.
func (m *Map) LoadOrStore(key, value interface{}) (actual interface{}, loaded bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	actual, loaded = m.m[key]
	if loaded {
		return actual, true
	}
	m.m[key] = value
	return value, false
}

// Delete deletes the value for a key
func (m *Map) Delete(key interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.m, key)
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
func (m *Map) Range(f func(key, value interface{}) bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for k, v := range m.m {
		if !f(k, v) {
			break
		}
	}
}

// Len returns the number of items in the map
func (m *Map) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.m)
}

// Clear removes all entries from the map
func (m *Map) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.m = make(map[interface{}]interface{})
}
