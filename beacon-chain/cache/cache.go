package cache

import (
	"fmt"
	"reflect"

	lru "github.com/hashicorp/golang-lru/v2"
)

// lruCache is a thread-safe size LRU cache.
type lruCache[K comparable, V any] interface {
	// Clear the cache
	Clear()

	// access to cache
	get() *lru.Cache[K, V]

	// metrics
	hitCache()
	missCache()
}

// newLRUCache initialise a new thread-safe cache with Prometheus metrics
func newLRUCache[K comparable, V any](cacheSize int) (*lru.Cache[K, V], error) {
	cache, err := lru.New[K, V](cacheSize)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNilCache, err)
	}

	return cache, nil
}

// get looks for a value in the cache, returns nil if not found
// and increments the prometheus counters
func get[K comparable, V any](c lruCache[K, V], key K) (V, error) {
	value, ok := c.get().Get(key)
	if !ok {
		c.missCache()
		var noValue V
		return noValue, ErrNotFound
	}
	c.hitCache()
	return value, nil
}

// add adds a value to the cache
// it returns an error if the value is nil
func add[K comparable, V any](c lruCache[K, V], key K, value V) error {
	if isNil(value) {
		return ErrNilValueProvided
	}
	c.get().Add(key, value)
	return nil
}

// keys returns all the keys present in the cache
func keys[K comparable, V any](c lruCache[K, V]) []K {
	return c.get().Keys()
}

// purge removes all the keys and values in the cache
func purge[K comparable, V any](c lruCache[K, V]) {
	c.get().Purge()
}

// resize changes the size of the cache
func resize[K comparable, V any](c lruCache[K, V], size int) {
	c.get().Resize(size)
}

// exist looks into the cache whether the key exists or not
func exist[K comparable, V any](c lruCache[K, V], key K) bool {
	_, ok := c.get().Get(key)
	return ok
}

// isNil is a safeguard avoiding the case where a caller tries to insert a key with a value that is nil
//
// it helps with the "confusing" Go (v 1.21) case where a pointer structure is considered nil and not flagged by
// the check "V == nil"
//
// we know that an interface in Go holds two values, for example:
//
// var i interface{}       // (type=nil,value=nil)
//
// the nil instruction compares the values of the above interface, which are type and value.
//
// for i == nil to be true, i values should be type=nil and value=nil
//
// in the case of generics, when a type is declared such as any(V), "V == nil" is not always true
//
// Let's consider the following:
//
// ----------------------------------------------------------------------------------------------------------
//
// var b *beaconState               // (type=*beaconState,value=nil)
// var i interface{}                // (type=nil,value=nil)
//
// if i != nil {                    // (type=nil,value=nil) != (type=nil,value=nil)
//
//	   panic("not nil 1")           // i is in fact nil
//	}
//
// i = b                            // assign b to i
//
// if i != nil {                    // (type=*beaconState,value=nil) != (type=nil,value=nil)
//
//	   panic("not nil 2")           // panic here
//	}
//
// ----------------------------------------------------------------------------------------------------------
//
// isNil helps compare the value of its value property in order to avoid ending up inserting a nil
// value in cache.
//
// Another alternative would be to use the unsafe package:
//
// ----------------------------------------------------------------------------------------------------------
//
//	func isNil[T any](v T) bool {
//		return (*[2]uintptr)(unsafe.Pointer(&v))[1] == 0
//	}
//
// ----------------------------------------------------------------------------------------------------------
//
// It's a less expensive operation(by around 3 times less) than the reflect package but makes security scanner throw
// some concerns.
func isNil[T any](v T) bool {
	iv := reflect.ValueOf(v)
	if !iv.IsValid() {
		return true
	}
	switch iv.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Func, reflect.Interface:
		return iv.IsNil()
	default:
		return false
	}
}
