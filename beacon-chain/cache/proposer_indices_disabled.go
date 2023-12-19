//go:build fuzz

// This file is used in fuzzer builds to bypass proposer indices caches.
package cache

import (
	forkchoicetypes "github.com/prysmaticlabs/prysm/v4/beacon-chain/forkchoice/types"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
)

// FakeProposerIndicesCache is a struct with 1 queue for looking up proposer indices by root.
type FakeProposerIndicesCache struct {
}

// NewProposerIndicesCache creates a new proposer indices cache for storing/accessing proposer index assignments of an epoch.
func NewProposerIndicesCache() *FakeProposerIndicesCache {
	return &FakeProposerIndicesCache{}
}

// ProposerIndices is a stub.
func (c *FakeProposerIndicesCache) ProposerIndices(_ primitives.Epoch, _ [32]byte) ([fieldparams.SlotsPerEpoch]primitives.ValidatorIndex, bool) {
	return [fieldparams.SlotsPerEpoch]primitives.ValidatorIndex{}, false
}

// UnsafeProposerIndices is a stub.
func (c *FakeProposerIndicesCache) UnsafeProposerIndices(_ primitives.Epoch, _ [32]byte) ([fieldparams.SlotsPerEpoch]primitives.ValidatorIndex, bool) {
	return [fieldparams.SlotsPerEpoch]primitives.ValidatorIndex{}, false
}

// Prune is a stub.
func (p *FakeProposerIndicesCache) Prune(epoch primitives.Epoch) {}

// Set is a stub.
func (p *FakeProposerIndicesCache) Set(epoch primitives.Epoch, root [32]byte, indices [fieldparams.SlotsPerEpoch]primitives.ValidatorIndex) {
}

// SetUnsafe is a stub.
func (p *FakeProposerIndicesCache) SetUnsafe(epoch primitives.Epoch, root [32]byte, indices [fieldparams.SlotsPerEpoch]primitives.ValidatorIndex) {
}

// SetCheckpoint is a stub.
func (p *FakeProposerIndicesCache) SetCheckpoint(c forkchoicetypes.Checkpoint, root [32]byte) {}

// IndicesFromCheckpoint is a stub.
func (p *FakeProposerIndicesCache) IndicesFromCheckpoint(_ forkchoicetypes.Checkpoint) ([fieldparams.SlotsPerEpoch]primitives.ValidatorIndex, bool) {
	return [fieldparams.SlotsPerEpoch]primitives.ValidatorIndex{}, false
}
