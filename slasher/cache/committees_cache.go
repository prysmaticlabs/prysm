package cache

import (
	"context"
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

var (
	// Metrics
	committeesCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "skip_slot_cache_hit",
		Help: "The total number of cache hits on the skip slot cache.",
	})
	committeesCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "skip_slot_cache_miss",
		Help: "The total number of cache misses on the skip slot cache.",
	})
)

// SkipSlotCache is used to store the cached results of processing skip slots in state.ProcessSlots.
type CommitteesCache struct {
	cache *lru.Cache
	lock  sync.RWMutex
}

// NewCommitteesCache initializes the map and underlying cache.
func NewCommitteesCache() *CommitteesCache {
	cache, err := lru.New(50)
	if err != nil {
		panic(err)
	}
	return &CommitteesCache{
		cache: cache,
	}
}

// Get waits for any in progress calculation to complete before returning a
// cached response, if any.
func (c *CommitteesCache) Get(ctx context.Context, slot uint64) (*ethpb.BeaconCommittees, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	c.lock.RLock()

	item, exists := c.cache.Get(slot)

	if exists && item != nil {
		committeesCacheHit.Inc()
		return item.(*ethpb.BeaconCommittees), nil
	}
	committeesCacheMiss.Inc()
	return nil, nil
}

// Put the response in the cache.
func (c *CommitteesCache) Put(ctx context.Context, epoch uint64, committees *ethpb.BeaconCommittees) error {
	c.cache.Add(epoch, committees)
	return nil
}
