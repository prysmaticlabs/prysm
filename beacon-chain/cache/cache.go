package cache

import (
	"fmt"
	"unsafe"

	"github.com/prometheus/client_golang/prometheus"

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
func newLRUCache[K comparable, V any](cacheSize int, committeeCacheHit, committeeCacheMiss prometheus.Counter) *lru.Cache[K, V] {
	cache, err := lru.New[K, V](cacheSize)
	if err != nil {
		panic(fmt.Errorf("%w: %v", ErrNilCache, err))
	}

	isCacheHitNil, isCacheMissNil := committeeCacheHit == nil, committeeCacheMiss == nil
	if isCacheHitNil || isCacheMissNil {
		panic(fmt.Errorf("%w: isCacheHitNil=<%t>, isCacheMissNil=<%t>",
			ErrNilMetrics,
			isCacheHitNil,
			isCacheMissNil,
		))
	}

	return cache
}

// get looks for a value in the cache, returns nil if not found
// and increments the prometheus counters
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
