package cache

import (
	"errors"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"k8s.io/client-go/tools/cache"
)

var (
	// ErrNotCommittee will be returned when a cache object is not a pointer to
	// a Committee struct.
	ErrNotCommittee = errors.New("object is not a committee struct")

	// maxShuffledIndicesSize defines the max number of shuffled indices list can cache.
	// 3 for previous, current epoch and next epoch.
	maxShuffledIndicesSize = 3

	// CommitteeCacheMiss tracks the number of committee requests that aren't present in the cache.
	CommitteeCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "committee_cache_miss",
		Help: "The number of committee requests that aren't present in the cache.",
	})
	// CommitteeCacheHit tracks the number of committee requests that are in the cache.
	CommitteeCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "committee_cache_hit",
		Help: "The number of committee requests that are present in the cache.",
	})
)

// Committee defines the committee per epoch and index.
type Committee struct {
	CommitteeCount uint64
	Epoch          uint64
	Committee      []uint64
}

// CommitteeCache is a struct with 1 queue for looking up shuffled indices list by epoch and committee index.
type CommitteeCache struct {
	CommitteeCache *cache.FIFO
	lock           sync.RWMutex
}

// committeeKeyFn takes the epoch as the key to retrieve shuffled indices of a committee in a given epoch.
func committeeKeyFn(obj interface{}) (string, error) {
	info, ok := obj.(*Committee)
	if !ok {
		return "", ErrNotCommittee
	}

	return strconv.Itoa(int(info.Epoch)), nil
}

// NewCommitteeCache creates a new committee cache for storing/accessing shuffled indices of a committee.
func NewCommitteeCache() *CommitteeCache {
	return &CommitteeCache{
		CommitteeCache: cache.NewFIFO(committeeKeyFn),
	}
}

// ShuffledIndices fetches the shuffled indices by slot and committee index. Every list of indices
// represent one committee. Returns true if the list exists with slot and committee index. Otherwise returns false, nil.
func (c *CommitteeCache) ShuffledIndices(slot uint64, index uint64) ([]uint64, error) {
	if !featureconfig.Get().EnableShuffledIndexCache && !featureconfig.Get().EnableNewCache {
		return nil, nil
	}
	c.lock.RLock()
	defer c.lock.RUnlock()

	epoch := int(slot / params.BeaconConfig().SlotsPerEpoch)
	obj, exists, err := c.CommitteeCache.GetByKey(strconv.Itoa(epoch))
	if err != nil {
		return nil, err
	}

	if exists {
		CommitteeCacheHit.Inc()
	} else {
		CommitteeCacheMiss.Inc()
		return nil, nil
	}

	item, ok := obj.(*Committee)
	if !ok {
		return nil, ErrNotCommittee
	}

	committeeCountPerSlot := uint64(1)
	if item.CommitteeCount/params.BeaconConfig().SlotsPerEpoch > 1 {
		committeeCountPerSlot = item.CommitteeCount / params.BeaconConfig().SlotsPerEpoch
	}

	indexOffSet := index + (slot%params.BeaconConfig().SlotsPerEpoch)*committeeCountPerSlot
	start, end := startEndIndices(item, indexOffSet)
	return item.Committee[start:end], nil
}

// AddCommitteeShuffledList adds Committee shuffled list object to the cache. T
// his method also trims the least recently list if the cache size has ready the max cache size limit.
func (c *CommitteeCache) AddCommitteeShuffledList(committee *Committee) error {
	if !featureconfig.Get().EnableShuffledIndexCache && !featureconfig.Get().EnableNewCache {
		return nil
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	if err := c.CommitteeCache.AddIfNotPresent(committee); err != nil {
		return err
	}
	trim(c.CommitteeCache, maxShuffledIndicesSize)
	return nil
}

// Epochs returns the epochs stored in the committee cache. These are the keys to the cache.
func (c *CommitteeCache) Epochs() ([]uint64, error) {
	if !featureconfig.Get().EnableShuffledIndexCache {
		return nil, nil
	}
	c.lock.RLock()
	defer c.lock.RUnlock()

	epochs := make([]uint64, len(c.CommitteeCache.ListKeys()))
	for i, s := range c.CommitteeCache.ListKeys() {
		epoch, err := strconv.Atoi(s)
		if err != nil {
			return nil, err
		}
		epochs[i] = uint64(epoch)
	}
	return epochs, nil
}

// EpochInCache returns true if an input epoch is part of keys in cache.
func (c *CommitteeCache) EpochInCache(wantedEpoch uint64) (bool, error) {
	if !featureconfig.Get().EnableShuffledIndexCache && !featureconfig.Get().EnableNewCache {
		return false, nil
	}
	c.lock.RLock()
	defer c.lock.RUnlock()

	for _, s := range c.CommitteeCache.ListKeys() {
		epoch, err := strconv.Atoi(s)
		if err != nil {
			return false, err
		}
		if wantedEpoch == uint64(epoch) {
			return true, nil
		}
	}
	return false, nil
}

// CommitteeCountPerSlot returns the number of committees in a given slot as stored in cache.
func (c *CommitteeCache) CommitteeCountPerSlot(slot uint64) (uint64, bool, error) {
	if !featureconfig.Get().EnableShuffledIndexCache && !featureconfig.Get().EnableNewCache {
		return 0, false, nil
	}
	c.lock.RLock()
	defer c.lock.RUnlock()
	epoch := int(slot / params.BeaconConfig().SlotsPerEpoch)
	obj, exists, err := c.CommitteeCache.GetByKey(strconv.Itoa(int(epoch)))
	if err != nil {
		return 0, false, err
	}

	if exists {
		CommitteeCacheHit.Inc()
	} else {
		CommitteeCacheMiss.Inc()
		return 0, false, nil
	}

	item, ok := obj.(*Committee)
	if !ok {
		return 0, false, ErrNotCommittee
	}

	return item.CommitteeCount / params.BeaconConfig().SlotsPerEpoch, true, nil
}

// ActiveIndices returns the active indices of a given epoch stored in cache.
func (c *CommitteeCache) ActiveIndices(epoch uint64) ([]uint64, error) {
	if !featureconfig.Get().EnableShuffledIndexCache && !featureconfig.Get().EnableNewCache {
		return nil, nil
	}

	c.lock.RLock()
	defer c.lock.RUnlock()
	obj, exists, err := c.CommitteeCache.GetByKey(strconv.Itoa(int(epoch)))
	if err != nil {
		return nil, err
	}

	if exists {
		CommitteeCacheHit.Inc()
	} else {
		CommitteeCacheMiss.Inc()
		return nil, nil
	}

	item, ok := obj.(*Committee)
	if !ok {
		return nil, ErrNotCommittee
	}

	return item.Committee, nil
}

func startEndIndices(c *Committee, index uint64) (uint64, uint64) {
	validatorCount := uint64(len(c.Committee))
	start := sliceutil.SplitOffset(validatorCount, c.CommitteeCount, index)
	end := sliceutil.SplitOffset(validatorCount, c.CommitteeCount, index+1)

	return start, end
}
