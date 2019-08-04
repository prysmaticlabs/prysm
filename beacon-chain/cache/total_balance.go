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
	// ErrNotTotalBalanceInfo will be returned when a cache object is not a pointer to
	// a TotalBalanceByEpoch struct.
	ErrNotTotalBalanceInfo = errors.New("object is not a total balance obj")

	// maxTotalBalanceListSize defines the max number of total balance can cache.
	maxTotalBalanceListSize = 1000

	// Metrics.
	totalBalanceCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "total_balance_cache_miss",
		Help: "The number of total balance requests that aren't present in the cache.",
	})
	totalBalanceCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "total_balance_cache_hit",
		Help: "The number of total balance requests that are present in the cache.",
	})
)

// TotalBalanceByEpoch defines the total validator balance per epoch.
type TotalBalanceByEpoch struct {
	Epoch        uint64
	TotalBalance uint64
}

// TotalBalanceCache is a struct with 1 queue for looking up total balance by epoch.
type TotalBalanceCache struct {
	totalBalanceCache *cache.FIFO
	lock              sync.RWMutex
}

// totalBalanceKeyFn takes the epoch as the key for the total balance of a given epoch.
func totalBalanceKeyFn(obj interface{}) (string, error) {
	tInfo, ok := obj.(*TotalBalanceByEpoch)
	if !ok {
		return "", ErrNotTotalBalanceInfo
	}

	return strconv.Itoa(int(tInfo.Epoch)), nil
}

// NewTotalBalanceCache creates a new total balance cache for storing/accessing total validator balance.
func NewTotalBalanceCache() *TotalBalanceCache {
	return &TotalBalanceCache{
		totalBalanceCache: cache.NewFIFO(totalBalanceKeyFn),
	}
}

// TotalBalanceInEpoch fetches TotalBalanceByEpoch by epoch. Returns true with a
// reference to the TotalBalanceInEpoch info, if exists. Otherwise returns false, nil.
func (c *TotalBalanceCache) TotalBalanceInEpoch(epoch uint64) (uint64, error) {
	if !featureconfig.FeatureConfig().EnableTotalBalanceCache {
		// Return a miss result if cache is not enabled.
		totalBalanceCacheMiss.Inc()
		return params.BeaconConfig().FarFutureEpoch, nil
	}

	c.lock.RLock()
	defer c.lock.RUnlock()
	obj, exists, err := c.totalBalanceCache.GetByKey(strconv.Itoa(int(epoch)))
	if err != nil {
		return params.BeaconConfig().FarFutureEpoch, err
	}

	if exists {
		totalBalanceCacheHit.Inc()
	} else {
		totalBalanceCacheMiss.Inc()
		return params.BeaconConfig().FarFutureEpoch, nil
	}

	tInfo, ok := obj.(*TotalBalanceByEpoch)
	if !ok {
		return params.BeaconConfig().FarFutureEpoch, ErrNotTotalBalanceInfo
	}

	return tInfo.TotalBalance, nil
}

// AddTotalBalance adds TotalBalanceByEpoch object to the cache. This method also trims the least
// recently added TotalBalanceByEpoch object if the cache size has ready the max cache size limit.
func (c *TotalBalanceCache) AddTotalBalance(totalBalance *TotalBalanceByEpoch) error {
	if !featureconfig.FeatureConfig().EnableTotalBalanceCache {
		return nil
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	if err := c.totalBalanceCache.AddIfNotPresent(totalBalance); err != nil {
		return err
	}

	trim(c.totalBalanceCache, maxTotalBalanceListSize)
	return nil
}
