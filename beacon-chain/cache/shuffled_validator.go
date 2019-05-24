package cache

import (
	"errors"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"k8s.io/client-go/tools/cache"
)

var (
	// ErrNotValidatorListInfo will be returned when a cache object is not a pointer to
	// a ValidatorList struct.
	ErrNotValidatorListInfo = errors.New("object is not a shuffled validator list")

	// maxShuffledListSize defines the max number of shuffled list can cache.
	maxShuffledListSize = 4

	// Metrics.
	shuffledValidatorsCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "shuffled_validators_cache_miss",
		Help: "The number of shuffled validators requests that aren't present in the cache.",
	})
	shuffledValidatorsCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "shuffled_validators_cache_hit",
		Help: "The number of shuffled validators requests that are present in the cache.",
	})
)


// ShuffledValidatorsBySeed defines the shuffled validator indices per randao seed.
type ShuffledValidatorsBySeed struct {
	Seed []byte
	ShuffledValidators []uint64
}

// ShuffledValidatorsCache is a struct with 1 queue for looking up shuffled validators by seed.
type ShuffledValidatorsCache struct {
	shuffledValidatorsCache *cache.FIFO
	lock            sync.RWMutex
}

// slotKeyFn takes the randao seed as the key for the shuffled validators of a given epoch.
func shuffleKeyFn(obj interface{}) (string, error) {
	sInfo, ok := obj.(*ShuffledValidatorsBySeed)
	if !ok {
		return "", ErrNotValidatorListInfo
	}

	return string(sInfo.Seed), nil
}

// NewShuffledValidatorsCache creates a new shuffled validators cache for storing/accessing shuffled validator indices
func NewShuffledValidatorsCache() *ShuffledValidatorsCache {
	return &ShuffledValidatorsCache{
		shuffledValidatorsCache: cache.NewFIFO(shuffleKeyFn),
	}
}

// ShuffledValidatorsByEpoch fetches ShuffledValidatorsInEpoch by epoch and seed. Returns true with a
// reference to the ShuffledValidatorsInEpoch info, if exists. Otherwise returns false, nil.
func (c *ShuffledValidatorsCache) ShuffledValidatorsByEpoch(seed []byte) ([]uint64, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	obj, exists, err := c.shuffledValidatorsCache.GetByKey(string(seed))
	if err != nil {
		return nil, err
	}

	if exists {
		shuffledValidatorsCacheHit.Inc()
	} else {
		shuffledValidatorsCacheMiss.Inc()
		return nil, nil
	}

	cInfo, ok := obj.(*ShuffledValidatorsBySeed)
	if !ok {
		return nil, ErrNotValidatorListInfo
	}

	return cInfo.ShuffledValidators, nil
}

// AddCommittees adds CommitteesInSlot object to the cache. This method also trims the least
// recently added shuffled_validatorsInfo object if the cache size has ready the max cache size limit.
func (c *ShuffledValidatorsCache) AddShuffledValidatorList(shuffledValidators *ShuffledValidatorsBySeed) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if err := c.shuffledValidatorsCache.AddIfNotPresent(shuffledValidators); err != nil {
		return err
	}

	trim(c.shuffledValidatorsCache, maxShuffledListSize)
	return nil
}
