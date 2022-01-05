package v1

import (
	"bytes"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/go-bitfield"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// JustificationBits marking which epochs have been justified in the beacon chain.
func (b *BeaconState) JustificationBits() bitfield.Bitvector4 {
	if b.justificationBits == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.justificationBitsInternal()
}

// justificationBitsInternal marking which epochs have been justified in the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) justificationBitsInternal() bitfield.Bitvector4 {
	if b.justificationBits == nil {
		return nil
	}

	res := make([]byte, len(b.justificationBits.Bytes()))
	copy(res, b.justificationBits.Bytes())
	return res
}

// PreviousJustifiedCheckpoint denoting an epoch and block root.
func (b *BeaconState) PreviousJustifiedCheckpoint() *ethpb.Checkpoint {
	if b.previousJustifiedCheckpoint == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.previousJustifiedCheckpointInternal()
}

// previousJustifiedCheckpointInternal denoting an epoch and block root.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) previousJustifiedCheckpointInternal() *ethpb.Checkpoint {
	return ethpb.CopyCheckpoint(b.previousJustifiedCheckpoint)
}

// CurrentJustifiedCheckpoint denoting an epoch and block root.
func (b *BeaconState) CurrentJustifiedCheckpoint() *ethpb.Checkpoint {
	if b.currentJustifiedCheckpoint == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.currentJustifiedCheckpointInternal()
}

// currentJustifiedCheckpointInternal denoting an epoch and block root.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) currentJustifiedCheckpointInternal() *ethpb.Checkpoint {
	return ethpb.CopyCheckpoint(b.currentJustifiedCheckpoint)
}

// MatchCurrentJustifiedCheckpoint returns true if input justified checkpoint matches
// the current justified checkpoint in state.
func (b *BeaconState) MatchCurrentJustifiedCheckpoint(c *ethpb.Checkpoint) bool {
	if b.currentJustifiedCheckpoint == nil {
		return false
	}

	if c.Epoch != b.currentJustifiedCheckpoint.Epoch {
		return false
	}
	return bytes.Equal(c.Root, b.currentJustifiedCheckpoint.Root)
}

// MatchPreviousJustifiedCheckpoint returns true if the input justified checkpoint matches
// the previous justified checkpoint in state.
func (b *BeaconState) MatchPreviousJustifiedCheckpoint(c *ethpb.Checkpoint) bool {
	if b.previousJustifiedCheckpoint == nil {
		return false
	}

	if c.Epoch != b.previousJustifiedCheckpoint.Epoch {
		return false
	}
	return bytes.Equal(c.Root, b.previousJustifiedCheckpoint.Root)
}

// FinalizedCheckpoint denoting an epoch and block root.
func (b *BeaconState) FinalizedCheckpoint() *ethpb.Checkpoint {
	if b.finalizedCheckpoint == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.finalizedCheckpointInternal()
}

// finalizedCheckpointInternal denoting an epoch and block root.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) finalizedCheckpointInternal() *ethpb.Checkpoint {
	return ethpb.CopyCheckpoint(b.finalizedCheckpoint)
}

// FinalizedCheckpointEpoch returns the epoch value of the finalized checkpoint.
func (b *BeaconState) FinalizedCheckpointEpoch() types.Epoch {
	if b.finalizedCheckpoint == nil {
		return 0
	}
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.finalizedCheckpoint.Epoch
}
