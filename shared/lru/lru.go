package lru

import (
	"fmt"
	lru "github.com/hashicorp/golang-lru"
)

type Cache interface {
	Purge()
	Add(key, value interface{}) (evicted bool)
	Get(key interface{}) (value interface{}, ok bool)
	Contains(key interface{}) bool
	Peek(key interface{}) (value interface{}, ok bool)
	ContainsOrAdd(key, value interface{}) (ok, evicted bool)
	PeekOrAdd(key, value interface{}) (previous interface{}, ok, evicted bool)
	Remove(key interface{}) (present bool)
	Resize(size int) (evicted int)
	RemoveOldest() (key, value interface{}, ok bool)
	GetOldest() (key, value interface{}, ok bool)
	Keys() []interface{}
	Len() int
}

// New creates an LRU of the given size.
func New(size int) Cache {
	cache, err := lru.New(size)
	if err != nil {
		panic(fmt.Errorf("lru new failed: %w", err))
	}
	return cache
}

// NewWithEvict constructs a fixed size cache with the given eviction
// callback.
func NewWithEvict(size int, onEvicted func(key interface{}, value interface{})) Cache {
	cache, err := lru.NewWithEvict(size, onEvicted)
	if err != nil {
		panic(fmt.Errorf("lru new with evict failed: %w", err))
	}
	return cache
}
