// Package threadsafe contains generic containers that are
// protected either by Mutexes or atomics underneath the hood.
package threadsafe

import "sync"

// Map implements a simple thread-safe map protected by a mutex.
type Map[K comparable, V any] struct {
	items map[K]V
	lock  sync.RWMutex
}

// NewThreadSafeMap returns a thread-safe map instance from a normal map.
func NewThreadSafeMap[K comparable, V any](m map[K]V) *Map[K, V] {
	return &Map[K, V]{
		items: m,
	}
}

// Keys returns the keys of a thread-safe map.
func (m *Map[K, V]) Keys() []K {
	m.lock.RLock()
	defer m.lock.RUnlock()
	r := make([]K, 0, len(m.items))
	for k := range m.items {
		key := k
		r = append(r, key)
	}
	return r
}

// Len of the thread-safe map.
func (m *Map[K, V]) Len() int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return len(m.items)
}

// Get an item from a thread-safe map.
func (m *Map[K, V]) Get(k K) (V, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	v, ok := m.items[k]
	return v, ok
}

// Put an item into a thread-safe map.
func (m *Map[K, V]) Put(k K, v V) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.items[k] = v
}

// Delete an item from a thread-safe map.
func (m *Map[K, V]) Delete(k K) {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.items, k)
}
