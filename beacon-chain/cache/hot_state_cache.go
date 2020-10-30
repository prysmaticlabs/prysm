package cache

import (
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
)

var (
	// hotStateCacheSize defines the max number of hot state this can cache.
	hotStateCacheSize = 32
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

// HotStateCache is used to store the processed beacon state after finalized check point..
type HotStateCache struct {
	cache *lru.Cache
	lock  sync.RWMutex
}

// NewHotStateCache initializes the map and underlying cache.
func NewHotStateCache() *HotStateCache {
	cache, err := lru.New(hotStateCacheSize)
	if err != nil {
		panic(err)
	}
	return &HotStateCache{
		cache: cache,
	}
}

// Get returns a cached response via input block root, if any.
// The response is copied by default.
func (c *HotStateCache) Get(root [32]byte) *stateTrie.BeaconState {
	c.lock.RLock()
	defer c.lock.RUnlock()
	item, exists := c.cache.Get(root)

	if exists && item != nil {
		hotStateCacheHit.Inc()
		return item.(*stateTrie.BeaconState).Copy()
	}
	hotStateCacheMiss.Inc()
	return nil
}

// GetWithoutCopy returns a non-copied cached response via input block root.
func (c *HotStateCache) GetWithoutCopy(root [32]byte) *stateTrie.BeaconState {
	c.lock.RLock()
	defer c.lock.RUnlock()
	item, exists := c.cache.Get(root)
	if exists && item != nil {
		hotStateCacheHit.Inc()
		return item.(*stateTrie.BeaconState)
	}
	hotStateCacheMiss.Inc()
	return nil
}

// Put the response in the cache.
func (c *HotStateCache) Put(root [32]byte, state *stateTrie.BeaconState) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.cache.Add(root, state)
}

// Has returns true if the key exists in the cache.
func (c *HotStateCache) Has(root [32]byte) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.cache.Contains(root)
}

// Delete deletes the key exists in the cache.
func (c *HotStateCache) Delete(root [32]byte) bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.cache.Remove(root)
}
