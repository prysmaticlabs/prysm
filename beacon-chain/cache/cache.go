package cache

import (
	"github.com/hashicorp/golang-lru/v2"
	"reflect"
)

type Cache[K comparable, V any] interface {
	get() *lru.Cache[K, V]
	hitCache()
	missCache()
}

func Get[K comparable, V any](c Cache[K, V], key K) (V, error) {
	value, ok := c.get().Get(key)
	if !ok {
		c.missCache()
		var zero V
		return zero, ErrNotFound
	}
	c.hitCache()
	return value, nil
}

func Add[K comparable, V any](c Cache[K, V], key K, value V) {
	c.get().Add(key, value)
}

func Keys[K comparable, V any](c Cache[K, V]) []K {
	return c.get().Keys()
}

func Remove[K comparable, V any](c Cache[K, V], key K) {
	c.get().Remove(key)
}

func Purge[K comparable, V any](c Cache[K, V]) {
	c.get().Purge()
}

func isNil[T any](v T) bool {
	return reflect.ValueOf(&v).Elem().IsZero()
}
