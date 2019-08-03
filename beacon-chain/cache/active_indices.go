package cache

import (
	"errors"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"k8s.io/client-go/tools/cache"
)

var (
	// ErrNotActiveIndicesInfo will be returned when a cache object is not a pointer to
	// a ActiveIndicesByEpoch struct.
	ErrNotActiveIndicesInfo = errors.New("object is not a active indices list")

	// maxActiveIndicesListSize defines the max number of active indices can cache.
	maxActiveIndicesListSize = 4

	// Metrics.
	activeIndicesCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "active_validator_indices_cache_miss",
		Help: "The number of active validator indices requests that aren't present in the cache.",
	})
	activeIndicesCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "active_validator_indices_cache_hit",
		Help: "The number of active validator indices requests that are present in the cache.",
	})
)

// ActiveIndicesByEpoch defines the active validator indices per epoch.
type ActiveIndicesByEpoch struct {
	Epoch         uint64
	ActiveIndices []uint64
}

// ActiveIndicesCache is a struct with 1 queue for looking up active indices by epoch.
type ActiveIndicesCache struct {
	activeIndicesCache *cache.FIFO
	lock               sync.RWMutex
}

// activeIndicesKeyFn takes the epoch as the key for the active indices of a given epoch.
func activeIndicesKeyFn(obj interface{}) (string, error) {
	aInfo, ok := obj.(*ActiveIndicesByEpoch)
	if !ok {
		return "", ErrNotActiveIndicesInfo
	}

	return strconv.Itoa(int(aInfo.Epoch)), nil
}

// NewActiveIndicesCache creates a new active indices cache for storing/accessing active validator indices.
func NewActiveIndicesCache() *ActiveIndicesCache {
	return &ActiveIndicesCache{
		activeIndicesCache: cache.NewFIFO(activeIndicesKeyFn),
	}
}

// ActiveIndicesInEpoch fetches ActiveIndicesByEpoch by epoch. Returns true with a
// reference to the ActiveIndicesInEpoch info, if exists. Otherwise returns false, nil.
func (c *ActiveIndicesCache) ActiveIndicesInEpoch(epoch uint64) ([]uint64, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	obj, exists, err := c.activeIndicesCache.GetByKey(strconv.Itoa(int(epoch)))
	if err != nil {
		return nil, err
	}

	if exists {
		activeIndicesCacheHit.Inc()
	} else {
		activeIndicesCacheMiss.Inc()
		return nil, nil
	}

	aInfo, ok := obj.(*ActiveIndicesByEpoch)
	if !ok {
		return nil, ErrNotActiveIndicesInfo
	}

	return aInfo.ActiveIndices, nil
}

// AddActiveIndicesList adds ActiveIndicesByEpoch object to the cache. This method also trims the least
// recently added ActiveIndicesByEpoch object if the cache size has ready the max cache size limit.
func (c *ActiveIndicesCache) AddActiveIndicesList(activeIndices *ActiveIndicesByEpoch) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if err := c.activeIndicesCache.AddIfNotPresent(activeIndices); err != nil {
		return err
	}

	trim(c.activeIndicesCache, maxActiveIndicesListSize)
	return nil
}

// ActiveIndicesKeys returns the keys of the active indices cache.
func (c *ActiveIndicesCache) ActiveIndicesKeys() []string {
	return c.activeIndicesCache.ListKeys()
}
