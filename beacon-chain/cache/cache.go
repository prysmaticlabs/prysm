package cache

import (
	"unsafe"

	lru "github.com/hashicorp/golang-lru/v2"
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

func purge[K comparable, V any](c Cache[K, V]) {
	c.get().Purge()
}

func resize[K comparable, V any](c Cache[K, V], size int) {
	c.get().Resize(size)
}

func exist[K comparable, V any](c Cache[K, V], key K) bool {
	_, ok := c.get().Get(key)
	return ok
}

// helpers
func isNil[T any](v T) bool {
	return (*[2]uintptr)(unsafe.Pointer(&v))[1] == 0
}
