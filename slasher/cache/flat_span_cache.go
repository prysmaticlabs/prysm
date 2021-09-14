package cache

import (
	lru "github.com/hashicorp/golang-lru"
	types "github.com/prysmaticlabs/eth2-types"
	lruwrpr "github.com/prysmaticlabs/prysm/cache/lru"
	slashertypes "github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
)

// EpochFlatSpansCache is used to store the spans needed on a per-epoch basis for slashing detection.
type EpochFlatSpansCache struct {
	cache *lru.Cache
}

// NewEpochFlatSpansCache initializes the underlying cache with the given size and on evict function.
func NewEpochFlatSpansCache(size int, onEvicted func(key interface{}, value interface{})) (*EpochFlatSpansCache, error) {
	if size != 0 {
		epochSpansCacheSize = size
	}
	return &EpochFlatSpansCache{cache: lruwrpr.NewWithEvict(epochSpansCacheSize, onEvicted)}, nil
}

// Get returns an ok bool and the cached value for the requested epoch key, if any.
func (c *EpochFlatSpansCache) Get(epoch types.Epoch) (*slashertypes.EpochStore, bool) {
	item, exists := c.cache.Get(epoch)
	if exists && item != nil {
		epochSpansCacheHit.Inc()
		return item.(*slashertypes.EpochStore), true
	}

	epochSpansCacheMiss.Inc()
	return &slashertypes.EpochStore{}, false
}

// Set the response in the cache.
func (c *EpochFlatSpansCache) Set(epoch types.Epoch, epochSpans *slashertypes.EpochStore) {
	_ = c.cache.Add(epoch, epochSpans)
}

// Delete removes an epoch from the cache and returns if it existed or not.
// Performs the onEviction function before removal.
func (c *EpochFlatSpansCache) Delete(epoch types.Epoch) bool {
	return c.cache.Remove(epoch)
}

// PruneOldest removes the oldest key from the span cache, calling its OnEvict function.
func (c *EpochFlatSpansCache) PruneOldest() uint64 {
	if c.cache.Len() == epochSpansCacheSize {
		epoch, _, _ := c.cache.RemoveOldest()
		return epoch.(uint64)
	}
	return 0
}

// Has returns true if the key exists in the cache.
func (c *EpochFlatSpansCache) Has(epoch types.Epoch) bool {
	return c.cache.Contains(epoch)
}

// Purge removes all keys from the SpanCache and evicts all current data.
func (c *EpochFlatSpansCache) Purge() {
	log.Info("Saving all cached data to DB, please wait for completion.")
	c.cache.Purge()
}

// Length returns the number of cached items.
func (c *EpochFlatSpansCache) Length() int {
	return c.cache.Len()
}
