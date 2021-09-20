// +build !libfuzzer

package cache

import (
	"errors"
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	types "github.com/prysmaticlabs/eth2-types"
	lruwrpr "github.com/prysmaticlabs/prysm/cache/lru"
	"github.com/prysmaticlabs/prysm/container/slice"
	"github.com/prysmaticlabs/prysm/math"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var (
	// maxCommitteesCacheSize defines the max number of shuffled committees on per randao basis can cache.
	// Due to reorgs and long finality, it's good to keep the old cache around for quickly switch over.
	maxCommitteesCacheSize = uint64(32)

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

// CommitteeCache is a struct with 1 queue for looking up shuffled indices list by seed.
type CommitteeCache struct {
	CommitteeCache *lru.Cache
	lock           sync.RWMutex
}

// committeeKeyFn takes the seed as the key to retrieve shuffled indices of a committee in a given epoch.
func committeeKeyFn(obj interface{}) (string, error) {
	info, ok := obj.(*Committees)
	if !ok {
		return "", ErrNotCommittee
	}
	return key(info.Seed), nil
}

// NewCommitteesCache creates a new committee cache for storing/accessing shuffled indices of a committee.
func NewCommitteesCache() *CommitteeCache {
	return &CommitteeCache{
		CommitteeCache: lruwrpr.New(int(maxCommitteesCacheSize)),
	}
}

// Committee fetches the shuffled indices by slot and committee index. Every list of indices
// represent one committee. Returns true if the list exists with slot and committee index. Otherwise returns false, nil.
func (c *CommitteeCache) Committee(slot types.Slot, seed [32]byte, index types.CommitteeIndex) ([]types.ValidatorIndex, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	obj, exists := c.CommitteeCache.Get(key(seed))
	if exists {
		CommitteeCacheHit.Inc()
	} else {
		CommitteeCacheMiss.Inc()
		return nil, nil
	}

	item, ok := obj.(*Committees)
	if !ok {
		return nil, ErrNotCommittee
	}

	committeeCountPerSlot := uint64(1)
	if item.CommitteeCount/uint64(params.BeaconConfig().SlotsPerEpoch) > 1 {
		committeeCountPerSlot = item.CommitteeCount / uint64(params.BeaconConfig().SlotsPerEpoch)
	}

	indexOffSet, err := math.Add64(uint64(index), uint64(slot.ModSlot(params.BeaconConfig().SlotsPerEpoch).Mul(committeeCountPerSlot)))
	if err != nil {
		return nil, err
	}
	start, end := startEndIndices(item, indexOffSet)

	if end > uint64(len(item.ShuffledIndices)) || end < start {
		return nil, errors.New("requested index out of bound")
	}

	return item.ShuffledIndices[start:end], nil
}

// AddCommitteeShuffledList adds Committee shuffled list object to the cache. T
// his method also trims the least recently list if the cache size has ready the max cache size limit.
func (c *CommitteeCache) AddCommitteeShuffledList(committees *Committees) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	key, err := committeeKeyFn(committees)
	if err != nil {
		return err
	}
	_ = c.CommitteeCache.Add(key, committees)
	return nil
}

// ActiveIndices returns the active indices of a given seed stored in cache.
func (c *CommitteeCache) ActiveIndices(seed [32]byte) ([]types.ValidatorIndex, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	obj, exists := c.CommitteeCache.Get(key(seed))

	if exists {
		CommitteeCacheHit.Inc()
	} else {
		CommitteeCacheMiss.Inc()
		return nil, nil
	}

	item, ok := obj.(*Committees)
	if !ok {
		return nil, ErrNotCommittee
	}

	return item.SortedIndices, nil
}

// ActiveIndicesCount returns the active indices count of a given seed stored in cache.
func (c *CommitteeCache) ActiveIndicesCount(seed [32]byte) (int, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	obj, exists := c.CommitteeCache.Get(key(seed))
	if exists {
		CommitteeCacheHit.Inc()
	} else {
		CommitteeCacheMiss.Inc()
		return 0, nil
	}

	item, ok := obj.(*Committees)
	if !ok {
		return 0, ErrNotCommittee
	}

	return len(item.SortedIndices), nil
}

// HasEntry returns true if the committee cache has a value.
func (c *CommitteeCache) HasEntry(seed string) bool {
	_, ok := c.CommitteeCache.Get(seed)
	return ok
}

func startEndIndices(c *Committees, index uint64) (uint64, uint64) {
	validatorCount := uint64(len(c.ShuffledIndices))
	start := slice.SplitOffset(validatorCount, c.CommitteeCount, index)
	end := slice.SplitOffset(validatorCount, c.CommitteeCount, index+1)
	return start, end
}

// Using seed as source for key to handle reorgs in the same epoch.
// The seed is derived from state's array of randao mixes and epoch value
// hashed together. This avoids collisions on different validator set. Spec definition:
// https://github.com/ethereum/consensus-specs/blob/v0.9.3/specs/core/0_beacon-chain.md#get_seed
func key(seed [32]byte) string {
	return string(seed[:])
}
