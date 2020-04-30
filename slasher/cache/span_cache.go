// Package cache contains critical caches necessary for the runtime
// of the slasher service, being able to keep attestation spans by epoch
// for the validators active in the beacon chain.
package cache

import (
	lru "github.com/hashicorp/golang-lru"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
)

var (
	// epochSpansCacheSize defines the max number of epoch spans the cache can hold.
	epochSpansCacheSize = 256
	// Metrics for the span cache.
	epochSpansCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "epoch_spans_cache_hit",
		Help: "The total number of cache hits on the epoch spans cache.",
	})
	epochSpansCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "epoch_spans_cache_miss",
		Help: "The total number of cache misses on the epoch spans cache.",
	})
)

// EpochSpansCache is used to store the spans needed on a per-epoch basis for slashing detection.
type EpochSpansCache struct {
	cache *lru.Cache
}

// NewEpochSpansCache initializes the map and underlying cache.
func NewEpochSpansCache(size int, onEvicted func(key interface{}, value interface{})) (*EpochSpansCache, error) {
	if size != 0 {
		epochSpansCacheSize = size
	}
	cache, err := lru.NewWithEvict(epochSpansCacheSize, onEvicted)
	if err != nil {
		return nil, err
	}
	return &EpochSpansCache{cache: cache}, nil
}

// Get returns an ok bool and the cached value for the requested epoch key, if any.
func (c *EpochSpansCache) Get(epoch uint64) (map[uint64]types.Span, bool) {
	item, exists := c.cache.Get(epoch)
	if exists && item != nil {
		epochSpansCacheHit.Inc()
		return item.(map[uint64]types.Span), true
	}

	epochSpansCacheMiss.Inc()
	return make(map[uint64]types.Span), false
}

// Set the response in the cache.
func (c *EpochSpansCache) Set(epoch uint64, epochSpans map[uint64]types.Span) {
	_ = c.cache.Add(epoch, epochSpans)
}

// Delete removes an epoch from the cache and returns if it existed or not.
// Performs the onEviction function before removal.
func (c *EpochSpansCache) Delete(epoch uint64) bool {
	return c.cache.Remove(epoch)
}

// Has returns true if the key exists in the cache.
func (c *EpochSpansCache) Has(epoch uint64) bool {
	return c.cache.Contains(epoch)
}

// Clear removes all keys from the SpanCache.
func (c *EpochSpansCache) Clear() {
	c.cache.Purge()
}
