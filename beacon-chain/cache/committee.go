// -build libfuzzer

package cache

import (
	"errors"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"k8s.io/client-go/tools/cache"
)

var (
	// ErrNotCommittee will be returned when a cache object is not a pointer to
	// a Committee struct.
	ErrNotCommittee = errors.New("object is not a committee struct")

	// maxCommitteesCacheSize defines the max number of shuffled committees on per randao basis can cache.
	// Due to reorgs, it's good to keep the old cache around for quickly switch over. 10 is a generous
	// cache size as it considers 3 concurrent branches over 3 epochs.
	maxCommitteesCacheSize = uint64(10)

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

// Committees defines the shuffled committees seed.
type Committees struct {
	CommitteeCount  uint64
	Seed            [32]byte
	ShuffledIndices []uint64
	SortedIndices   []uint64
	ProposerIndices []uint64
}

// CommitteeCache is a struct with 1 queue for looking up shuffled indices list by seed.
type CommitteeCache struct {
	CommitteeCache *cache.FIFO
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
		CommitteeCache: cache.NewFIFO(committeeKeyFn),
	}
}

// Committee fetches the shuffled indices by slot and committee index. Every list of indices
// represent one committee. Returns true if the list exists with slot and committee index. Otherwise returns false, nil.
func (c *CommitteeCache) Committee(slot uint64, seed [32]byte, index uint64) ([]uint64, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	obj, exists, err := c.CommitteeCache.GetByKey(key(seed))
	if err != nil {
		return nil, err
	}

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
	if item.CommitteeCount/params.BeaconConfig().SlotsPerEpoch > 1 {
		committeeCountPerSlot = item.CommitteeCount / params.BeaconConfig().SlotsPerEpoch
	}

	indexOffSet := index + (slot%params.BeaconConfig().SlotsPerEpoch)*committeeCountPerSlot
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

	if err := c.CommitteeCache.AddIfNotPresent(committees); err != nil {
		return err
	}
	trim(c.CommitteeCache, maxCommitteesCacheSize)
	return nil
}

// AddProposerIndicesList updates the committee shuffled list with proposer indices.
func (c *CommitteeCache) AddProposerIndicesList(seed [32]byte, indices []uint64) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	obj, exists, err := c.CommitteeCache.GetByKey(key(seed))
	if err != nil {
		return err
	}
	if !exists {
		committees := &Committees{ProposerIndices: indices}
		if err := c.CommitteeCache.Add(committees); err != nil {
			return err
		}
	} else {
		committees, ok := obj.(*Committees)
		if !ok {
			return ErrNotCommittee
		}
		committees.ProposerIndices = indices
		if err := c.CommitteeCache.Add(committees); err != nil {
			return err
		}
	}

	trim(c.CommitteeCache, maxCommitteesCacheSize)
	return nil
}

// ActiveIndices returns the active indices of a given seed stored in cache.
func (c *CommitteeCache) ActiveIndices(seed [32]byte) ([]uint64, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	obj, exists, err := c.CommitteeCache.GetByKey(key(seed))
	if err != nil {
		return nil, err
	}

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
	obj, exists, err := c.CommitteeCache.GetByKey(key(seed))
	if err != nil {
		return 0, err
	}

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

// ProposerIndices returns the proposer indices of a given seed.
func (c *CommitteeCache) ProposerIndices(seed [32]byte) ([]uint64, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	obj, exists, err := c.CommitteeCache.GetByKey(key(seed))
	if err != nil {
		return nil, err
	}

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

	return item.ProposerIndices, nil
}

// HasEntry returns true if the committee cache has a value.
func (c *CommitteeCache) HasEntry(seed string) bool {
	_, ok, err := c.CommitteeCache.GetByKey(seed)
	return err == nil && ok
}

func startEndIndices(c *Committees, index uint64) (uint64, uint64) {
	validatorCount := uint64(len(c.ShuffledIndices))
	start := sliceutil.SplitOffset(validatorCount, c.CommitteeCount, index)
	end := sliceutil.SplitOffset(validatorCount, c.CommitteeCount, index+1)
	return start, end
}

// Using seed as source for key to handle reorgs in the same epoch.
// The seed is derived from state's array of randao mixes and epoch value
// hashed together. This avoids collisions on different validator set. Spec definition:
// https://github.com/ethereum/eth2.0-specs/blob/v0.9.3/specs/core/0_beacon-chain.md#get_seed
func key(seed [32]byte) string {
	return string(seed[:])
}
