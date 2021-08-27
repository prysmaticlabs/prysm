package cache

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/shared/lru"
)

var (
	// validatorsCacheSize defines the max number of validators public keys the cache can hold.
	validatorsCacheSize = 300000
	// Metrics for the validator cache.
	validatorsCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "validators_cache_hit",
		Help: "The total number of cache hits on the validators cache.",
	})
	validatorsCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "validators_cache_miss",
		Help: "The total number of cache misses on the validators cache.",
	})
)

// PublicKeyCache is used to store the public keys needed for signature verification.
type PublicKeyCache struct {
	cache lru.Cache
}

// NewPublicKeyCache initializes the cache.
func NewPublicKeyCache(size int, onEvicted func(key interface{}, value interface{})) (*PublicKeyCache, error) {
	if size != 0 {
		validatorsCacheSize = size
	}
	return &PublicKeyCache{cache: lru.NewWithEvict(validatorsCacheSize, onEvicted)}, nil
}

// Get returns an ok bool and the cached value for the requested validator id key, if any.
func (c *PublicKeyCache) Get(validatorIndex types.ValidatorIndex) ([]byte, bool) {
	item, exists := c.cache.Get(validatorIndex)
	if exists && item != nil {
		validatorsCacheHit.Inc()
		return item.([]byte), true
	}

	validatorsCacheMiss.Inc()
	return nil, false
}

// Set the response in the cache.
func (c *PublicKeyCache) Set(validatorIndex types.ValidatorIndex, publicKey []byte) {
	_ = c.cache.Add(validatorIndex, publicKey)
}

// Delete removes a validator id from the cache and returns if it existed or not.
// Performs the onEviction function before removal.
func (c *PublicKeyCache) Delete(validatorIndex types.ValidatorIndex) bool {
	return c.cache.Remove(validatorIndex)
}

// Has returns true if the key exists in the cache.
func (c *PublicKeyCache) Has(validatorIndex types.ValidatorIndex) bool {
	return c.cache.Contains(validatorIndex)
}

// Clear removes all keys from the ValidatorCache.
func (c *PublicKeyCache) Clear() {
	c.cache.Purge()
}
