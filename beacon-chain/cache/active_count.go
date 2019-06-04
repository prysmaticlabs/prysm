package cache

import (
	"errors"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/shared/params"
	"k8s.io/client-go/tools/cache"
)

var (
	// ErrNotActiveCountInfo will be returned when a cache object is not a pointer to
	// a ActiveCountByEpoch struct.
	ErrNotActiveCountInfo = errors.New("object is not a active count obj")

	// maxActiveCountListSize defines the max number of active count can cache.
	maxActiveCountListSize = 1000

	// Metrics.
	activeCountCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "active_validator_count_cache_miss",
		Help: "The number of active validator count requests that aren't present in the cache.",
	})
	activeCountCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "active_validator_count_cache_hit",
		Help: "The number of active validator count requests that are present in the cache.",
	})
)

// ActiveCountByEpoch defines the active validator count per epoch.
type ActiveCountByEpoch struct {
	Epoch       uint64
	ActiveCount uint64
}

// ActiveCountCache is a struct with 1 queue for looking up active count by epoch.
type ActiveCountCache struct {
	activeCountCache *cache.FIFO
	lock             sync.RWMutex
}

// activeCountKeyFn takes the epoch as the key for the active count of a given epoch.
func activeCountKeyFn(obj interface{}) (string, error) {
	aInfo, ok := obj.(*ActiveCountByEpoch)
	if !ok {
		return "", ErrNotActiveCountInfo
	}

	return strconv.Itoa(int(aInfo.Epoch)), nil
}

// NewActiveCountCache creates a new active count cache for storing/accessing active validator count.
func NewActiveCountCache() *ActiveCountCache {
	return &ActiveCountCache{
		activeCountCache: cache.NewFIFO(activeCountKeyFn),
	}
}

// ActiveCountInEpoch fetches ActiveCountByEpoch by epoch. Returns true with a
// reference to the ActiveCountInEpoch info, if exists. Otherwise returns false, nil.
func (c *ActiveCountCache) ActiveCountInEpoch(epoch uint64) (uint64, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	obj, exists, err := c.activeCountCache.GetByKey(strconv.Itoa(int(epoch)))
	if err != nil {
		return params.BeaconConfig().FarFutureEpoch, err
	}

	if exists {
		activeCountCacheHit.Inc()
	} else {
		activeCountCacheMiss.Inc()
		return params.BeaconConfig().FarFutureEpoch, nil
	}

	aInfo, ok := obj.(*ActiveCountByEpoch)
	if !ok {
		return params.BeaconConfig().FarFutureEpoch, ErrNotActiveCountInfo
	}

	return aInfo.ActiveCount, nil
}

// AddActiveCount adds ActiveCountByEpoch object to the cache. This method also trims the least
// recently added ActiveCountByEpoch object if the cache size has ready the max cache size limit.
func (c *ActiveCountCache) AddActiveCount(activeCount *ActiveCountByEpoch) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if err := c.activeCountCache.AddIfNotPresent(activeCount); err != nil {
		return err
	}

	trim(c.activeCountCache, maxActiveCountListSize)
	return nil
}
