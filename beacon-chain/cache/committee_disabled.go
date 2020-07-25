// +build libfuzzer

package cache

import "errors"

var ErrNotCommittee = errors.New("object is not a committee struct")

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
}

// NewCommitteesCache creates a new committee cache for storing/accessing shuffled indices of a committee.
func NewCommitteesCache() *CommitteeCache {
	return &CommitteeCache{}
}

// Committee fetches the shuffled indices by slot and committee index. Every list of indices
// represent one committee. Returns true if the list exists with slot and committee index. Otherwise returns false, nil.
func (c *CommitteeCache) Committee(slot uint64, seed [32]byte, index uint64) ([]uint64, error) {
	return nil, nil
}

// AddCommitteeShuffledList adds Committee shuffled list object to the cache. T
// his method also trims the least recently list if the cache size has ready the max cache size limit.
func (c *CommitteeCache) AddCommitteeShuffledList(committees *Committees) error {
	return nil
}

// AddProposerIndicesList updates the committee shuffled list with proposer indices.
func (c *CommitteeCache) AddProposerIndicesList(seed [32]byte, indices []uint64) error {
	return nil
}

// ActiveIndices returns the active indices of a given seed stored in cache.
func (c *CommitteeCache) ActiveIndices(seed [32]byte) ([]uint64, error) {
	return nil, nil
}

// ActiveIndicesCount returns the active indices count of a given seed stored in cache.
func (c *CommitteeCache) ActiveIndicesCount(seed [32]byte) (int, error) {
	return 0, nil
}

// ProposerIndices returns the proposer indices of a given seed.
func (c *CommitteeCache) ProposerIndices(seed [32]byte) ([]uint64, error) {
	return nil, nil
}

// HasEntry ... TODO.
func (c *CommitteeCache) HasEntry(string) bool {
	return false
}
