package cache

import (
	"errors"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"k8s.io/client-go/tools/cache"
)

var (
	// ErrNotStartShardInfo will be returned when a cache object is not a pointer to
	// a StartShardByEpoch struct.
	ErrNotStartShardInfo = errors.New("object is not a start shard obj")

	// maxStartShardListSize defines the max number of start shard can cache.
	maxStartShardListSize = int(params.BeaconConfig().ShardCount)

	// Metrics.
	startShardCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "start_shard_cache_miss",
		Help: "The number of start shard requests that aren't present in the cache.",
	})
	startShardCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "start_shard_cache_hit",
		Help: "The number of start shard requests that are present in the cache.",
	})
)

// StartShardByEpoch defines the start shard of the epoch.
type StartShardByEpoch struct {
	Epoch      uint64
	StartShard uint64
}

// StartShardCache is a struct with 1 queue for looking up start shard by epoch.
type StartShardCache struct {
	startShardCache *cache.FIFO
	lock            sync.RWMutex
}

// startShardKeyFn takes the epoch as the key for the start shard of a given epoch.
func startShardKeyFn(obj interface{}) (string, error) {
	sInfo, ok := obj.(*StartShardByEpoch)
	if !ok {
		return "", ErrNotStartShardInfo
	}

	return strconv.Itoa(int(sInfo.Epoch)), nil
}

// NewStartShardCache creates a new start shard cache for storing/accessing start shard.
func NewStartShardCache() *StartShardCache {
	return &StartShardCache{
		startShardCache: cache.NewFIFO(startShardKeyFn),
	}
}

// StartShardInEpoch fetches StartShardByEpoch by epoch. Returns true with a
// reference to the StartShardInEpoch info, if exists. Otherwise returns false, nil.
func (c *StartShardCache) StartShardInEpoch(epoch uint64) (uint64, error) {
	if !featureconfig.FeatureConfig().EnableStartShardCache {
		// Return a miss result if cache is not enabled.
		startShardCacheMiss.Inc()
		return params.BeaconConfig().FarFutureEpoch, nil
	}

	c.lock.RLock()
	defer c.lock.RUnlock()
	obj, exists, err := c.startShardCache.GetByKey(strconv.Itoa(int(epoch)))
	if err != nil {
		return params.BeaconConfig().FarFutureEpoch, err
	}

	if exists {
		startShardCacheHit.Inc()
	} else {
		startShardCacheMiss.Inc()
		return params.BeaconConfig().FarFutureEpoch, nil
	}

	sInfo, ok := obj.(*StartShardByEpoch)
	if !ok {
		return params.BeaconConfig().FarFutureEpoch, ErrNotStartShardInfo
	}

	return sInfo.StartShard, nil
}

// AddStartShard adds StartShardByEpoch object to the cache. This method also trims the least
// recently added StartShardByEpoch object if the cache size has ready the max cache size limit.
func (c *StartShardCache) AddStartShard(startShard *StartShardByEpoch) error {
	if !featureconfig.FeatureConfig().EnableStartShardCache {
		return nil
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	if err := c.startShardCache.AddIfNotPresent(startShard); err != nil {
		return err
	}

	trim(c.startShardCache, maxStartShardListSize)
	return nil
}
