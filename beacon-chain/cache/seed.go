package cache

import (
	"errors"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"k8s.io/client-go/tools/cache"
)

var (
	// ErrNotSeedInfo will be returned when a cache object is not a pointer to
	// a SeedByEpoch struct.
	ErrNotSeedInfo = errors.New("object is not a seed obj")

	// maxSeedListSize defines the max number of seed can cache.
	maxSeedListSize = 1000

	// Metrics.
	seedCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "seed_cache_miss",
		Help: "The number of seed requests that aren't present in the cache.",
	})
	seedCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "seed_cache_hit",
		Help: "The number of seed requests that are present in the cache.",
	})
)

// SeedByEpoch defines the seed of the epoch.
type SeedByEpoch struct {
	Epoch uint64
	Seed  []byte
}

// SeedCache is a struct with 1 queue for looking up seed by epoch.
type SeedCache struct {
	seedCache *cache.FIFO
	lock      sync.RWMutex
}

// seedKeyFn takes the epoch as the key for the seed of a given epoch.
func seedKeyFn(obj interface{}) (string, error) {
	sInfo, ok := obj.(*SeedByEpoch)
	if !ok {
		return "", ErrNotSeedInfo
	}

	return strconv.Itoa(int(sInfo.Epoch)), nil
}

// NewSeedCache creates a new seed cache for storing/accessing seed.
func NewSeedCache() *SeedCache {
	return &SeedCache{
		seedCache: cache.NewFIFO(seedKeyFn),
	}
}

// SeedInEpoch fetches SeedByEpoch by epoch. Returns true with a
// reference to the SeedInEpoch info, if exists. Otherwise returns false, nil.
func (c *SeedCache) SeedInEpoch(epoch uint64) ([]byte, error) {
	if !featureconfig.FeatureConfig().EnableSeedCache {
		// Return a miss result if cache is not enabled.
		seedCacheMiss.Inc()
		return nil, nil
	}

	c.lock.RLock()
	defer c.lock.RUnlock()
	obj, exists, err := c.seedCache.GetByKey(strconv.Itoa(int(epoch)))
	if err != nil {
		return nil, err
	}

	if exists {
		seedCacheHit.Inc()
	} else {
		seedCacheMiss.Inc()
		return nil, nil
	}

	sInfo, ok := obj.(*SeedByEpoch)
	if !ok {
		return nil, ErrNotSeedInfo
	}

	return sInfo.Seed, nil
}

// AddSeed adds SeedByEpoch object to the cache. This method also trims the least
// recently added SeedByEpoch object if the cache size has ready the max cache size limit.
func (c *SeedCache) AddSeed(seed *SeedByEpoch) error {
	if !featureconfig.FeatureConfig().EnableSeedCache {
		return nil
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	if err := c.seedCache.AddIfNotPresent(seed); err != nil {
		return err
	}

	trim(c.seedCache, maxSeedListSize)
	return nil
}
