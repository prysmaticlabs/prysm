//go:build !fuzz

package cache

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/container/slice"
	mathutil "github.com/prysmaticlabs/prysm/v5/math"
	log "github.com/sirupsen/logrus"
)

const (
	// maxCommitteesCacheSize defines the max number of shuffled committees on per randao basis can cache.
	// Due to reorgs and long finality, it's good to keep the old cache around for quickly switch over.
	maxCommitteesCacheSize = int(4)
	// expandedCommitteeCacheSize defines the expanded size of the committee cache in the event we
	// do not have finality to deal with long forks better.
	expandedCommitteeCacheSize = int(32)
)

var (
	// committeeCacheMiss tracks the number of committee requests that aren't present in the cache.
	committeeCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "committee_cache_miss",
		Help: "The number of committee requests that aren't present in the cache.",
	})
	// committeeCacheHit tracks the number of committee requests that are in the cache.
	committeeCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "committee_cache_hit",
		Help: "The number of committee requests that are present in the cache.",
	})
)

var (
	ErrRequestIndexOutOfBound = errors.New("requested index out of bound")
)

// CommitteeCache is a struct with 1 queue for looking up shuffled indices list by seed.
type CommitteeCache[K string, V Committees] struct {
	lru                         *lru.Cache[K, V]
	promCacheMiss, promCacheHit prometheus.Counter

	lock       sync.RWMutex
	inProgress map[string]bool
	size       int
}

// NewCommitteesCache creates a new committee cache for storing/accessing shuffled indices of a committee.
func NewCommitteesCache[K string, V Committees]() *CommitteeCache[K, V] {
	return &CommitteeCache[K, V]{
		lru:           newLRUCacheOrPanics[K, V](maxCommitteesCacheSize, committeeCacheHit, committeeCacheMiss),
		promCacheMiss: committeeCacheMiss,
		promCacheHit:  committeeCacheHit,
		inProgress:    make(map[string]bool),
	}
}

// get returns the underlying lru cache.
func (c *CommitteeCache[K, V]) get() *lru.Cache[K, V] { //nolint: unused, -- bug in golangci-lint v1.55.2
	return c.lru
}

// hitCache increments the cache miss counter.
func (c *CommitteeCache[K, V]) hitCache() { //nolint: unused, -- bug in golangci-lint v1.55.2
	c.promCacheHit.Inc()
}

// missCache increments the cache miss counter.
func (c *CommitteeCache[K, V]) missCache() { //nolint: unused, -- bug in golangci-lint v1.55.2
	c.promCacheMiss.Inc()
}

// Clear the CommitteeCache to its initial state
func (c *CommitteeCache[K, V]) Clear() {
	c.lock.Lock()
	defer c.lock.Unlock()

	purge[K, V](c)
	c.compressCommitteeCache()
	c.inProgress = make(map[string]bool)
}

// ------------------------------------------------------------------------------------ //

// ExpandCommitteeCache expands the size of the committee cache.
func (c *CommitteeCache[K, V]) ExpandCommitteeCache() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.size == expandedCommitteeCacheSize {
		return
	}
	resize[K, V](c, expandedCommitteeCacheSize)
	c.size = expandedCommitteeCacheSize
	log.Warnf("Expanding committee cache size from %d to %d", maxCommitteesCacheSize, expandedCommitteeCacheSize)
}

// compressCommitteeCache compresses the size of the committee cache.
// This method is not thread safe and should be called with a lock.
func (c *CommitteeCache[K, V]) compressCommitteeCache() {
	if c.size == maxCommitteesCacheSize {
		return
	}
	resize[K, V](c, maxCommitteesCacheSize)
	c.size = maxCommitteesCacheSize
	log.Warnf("Reducing committee cache size from %d to %d", expandedCommitteeCacheSize, maxCommitteesCacheSize)
}

// CompressCommitteeCache compresses the size of the committee cache.
// This method is thread safe and should be called without a lock.
func (c *CommitteeCache[K, V]) CompressCommitteeCache() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.size == maxCommitteesCacheSize {
		return
	}
	resize[K, V](c, maxCommitteesCacheSize)
	c.size = maxCommitteesCacheSize
	log.Warnf("Reducing committee cache size from %d to %d", expandedCommitteeCacheSize, maxCommitteesCacheSize)
}

