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
		Name: "committees_cache_hit",
		Help: "The total number of cache hits on the committees cache.",
	})
	committeesCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "committees_cache_miss",
		Help: "The total number of cache misses on the committees cache.",
	})
)

// CommitteesCache is used to store the cached results of committees for epoch.
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

// Get returns the cached response.
func (c *CommitteesCache) Get(ctx context.Context, epoch uint64) (*ethpb.BeaconCommittees, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	c.lock.RLock()

	item, exists := c.cache.Get(epoch)

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
