package stategen

import (
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
)

var (
	// epochStateCacheSize defines the max number of epoch state this can cache.
	// 8 means it can handle no finality for up to an hour before starting to replay
	// from the last finalized check point.
	epochStateCacheSize = 8
	// Metrics
	epochStateCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "epoch_state_cache_hit",
		Help: "The total number of cache hits on the epoch state cache.",
	})
	epochStateCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "epoch_state_cache_miss",
		Help: "The total number of cache misses on the epoch state cache.",
	})
)

// epochStateCache is used to store the processed epoch boundary state and finalized state post finalization.
type epochStateCache struct {
	fRoot [32]byte
	fState *stateTrie.BeaconState
	cache *lru.Cache
	lock  sync.RWMutex
}

// newEpochStateCache initializes the map and underlying cache.
func newEpochStateCache(fRoot [32]byte, fState *stateTrie.BeaconState) *epochStateCache {
	cache, err := lru.New(epochStateCacheSize)
	if err != nil {
		panic(err)
	}
	return &epochStateCache{
		fRoot: fRoot,
		fState: fState,
		cache: cache,
	}
}

// Get returns a cached finalized or epoch boundary state via input block root.
// The response is copied by default.
func (e *epochStateCache) Get(root [32]byte) *stateTrie.BeaconState {
	e.lock.RLock()
	defer e.lock.RUnlock()

	// First checks if the root is a finalized state.
	if root == e.fRoot {
		epochStateCacheHit.Inc()
		return e.fState.Copy()
	}

	item, exists := e.cache.Get(root)
	if exists && item != nil {
		epochStateCacheHit.Inc()
		return item.(*stateTrie.BeaconState).Copy()
	}

	epochStateCacheMiss.Inc()
	return nil
}

// PutEpochBoundaryState puts the epoch boundary state in the cache.
func (e *epochStateCache) PutEpochBoundaryState(root [32]byte, state *stateTrie.BeaconState) {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.cache.Add(root, state)
}

// PutFinalizedState puts the finalized state in the cache.
func (e *epochStateCache) PutFinalizedState(root [32]byte, state *stateTrie.BeaconState) {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.fRoot = root
	e.fState = state

	// Remove finalized state from cache since it's saved outside of the cache.
	e.cache.Remove(root)
}

// Has returns true if the key exists in the epoch boundary cache.
func (e *epochStateCache) Has(root [32]byte) bool {
	e.lock.RLock()
	defer e.lock.RUnlock()

	return root == e.fRoot  || e.cache.Contains(root)
}