// Committee fetches the shuffled indices by slot and committee index. Every list of indices
// represent one committee. Returns true if the list exists with slot and committee index. Otherwise returns false, nil.
func (c *CommitteeCache[K, V]) Committee(ctx context.Context, slot primitives.Slot, seed [32]byte, index primitives.CommitteeIndex) ([]primitives.ValidatorIndex, error) {
	var err error
	if err = c.checkInProgress(ctx, seed); err != nil {
		return nil, fmt.Errorf("failed to check progress: %w", err)
	}

	var item V
	if item, err = get[K, V](c, K(committeeCachesKey(seed))); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	committees, ok := any(item).(Committees)
	if !ok {
		return nil, fmt.Errorf("%v: %w", ErrCast, errNotCommittees)
	}

	committeeCountPerSlot := uint64(1)
	if committees.CommitteeCount/uint64(params.BeaconConfig().SlotsPerEpoch) > 1 {
		committeeCountPerSlot = committees.CommitteeCount / uint64(params.BeaconConfig().SlotsPerEpoch)
	}

	indexOffSet, err := mathutil.Add64(uint64(index), uint64(slot.ModSlot(params.BeaconConfig().SlotsPerEpoch).Mul(committeeCountPerSlot)))
	if err != nil {
		return nil, fmt.Errorf("failed to calculate indexOffSet: %w", err)
	}
	start, end := startEndIndices(&committees, indexOffSet)

	if end > uint64(len(committees.ShuffledIndices)) || end < start {
		return nil, ErrRequestIndexOutOfBound
	}

	return committees.ShuffledIndices[start:end], nil
}

// AddCommitteeShuffledList adds Committee shuffled list object to the cache. T
// his method also trims the least recently list if the cache size has ready the max cache size limit.
func (c *CommitteeCache[K, V]) AddCommitteeShuffledList(ctx context.Context, committees *V) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("failed to continue with context error: %w", err)
	}

	key, err := committeeCachesKeyFn[K, V](committees)
	if err != nil {
		return fmt.Errorf("failed to compute committee key fn: %w", err)
	}

	return add[K, V](c, key, *committees)
}

// ActiveIndices returns the active indices of a given seed stored in cache.
func (c *CommitteeCache[K, V]) ActiveIndices(ctx context.Context, seed [32]byte) ([]primitives.ValidatorIndex, error) {
	var err error
	if err = c.checkInProgress(ctx, seed); err != nil {
		return nil, fmt.Errorf("failed to check progress: %w", err)
	}

	var item V
	if item, err = get[K, V](c, K(committeeCachesKey(seed))); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	committees, ok := any(item).(Committees)
	if !ok {
		return nil, fmt.Errorf("%v: %w", ErrCast, errNotCommittees)
	}

	return committees.SortedIndices, nil
}

// ActiveIndicesCount returns the active indices count of a given seed stored in cache.
func (c *CommitteeCache[K, V]) ActiveIndicesCount(ctx context.Context, seed [32]byte) (int, error) {
	indices, err := c.ActiveIndices(ctx, seed)
	if err != nil {
		return 0, err
	}

	return len(indices), nil
}

// HasEntry returns true if the committee cache has a value.
func (c *CommitteeCache[K, V]) HasEntry(seed [32]byte) bool {
	return exist[K, V](c, K(committeeCachesKey(seed)))
}

// MarkInProgress a request so that any other similar requests will block on
// Get until MarkNotInProgress is called.
func (c *CommitteeCache[K, V]) MarkInProgress(seed [32]byte) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	s := committeeCachesKey(seed)
	if c.inProgress[s] {
		return ErrAlreadyInProgress
	}
	c.inProgress[s] = true
	return nil
}

// MarkNotInProgress will release the lock on a given request. This should be
// called after put.
func (c *CommitteeCache[K, V]) MarkNotInProgress(seed [32]byte) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	s := committeeCachesKey(seed)
	delete(c.inProgress, s)
	return nil
}

func startEndIndices(c *Committees, index uint64) (uint64, uint64) {
	validatorCount := uint64(len(c.ShuffledIndices))
	start := slice.SplitOffset(validatorCount, c.CommitteeCount, index)
	end := slice.SplitOffset(validatorCount, c.CommitteeCount, index+1)
	return start, end
}

func (c *CommitteeCache[K, V]) checkInProgress(ctx context.Context, seed [32]byte) error {
	delay := minDelay
	// Another identical request may be in progress already. Let's wait until
	// any in progress request resolves or our timeout is exceeded.
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		c.lock.RLock()
		if !c.inProgress[committeeCachesKey(seed)] {
			c.lock.RUnlock()
			break
		}
		c.lock.RUnlock()

		// This increasing backoff is to decrease the CPU cycles while waiting
		// for the in progress boolean to flip to false.
		time.Sleep(time.Duration(delay) * time.Nanosecond)
		delay *= delayFactor
		delay = math.Min(delay, maxDelay)
	}
	return nil
}

// Using seed as source for key to handle reorgs in the same epoch.
// The seed is derived from state's array of randao mixes and epoch value
// hashed together. This avoids collisions on different validator set. Spec definition:
// https://github.com/ethereum/consensus-specs/blob/v0.9.3/specs/core/0_beacon-chain.md#get_seed
func committeeCachesKey[K string](seed [32]byte) K {
	return K(seed[:])
}

// committeeCachesKeyFn takes the seed as the key to retrieve shuffled indices of a committee in a given epoch.
func committeeCachesKeyFn[K string, V Committees](obj *V) (K, error) {
	var noKey K
	committees, ok := any(obj).(*Committees)
	if !ok {
		return noKey, fmt.Errorf("%v: %w", ErrCast, errNotCommittees)
	}

	if committees == nil {
		return noKey, ErrNilValueProvided
	}

	return committeeCachesKey[K](committees.Seed), nil
}
