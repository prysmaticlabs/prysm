//go:build fuzz

// This file is used in fuzzer builds to bypass global committee caches.
package cache

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
)

var (
	// committeeCacheMiss tracks the number of committee requests that aren't present in the cache.
	fakeCommitteeCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "fake_committee_cache_miss",
		Help: "The number of committee requests that aren't present in the cache.",
	})
	// committeeCacheHit tracks the number of committee requests that are in the cache.
	fakeCommitteeCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "fake_committee_cache_hit",
		Help: "The number of committee requests that are present in the cache.",
	})
)

func InitializeCommitteeCacheOrPanic[K string, V Committees]() *FakeCommitteeCache[K, V] {
	c, err := NewCommitteesCache[K, V]()
	if err != nil {
		panic(err)
	}
	return c
}

// FakeCommitteeCache is a struct with 1 queue for looking up shuffled indices list by seed.
type FakeCommitteeCache[K string, V Committees] struct {
	promCacheMiss, promCacheHit prometheus.Counter
}

// NewCommitteesCache creates a new committee cache for storing/accessing shuffled indices of a committee.
func NewCommitteesCache[K string, V Committees]() (*FakeCommitteeCache[K, V], error) {
	return &FakeCommitteeCache[K, V]{
		promCacheMiss: fakeCommitteeCacheMiss,
		promCacheHit:  fakeCommitteeCacheHit,
	}, nil
}

// Committee fetches the shuffled indices by slot and committee index. Every list of indices
// represent one committee. Returns true if the list exists with slot and committee index. Otherwise returns false, nil.
func (c *FakeCommitteeCache[K, V]) Committee(ctx context.Context, slot primitives.Slot, seed [32]byte, index primitives.CommitteeIndex) ([]primitives.ValidatorIndex, error) {
	return nil, nil
}

// AddCommitteeShuffledList adds Committee shuffled list object to the cache. T
// his method also trims the least recently list if the cache size has ready the max cache size limit.
func (c *FakeCommitteeCache[K, V]) AddCommitteeShuffledList(ctx context.Context, committees *V) error {
	return nil
}

// AddProposerIndicesList updates the committee shuffled list with proposer indices.
func (c *FakeCommitteeCache[K, V]) AddProposerIndicesList(seed [32]byte, indices []primitives.ValidatorIndex) error {
	return nil
}

// ActiveIndices returns the active indices of a given seed stored in cache.
func (c *FakeCommitteeCache[K, V]) ActiveIndices(ctx context.Context, seed [32]byte) ([]primitives.ValidatorIndex, error) {
	return nil, nil
}

// ActiveIndicesCount returns the active indices count of a given seed stored in cache.
func (c *FakeCommitteeCache[K, V]) ActiveIndicesCount(ctx context.Context, seed [32]byte) (int, error) {
	return 0, nil
}

// ActiveBalance returns the active balance of a given seed stored in cache.
func (c *FakeCommitteeCache[K, V]) ActiveBalance(seed [32]byte) (uint64, error) {
	return 0, nil
}

// ProposerIndices returns the proposer indices of a given seed.
func (c *FakeCommitteeCache[K, V]) ProposerIndices(seed [32]byte) ([]primitives.ValidatorIndex, error) {
	return nil, nil
}

// HasEntry returns true if the committee cache has a value.
func (c *FakeCommitteeCache[K, V]) HasEntry(seed [32]byte) bool {
	return false
}

// MarkInProgress is a stub.
func (c *FakeCommitteeCache[K, V]) MarkInProgress(seed [32]byte) error {
	return nil
}

// MarkNotInProgress is a stub.
func (c *FakeCommitteeCache[K, V]) MarkNotInProgress(seed [32]byte) error {
	return nil
}

// Clear is a stub.
func (c *FakeCommitteeCache[K, V]) Clear() {
	return
}

func (c *FakeCommitteeCache[K, V]) ExpandCommitteeCache() {
	return
}

func (c *FakeCommitteeCache[K, V]) CompressCommitteeCache() {
	return
}
