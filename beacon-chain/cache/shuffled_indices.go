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
	shuffledIndicesCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "shuffled_validators_cache_miss",
		Help: "The number of shuffled validators requests that aren't present in the cache.",
	})
	shuffledIndicesCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "shuffled_validators_cache_hit",
		Help: "The number of shuffled validators requests that are present in the cache.",
	})
)

// ShuffledIndicessBySeed defines the shuffled validator indices per randao seed.
type ShuffledIndicesBySeed struct {
	Seed            []byte
	ShuffledIndices []uint64
}

// ShuffledIndicesCache is a struct with 1 queue for looking up shuffled validators by seed.
type ShuffledIndicesCache struct {
	shuffledIndicesCache *cache.FIFO
	lock                 sync.RWMutex
}

// slotKeyFn takes the randao seed as the key for the shuffled validators of a given epoch.
func shuffleKeyFn(obj interface{}) (string, error) {
	sInfo, ok := obj.(*ShuffledIndicesBySeed)
	if !ok {
		return "", ErrNotValidatorListInfo
	}

	return string(sInfo.Seed), nil
}

// NewShuffledIndicesCache creates a new shuffled validators cache for storing/accessing shuffled validator indices
func NewShuffledIndicesCache() *ShuffledIndicesCache {
	return &ShuffledIndicesCache{
		shuffledIndicesCache: cache.NewFIFO(shuffleKeyFn),
	}
}

// ShuffledIndicesBySeed fetches ShuffledIndicesBySeed by epoch and seed. Returns true with a
// reference to the ShuffledIndicesInEpoch info, if exists. Otherwise returns false, nil.
func (c *ShuffledIndicesCache) ShuffledIndicesBySeed(seed []byte) ([]uint64, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	obj, exists, err := c.shuffledIndicesCache.GetByKey(string(seed))
	if err != nil {
		return nil, err
	}

	if exists {
		shuffledIndicesCacheHit.Inc()
	} else {
		shuffledIndicesCacheMiss.Inc()
		return nil, nil
	}

	cInfo, ok := obj.(*ShuffledIndicesBySeed)
	if !ok {
		return nil, ErrNotValidatorListInfo
	}

	return cInfo.ShuffledIndices, nil
}

// AddCommittees adds CommitteesInSlot object to the cache. This method also trims the least
// recently added ShuffledIndicessBySeed object if the cache size has ready the max cache size limit.
func (c *ShuffledIndicesCache) AddShuffledValidatorList(shuffledIndices *ShuffledIndicesBySeed) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if err := c.shuffledIndicesCache.AddIfNotPresent(shuffledIndices); err != nil {
		return err
	}

	trim(c.shuffledIndicesCache, maxShuffledListSize)
	return nil
}
