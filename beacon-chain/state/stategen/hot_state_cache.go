package stategen

import (
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
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

// hotStateCache is used to store the processed beacon state after finalized check point..
type hotStateCache struct {
	cache *lru.Cache
	lock  sync.RWMutex
}

// newHotStateCache initializes the map and underlying cache.
func newHotStateCache() *hotStateCache {
	cache, err := lru.New(hotStateCacheSize)
	if err != nil {
		panic(err)
	}
	return &hotStateCache{
		cache: cache,
	}
}

// Get returns a cached response via input block root, if any.
// The response is copied by default.
func (c *hotStateCache) get(root [32]byte) iface.BeaconState {
	c.lock.RLock()
	defer c.lock.RUnlock()
	item, exists := c.cache.Get(root)

	if exists && item != nil {
		hotStateCacheHit.Inc()
		return item.(iface.BeaconState).Copy()
	}
	hotStateCacheMiss.Inc()
	return nil
}

// GetWithoutCopy returns a non-copied cached response via input block root.
func (c *hotStateCache) getWithoutCopy(root [32]byte) iface.BeaconState {
	c.lock.RLock()
	defer c.lock.RUnlock()
	item, exists := c.cache.Get(root)
	if exists && item != nil {
		hotStateCacheHit.Inc()
		return item.(iface.BeaconState)
	}
	hotStateCacheMiss.Inc()
	return nil
}

// put the response in the cache.
func (c *hotStateCache) put(root [32]byte, state iface.BeaconState) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.cache.Add(root, state)
}

// has returns true if the key exists in the cache.
func (c *hotStateCache) has(root [32]byte) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.cache.Contains(root)
}

// delete deletes the key exists in the cache.
func (c *hotStateCache) delete(root [32]byte) bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.cache.Remove(root)
}
