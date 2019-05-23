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
	// ErrNotValidatorListInfo will be returned when a cache object is not a pointer to
	// a ValidatorList struct.
	ErrNotValidatorListInfo = errors.New("object is not a shuffled validator list")

	// maxCacheSize can handle max 4 shuffled validator lists
	maxCacheSize = 4

	// Metrics
	shuffledValidatorsCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "shuffled_validators_cache_miss",
		Help: "The number of shuffled validators requests that aren't present in the cache.",
	})
	shuffledValidatorsCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "shuffled_validators_cache_hit",
		Help: "The number of shuffled validators requests that are present in the cache.",
	})
)


// ShuffledValidatorsInEpoch specifies how many shuffled validators are in a given epoch and seed.
type ShuffledValidatorsInEpoch struct {
	Epoch       uint64
	Seed []byte
	ShuffledValidators []uint64
}

// ShuffledValidatorsCache is a struct with 1 queue for looking up shuffled validators by epoch.
type ShuffledValidatorsCache struct {
	shuffledValidatorsCache *cache.FIFO
	lock            sync.RWMutex
}

// slotKeyFn takes the string representation of the epoch number and seed as the key
// for the shuffled validators of a given epoch.
func shuffleKeyFn(obj interface{}) (string, error) {
	sInfo, ok := obj.(*ShuffledValidatorsInEpoch)
	if !ok {
		return "", ErrNotValidatorListInfo
	}

	return strconv.Itoa(int(sInfo.Epoch)) + string(sInfo.Seed), nil
}

// NewShuffledValidatorsCache creates a new shuffled validators cache for storing/accessing shuffled validator indices
func NewShuffledValidatorsCache() *ShuffledValidatorsCache {
	return &ShuffledValidatorsCache{
		shuffledValidatorsCache: cache.NewFIFO(shuffleKeyFn),
	}
}

// ShuffledValidatorsByEpoch fetches ShuffledValidatorsInEpoch by epoch and seed. Returns true with a
// reference to the ShuffledValidatorsInEpoch info, if exists. Otherwise returns false, nil.
func (c *ShuffledValidatorsCache) ShuffledValidatorsByEpoch(epoch uint64, seed []byte) ([]uint64, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	key := strconv.Itoa(int(epoch)) + string(seed)
	obj, exists, err := c.shuffledValidatorsCache.GetByKey(key)
	if err != nil {
		return nil, err
	}

	if exists {
		shuffledValidatorsCacheHit.Inc()
	} else {
		shuffledValidatorsCacheMiss.Inc()
		return nil, nil
	}

	cInfo, ok := obj.(*ShuffledValidatorsInEpoch)
	if !ok {
		return nil, ErrNotValidatorListInfo
	}

	return cInfo.ShuffledValidators, nil
}

// AddCommittees adds CommitteesInSlot object to the cache. This method also trims the least
// recently added shuffled_validatorsInfo object if the cache size has ready the max cache size limit.
func (c *ShuffledValidatorsCache) AddShuffledValidatorList(shuffledValidators *ShuffledValidatorsInEpoch) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if err := c.shuffledValidatorsCache.AddIfNotPresent(shuffledValidators); err != nil {
		return err
	}

	trim(c.shuffledValidatorsCache, maxCacheSize)
	return nil
}

// trim the FIFO queue to the maxSize.
func trim(queue *cache.FIFO, maxSize int) {
	for s := len(queue.ListKeys()); s > maxSize; s-- {
		// #nosec G104 popProcessNoopFunc never returns an error
		_, _ = queue.Pop(popProcessNoopFunc)
	}
}

// popProcessNoopFunc is a no-op function that never returns an error.
func popProcessNoopFunc(obj interface{}) error {
	return nil
}
