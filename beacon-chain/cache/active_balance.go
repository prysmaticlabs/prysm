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
	// ErrNotActiveBalanceInfo will be returned when a cache object is not a pointer to
	// a ActiveBalanceByEpoch struct.
	ErrNotActiveBalanceInfo = errors.New("object is not a active balance obj")

	// maxActiveBalanceListSize defines the max number of active balance can cache.
	maxActiveBalanceListSize = 1000

	// Metrics.
	activeBalanceCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "active_balance_cache_miss",
		Help: "The number of active balance requests that aren't present in the cache.",
	})
	activeBalanceCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "active_balance_cache_hit",
		Help: "The number of active balance requests that are present in the cache.",
	})
)

// ActiveBalanceByEpoch defines the active validator balance per epoch.
type ActiveBalanceByEpoch struct {
	Epoch         uint64
	ActiveBalance uint64
}

// ActiveBalanceCache is a struct with 1 queue for looking up active balance by epoch.
type ActiveBalanceCache struct {
	activeBalanceCache *cache.FIFO
	lock               sync.RWMutex
}

// activeBalanceKeyFn takes the epoch as the key for the active balance of a given epoch.
func activeBalanceKeyFn(obj interface{}) (string, error) {
	tInfo, ok := obj.(*ActiveBalanceByEpoch)
	if !ok {
		return "", ErrNotActiveBalanceInfo
	}

	return strconv.Itoa(int(tInfo.Epoch)), nil
}

// NewActiveBalanceCache creates a new active balance cache for storing/accessing active validator balance.
func NewActiveBalanceCache() *ActiveBalanceCache {
	return &ActiveBalanceCache{
		activeBalanceCache: cache.NewFIFO(activeBalanceKeyFn),
	}
}

// ActiveBalanceInEpoch fetches ActiveBalanceByEpoch by epoch. Returns true with a
// reference to the ActiveBalanceInEpoch info, if exists. Otherwise returns FAR_FUTURE_EPOCH, nil.
func (c *ActiveBalanceCache) ActiveBalanceInEpoch(epoch uint64) (uint64, error) {
	if !featureconfig.FeatureConfig().EnableActiveBalanceCache {
		// Return a miss result if cache is not enabled.
		activeBalanceCacheMiss.Inc()
		return params.BeaconConfig().FarFutureEpoch, nil
	}

	c.lock.RLock()
	defer c.lock.RUnlock()
	obj, exists, err := c.activeBalanceCache.GetByKey(strconv.Itoa(int(epoch)))
	if err != nil {
		return params.BeaconConfig().FarFutureEpoch, err
	}

	if exists {
		activeBalanceCacheHit.Inc()
	} else {
		activeBalanceCacheMiss.Inc()
		return params.BeaconConfig().FarFutureEpoch, nil
	}

	tInfo, ok := obj.(*ActiveBalanceByEpoch)
	if !ok {
		return params.BeaconConfig().FarFutureEpoch, ErrNotActiveBalanceInfo
	}

	return tInfo.ActiveBalance, nil
}

// AddActiveBalance adds ActiveBalanceByEpoch object to the cache. This method also trims the least
// recently added ActiveBalanceByEpoch object if the cache size has ready the max cache size limit.
func (c *ActiveBalanceCache) AddActiveBalance(activeBalance *ActiveBalanceByEpoch) error {
	if !featureconfig.FeatureConfig().EnableActiveBalanceCache {
		return nil
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	if err := c.activeBalanceCache.AddIfNotPresent(activeBalance); err != nil {
		return err
	}

	trim(c.activeBalanceCache, maxActiveBalanceListSize)
	return nil
}
