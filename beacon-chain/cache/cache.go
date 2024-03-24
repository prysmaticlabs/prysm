package cache

import (
	"github.com/hashicorp/golang-lru/v2"
	"unsafe"
)

type Cache[K comparable, V any] interface {
	Clear()

	get() *lru.Cache[K, V]
	hitCache()
	missCache()
}

func get[K comparable, V any](c Cache[K, V], key K) (V, error) {
	value, ok := c.get().Get(key)
	if !ok {
		c.missCache()
		var zero V
		return zero, ErrNotFound
	}
	c.hitCache()
	return value, nil
}

func add[K comparable, V any](c Cache[K, V], key K, value V) error {
	if isNil(value) {
		return ErrNilValueProvided
	}
	c.get().Add(key, value)
	return nil
}

func keys[K comparable, V any](c Cache[K, V]) []K {
	return c.get().Keys()
}

func remove[K comparable, V any](c Cache[K, V], key K) {
	c.get().Remove(key)
}

func purge[K comparable, V any](c Cache[K, V]) {
	c.get().Purge()
}

func isNil[T any](v T) bool {
	return (*[2]uintptr)(unsafe.Pointer(&v))[1] == 0
}
