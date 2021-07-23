package cache

import (
	lru "github.com/hashicorp/golang-lru"
	slashpb "github.com/prysmaticlabs/prysm/proto/prysm/v2"
)

var (
	// highestAttCacheSize defines the max number of sets of highest attestation in cache.
	highestAttCacheSize = 3000
)

// HighestAttestationCache is used to store per validator id highest attestation in cache.
type HighestAttestationCache struct {
	cache *lru.Cache
}

// NewHighestAttestationCache initializes the cache.
func NewHighestAttestationCache(size int, onEvicted func(key interface{}, value interface{})) (*HighestAttestationCache, error) {
	if size != 0 {
		highestAttCacheSize = size
	}
	cache, err := lru.NewWithEvict(highestAttCacheSize, onEvicted)
	if err != nil {
		return nil, err
	}
	return &HighestAttestationCache{cache: cache}, nil
}

// Get returns an ok bool and the cached value for the requested validator id key, if any.
func (c *HighestAttestationCache) Get(setKey uint64) (map[uint64]*slashpb.HighestAttestation, bool) {
	item, exists := c.cache.Get(setKey)
	if exists && item != nil {
		return item.(map[uint64]*slashpb.HighestAttestation), true
	}
	return nil, false
}

// Set the response in the cache.
func (c *HighestAttestationCache) Set(setKey uint64, highest *slashpb.HighestAttestation) {
	set, ok := c.Get(setKey)
	if ok {
		set[highest.ValidatorIndex] = highest
	} else {
		set = map[uint64]*slashpb.HighestAttestation{
			highest.ValidatorIndex: highest,
		}
		c.cache.Add(setKey, set)
	}
}

// Delete removes a validator id from the cache and returns if it existed or not.
// Performs the onEviction function before removal.
func (c *HighestAttestationCache) Delete(setKey uint64) bool {
	return c.cache.Remove(setKey)
}

// Has returns true if the key exists in the cache.
func (c *HighestAttestationCache) Has(setKey uint64) bool {
	return c.cache.Contains(setKey)
}

// Clear removes all keys from the ValidatorCache.
func (c *HighestAttestationCache) Clear() {
	c.cache.Purge()
}

// Purge removes all keys from cache and evicts all current data.
func (c *HighestAttestationCache) Purge() {
	log.Info("Saving all highest attestation cache data to DB, please wait for completion.")
	c.cache.Purge()
}
