package cache

import (
	"fmt"
	"unsafe"

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

func newLRUCache[K comparable, V any]() *lru.Cache[K, V] {
	cache, err := lru.New[K, V](maxCommitteesCacheSize)
	if err != nil {
		panic(fmt.Errorf("%w: %v", ErrNilCache, err))
	}

	isCacheMissNil, isCacheHitNil := committeeCacheMiss == nil, committeeCacheMiss == nil
	if isCacheMissNil || isCacheHitNil {
		panic(fmt.Errorf("%w: isCacheMissNil=<%t>, isCacheHitNil=<%t>",
			ErrNilMetrics,
			isCacheHitNil,
			isCacheHitNil,
		))
	}

	return cache
}

func get[K comparable, V any](c lruCache[K, V], key K) (V, error) {
	value, ok := c.get().Get(key)
	if !ok {
		c.missCache()
		var zero V
		return zero, ErrNotFound
	}
	c.hitCache()
	return value, nil
}

// method helpers
func add[K comparable, V any](c lruCache[K, V], key K, value V) error {
	if isNil(value) {
		return ErrNilValueProvided
	}
	c.get().Add(key, value)
	return nil
}

func keys[K comparable, V any](c lruCache[K, V]) []K {
	return c.get().Keys()
}

func purge[K comparable, V any](c lruCache[K, V]) {
	c.get().Purge()
}

func resize[K comparable, V any](c lruCache[K, V], size int) {
	c.get().Resize(size)
}

func exist[K comparable, V any](c lruCache[K, V], key K) bool {
	_, ok := c.get().Get(key)
	return ok
}

// comparison helpers
//
// isNil is a safeguard when trying to insert key -> value, where value is nil
//
// it helps with the confusing Go case where a pointer structure is equal to nil and not flagged by "V == nil"
// we know that an interface in Go holds two values:
// var i interface{}       // (type=nil,value=nil)
// that the nil operator compares the values of type and value to both be nil such as:
// for i == nil to be true, i should have type==nil and value==nil
//
// Unfortunately, in the case of generics when a type any(V), "V == nil" is not always true
// Picture the following:
// var b *beaconState              // (type=*beaconState,value=nil)
// var i interface{}       		   // (type=nil,value=nil)
//
// if i != nil {                   // (type=nil,value=nil) != (type=nil,value=nil)
//
//		   panic("not nil 1")      // i is in fact nil
//	}
//
// i = p                   		   // assign b to i
//
// if i != nil {           		   // (type=*beaconState,value=nil) != (type=nil,value=nil)
//
//	      panic("not nil 2")       // panic here
//	}
//
// isNil helps comparing the second 'value' that an interface holds in order to avoid ending up inserting a nil
// value in cache even though multiple '== nil' have been done before hand.
// you can see this as a safeguard.
//
// Another alternative would be to use the reflect package
// reflect.ValueOf(v).IsNil()
// It's way more expensive though (around 3 times more ns/op)
func isNil[T any](v T) bool {
	return (*[2]uintptr)(unsafe.Pointer(&v))[1] == 0
}
