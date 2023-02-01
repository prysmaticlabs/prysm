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

func (m *Map[K, V]) do(fn func(mp map[K]V)) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	fn(m.items)
}

func (m *Map[K, V]) Do(fn func(mp map[K]V)) {
	m.do(fn)
}

// Keys returns the keys of a thread-safe map.
func (m *Map[K, V]) Keys() []K {
	r := make([]K, 0, len(m.items))
	m.do(func(mp map[K]V) {
		for k := range mp {
			key := k
			r = append(r, key)
		}
	})
	return r
}

// Len of the thread-safe map.
func (m *Map[K, V]) Len() (l int) {
	m.do(func(mp map[K]V) {
		l = len(m.items)
	})
	return
}

// Get an item from a thread-safe map.
func (m *Map[K, V]) Get(k K) (v V, ok bool) {
	m.do(func(mp map[K]V) {
		v, ok = mp[k]
	})
	return v, ok
}

// Put an item into a thread-safe map.
func (m *Map[K, V]) Put(k K, v V) {
	m.do(func(mp map[K]V) {
		mp[k] = v
	})
}

// Delete an item from a thread-safe map.
func (m *Map[K, V]) Delete(k K) {
	m.do(func(mp map[K]V) {
		delete(m.items, k)
	})
}

// Range runs the function fn(k K, v V) bool for each key value pair
// If fn returns false, then the loop stops
// Only one invocation of fn will be active at one time
// The order is unspecified
func (m *Map[K, V]) Range(fn func(k K, v V)) {
	m.do(func(mp map[K]V) {
		for k, v := range mp {
			fn(k, v)
		}
	})
}
