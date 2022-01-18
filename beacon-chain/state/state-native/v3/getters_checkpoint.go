package v3

import (
	"bytes"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/go-bitfield"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// JustificationBits marking which epochs have been justified in the beacon chain.
func (b *BeaconState) JustificationBits() bitfield.Bitvector4 {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.JustificationBits == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.justificationBits()
}

// justificationBits marking which epochs have been justified in the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) justificationBits() bitfield.Bitvector4 {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.JustificationBits == nil {
		return nil
	}

	res := make([]byte, len(b.state.JustificationBits.Bytes()))
	copy(res, b.state.JustificationBits.Bytes())
	return res
}

// PreviousJustifiedCheckpoint denoting an epoch and block root.
func (b *BeaconState) PreviousJustifiedCheckpoint() *ethpb.Checkpoint {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.PreviousJustifiedCheckpoint == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.previousJustifiedCheckpoint()
}

// previousJustifiedCheckpoint denoting an epoch and block root.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) previousJustifiedCheckpoint() *ethpb.Checkpoint {
	if !b.hasInnerState() {
		return nil
	}

	return ethpb.CopyCheckpoint(b.state.PreviousJustifiedCheckpoint)
}

// CurrentJustifiedCheckpoint denoting an epoch and block root.
func (b *BeaconState) CurrentJustifiedCheckpoint() *ethpb.Checkpoint {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.CurrentJustifiedCheckpoint == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.currentJustifiedCheckpoint()
}

// currentJustifiedCheckpoint denoting an epoch and block root.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) currentJustifiedCheckpoint() *ethpb.Checkpoint {
	if !b.hasInnerState() {
		return nil
	}

	return ethpb.CopyCheckpoint(b.state.CurrentJustifiedCheckpoint)
}

// MatchCurrentJustifiedCheckpoint returns true if input justified checkpoint matches
// the current justified checkpoint in state.
func (b *BeaconState) MatchCurrentJustifiedCheckpoint(c *ethpb.Checkpoint) bool {
	if !b.hasInnerState() {
		return false
	}
	if b.state.CurrentJustifiedCheckpoint == nil {
		return false
	}

	if c.Epoch != b.state.CurrentJustifiedCheckpoint.Epoch {
		return false
	}
	return bytes.Equal(c.Root, b.state.CurrentJustifiedCheckpoint.Root)
}

// MatchPreviousJustifiedCheckpoint returns true if the input justified checkpoint matches
// the previous justified checkpoint in state.
func (b *BeaconState) MatchPreviousJustifiedCheckpoint(c *ethpb.Checkpoint) bool {
	if !b.hasInnerState() {
		return false
	}
	if b.state.PreviousJustifiedCheckpoint == nil {
		return false
	}

	if c.Epoch != b.state.PreviousJustifiedCheckpoint.Epoch {
		return false
	}
	return bytes.Equal(c.Root, b.state.PreviousJustifiedCheckpoint.Root)
}

// FinalizedCheckpoint denoting an epoch and block root.
func (b *BeaconState) FinalizedCheckpoint() *ethpb.Checkpoint {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.FinalizedCheckpoint == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.finalizedCheckpoint()
}

// finalizedCheckpoint denoting an epoch and block root.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) finalizedCheckpoint() *ethpb.Checkpoint {
	if !b.hasInnerState() {
		return nil
	}

	return ethpb.CopyCheckpoint(b.state.FinalizedCheckpoint)
}

// FinalizedCheckpointEpoch returns the epoch value of the finalized checkpoint.
func (b *BeaconState) FinalizedCheckpointEpoch() types.Epoch {
	if !b.hasInnerState() {
		return 0
	}
	if b.state.FinalizedCheckpoint == nil {
		return 0
	}
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.state.FinalizedCheckpoint.Epoch
}
