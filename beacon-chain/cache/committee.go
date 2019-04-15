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
	maxCacheSize = int(4 * params.BeaconConfig().SlotsPerEpoch)

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

// CommitteesInSlot specifies how many CommitteeInfos are in a given slot.
type CommitteesInSlot struct {
	Slot       uint64
	Committees []*CommitteeInfo
}

// CommitteesCache structs with 1 queue for looking up committees by slot.
type CommitteesCache struct {
	committeesCache *cache.FIFO
	lock            sync.RWMutex
}

// slotKeyFn takes the string representation of the slot number as the key
// for the committees of a given slot (CommitteesInSlot).
func slotKeyFn(obj interface{}) (string, error) {
	cInfo, ok := obj.(*CommitteesInSlot)
	if !ok {
		return "", ErrNotACommitteeInfo
	}

	return strconv.Itoa(int(cInfo.Slot)), nil
}

// NewCommitteesCache creates a new committee cache for storing/accessing blockInfo from
// memory.
func NewCommitteesCache() *CommitteesCache {
	return &CommitteesCache{
		committeesCache: cache.NewFIFO(slotKeyFn),
	}
}

// CommitteesInfoBySlot fetches CommitteesInSlot by slot. Returns true with a
// reference to the committees info, if exists. Otherwise returns false, nil.
func (c *CommitteesCache) CommitteesInfoBySlot(slot uint64) (*CommitteesInSlot, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	obj, exists, err := c.committeesCache.GetByKey(strconv.Itoa(int(slot)))
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

// AddCommittees adds CommitteesInSlot object to the cache. This method also trims the least
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
