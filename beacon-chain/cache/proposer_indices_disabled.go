// +build libfuzzer

// This file is used in fuzzer builds to bypass proposer indices caches.
package cache

import types "github.com/prysmaticlabs/eth2-types"

// FakeProposerIndicesCache is a struct with 1 queue for looking up proposer indices by root.
type FakeProposerIndicesCache struct {
}

// NewProposerIndicesCache creates a new proposer indices cache for storing/accessing proposer index assignments of an epoch.
func NewProposerIndicesCache() *FakeProposerIndicesCache {
	return &FakeProposerIndicesCache{}
}

// AddProposerIndices adds ProposerIndices object to the cache.
// This method also trims the least recently list if the cache size has ready the max cache size limit.
func (c *FakeProposerIndicesCache) AddProposerIndices(p *ProposerIndices) error {
	return nil
}

// ProposerIndices returns the proposer indices of a block root seed.
func (c *FakeProposerIndicesCache) ProposerIndices(r [32]byte) ([]types.ValidatorIndex, error) {
	return nil, nil
}

// HasProposerIndices returns the proposer indices of a block root seed.
func (c *FakeProposerIndicesCache) HasProposerIndices(r [32]byte) (bool, error) {
	return false, nil
}
