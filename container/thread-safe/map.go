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

func (m *Map[K, V]) view(fn func(mp map[K]V)) {
	m.lock.RLock()
	fn(m.items)
	m.lock.RUnlock()
}

func (m *Map[K, V]) do(fn func(mp map[K]V)) {
	m.lock.Lock()
	fn(m.items)
	m.lock.Unlock()
}

// View an immutable snapshot of the map
func (m *Map[K, V]) View(fn func(mp map[K]V)) {
	m.view(fn)
}

// Do an action on the thread safe map
func (m *Map[K, V]) Do(fn func(mp map[K]V)) {
	m.do(fn)
}

// Keys returns the keys of a thread-safe map.
func (m *Map[K, V]) Keys() []K {
	r := make([]K, 0, len(m.items))
	m.view(func(mp map[K]V) {
		for k := range mp {
			key := k
			r = append(r, key)
		}
	})
	return r
}

// Len of the thread-safe map.
func (m *Map[K, V]) Len() (l int) {
	m.view(func(mp map[K]V) {
		l = len(m.items)
	})
	return
}

// Get an item from a thread-safe map.
func (m *Map[K, V]) Get(k K) (v V, ok bool) {
	m.view(func(mp map[K]V) {
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
// The keys are determined by a snapshot taken at the beginning of the range call
// If fn returns false, then the loop stops
// Only one invocation of fn will be active at one time, the iteration order is unspecified.
func (m *Map[K, V]) Range(fn func(k K, v V) bool) {
	m.view(func(mp map[K]V) {
		for k, v := range mp {
			if !fn(k, v) {
				return
			}
		}
	})
}
