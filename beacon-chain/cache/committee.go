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
	// ErrNotACommitteeInfo will be returned when a cache object is not a pointer to
	// a committeeInfo struct.
	ErrNotACommitteeInfo = errors.New("object is not an committee info")

	// maxCacheSize is 4x of the epoch length for additional cache padding.
	// Requests should be only accessing committees within defined epoch length.
	maxCacheSize = int(2 * params.BeaconConfig().SlotsPerEpoch)

	// Metrics
	committeeCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "committee_cache_miss",
		Help: "The number of committee requests that aren't present in the cache.",
	})
	committeeCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "committee_cache_hit",
		Help: "The number of committee requests that are present in the cache.",
	})
	committeeCacheSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "committee_cache_size",
		Help: "The number of committees in the committee cache",
	})
)

// CommitteeInfo defines the validator committee of slot and shard combinations.
type CommitteeInfo struct {
	Committee []uint64
	Shard     uint64
}

// CommitteesInSlot species the committees of a given slot.
type CommitteesInSlot struct {
	Slot       int
	Committees []*CommitteeInfo
}

// CommitteesCache struct with 1 queue for looking up crosslink committees by slot.
type CommitteesCache struct {
	committeesCache *cache.FIFO
	lock            sync.RWMutex
}

// slotKeyFn takes the string representation of the slot number as the key
// for a committeeInfo.
func slotKeyFn(obj interface{}) (string, error) {
	cInfo, ok := obj.(*CommitteesInSlot)
	if !ok {
		return "", ErrNotACommitteeInfo
	}

	return strconv.Itoa(cInfo.Slot), nil
}

// NewCommitteesCache creates a new committee cache for storing/accessing blockInfo from
// memory.
func NewCommitteesCache() *CommitteesCache {
	return &CommitteesCache{
		committeesCache: cache.NewFIFO(slotKeyFn),
	}
}

// CommitteesInfoBySlot fetches committeesInfo by slot. Returns true with a
// reference to the committees info, if exists. Otherwise returns false, nil.
func (c *CommitteesCache) CommitteesInfoBySlot(slot int) (*CommitteesInSlot, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	obj, exists, err := c.committeesCache.GetByKey(strconv.Itoa(slot))
	if err != nil {
		return nil, err
	}

	if exists {
		committeeCacheHit.Inc()
	} else {
		committeeCacheMiss.Inc()
		return nil, nil
	}

	cInfo, ok := obj.(*CommitteesInSlot)
	if !ok {
		return nil, ErrNotACommitteeInfo
	}

	return cInfo, nil
}

// AddCommittees adds committeeInfo object to the cache. This method also trims the least
// recently added committeeInfo object if the cache size has ready the max cache size limit.
func (c *CommitteesCache) AddCommittees(committees *CommitteesInSlot) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if err := c.committeesCache.AddIfNotPresent(committees); err != nil {
		return err
	}

	trim(c.committeesCache, maxCacheSize)
	committeeCacheSize.Set(float64(len(c.committeesCache.ListKeys())))
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
