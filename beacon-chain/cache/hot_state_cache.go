package cache

import (
	lru "github.com/hashicorp/golang-lru"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
)

var (
	// Metrics
	hotStateCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "hot_state_cache_hit",
		Help: "The total number of cache hits on the hot state cache.",
	})
	hotStateCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "hot_state_cache_miss",
		Help: "The total number of cache misses on the hot state cache.",
	})
)

// HotStateCache is used to store the cached results of processed state after finalized check point..
type HotStateCache struct {
	cache *lru.Cache
}

// NewHotStateCache initializes the map and underlying cache.
func NewHotStateCache() *HotStateCache {
	cache, err := lru.New(8)
	if err != nil {
		panic(err)
	}
	return &HotStateCache{
		cache: cache,
	}
}

// Get waits for any in progress calculation to complete before returning a
// cached response, if any.
func (c *HotStateCache) Get(root [32]byte) *stateTrie.BeaconState {
	item, exists := c.cache.Get(root)

	if exists && item != nil {
		hotStateCacheHit.Inc()
		return item.(*stateTrie.BeaconState).Copy()
	}
	hotStateCacheMiss.Inc()
	return nil
}

// Put the response in the cache.
func (c *HotStateCache) Put(root [32]byte, state *stateTrie.BeaconState) {
	c.cache.Add(root, state)
}

// Has returns true if the key exists in the cache.
func (c *HotStateCache) Has(root [32]byte) bool {
	return c.cache.Contains(root)
}
