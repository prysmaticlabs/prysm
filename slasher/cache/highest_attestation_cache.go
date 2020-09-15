package cache

import (
	lru "github.com/hashicorp/golang-lru"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
)

var (
	// validatorsCacheSize defines the max number of validators public keys the cache can hold.
	highestAttCacheSize = 300000
	//// Metrics for the validator cache.
	//validatorsCacheHit = promauto.NewCounter(prometheus.CounterOpts{
	//	Name: "validators_cache_hit",
	//	Help: "The total number of cache hits on the validators cache.",
	//})
	//validatorsCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
	//	Name: "validators_cache_miss",
	//	Help: "The total number of cache misses on the validators cache.",
	//})
)

// PublicKeyCache is used to store the public keys needed for signature verification.
type HighestAttestationCache struct {
	cache *lru.Cache
}

// NewPublicKeyCache initializes the cache.
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
func (c *HighestAttestationCache) Get(validatorIdx uint64) (*types.HighestAttestation, bool) {
	item, exists := c.cache.Get(validatorIdx)
	if exists && item != nil {
		//validatorsCacheHit.Inc()
		return item.(*types.HighestAttestation), true
	}

	//validatorsCacheMiss.Inc()
	return nil, false
}

// Set the response in the cache.
func (c *HighestAttestationCache) Set(validatorIdx uint64, highest *types.HighestAttestation) {
	_ = c.cache.Add(validatorIdx, highest)
}

// Delete removes a validator id from the cache and returns if it existed or not.
// Performs the onEviction function before removal.
func (c *HighestAttestationCache) Delete(validatorIdx uint64) bool {
	return c.cache.Remove(validatorIdx)
}

// Has returns true if the key exists in the cache.
func (c *HighestAttestationCache) Has(validatorIdx uint64) bool {
	return c.cache.Contains(validatorIdx)
}

// Clear removes all keys from the ValidatorCache.
func (c *HighestAttestationCache) Clear() {
	c.cache.Purge()
}

