package state

import (
	"bytes"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// JustificationBits marking which epochs have been justified in the beacon chain.
func (b *State) JustificationBits() bitfield.Bitvector4 {
	if b.justificationBits == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.justificationBitsVal()
}

// justificationBitsVal marking which epochs have been justified in the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *State) justificationBitsVal() bitfield.Bitvector4 {
	if b.justificationBits == nil {
		return nil
	}

	res := make([]byte, len(b.justificationBits.Bytes()))
	copy(res, b.justificationBits.Bytes())
	return res
}

// PreviousJustifiedCheckpoint denoting an epoch and block root.
func (b *State) PreviousJustifiedCheckpoint() *ethpb.Checkpoint {
	if b.previousJustifiedCheckpoint == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.previousJustifiedCheckpointVal()
}

// previousJustifiedCheckpointVal denoting an epoch and block root.
// This assumes that a lock is already held on BeaconState.
func (b *State) previousJustifiedCheckpointVal() *ethpb.Checkpoint {
	return ethpb.CopyCheckpoint(b.previousJustifiedCheckpoint)
}

// CurrentJustifiedCheckpoint denoting an epoch and block root.
func (b *State) CurrentJustifiedCheckpoint() *ethpb.Checkpoint {
	if b.currentJustifiedCheckpoint == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.currentJustifiedCheckpointVal()
}

// currentJustifiedCheckpointVal denoting an epoch and block root.
// This assumes that a lock is already held on BeaconState.
func (b *State) currentJustifiedCheckpointVal() *ethpb.Checkpoint {
	return ethpb.CopyCheckpoint(b.currentJustifiedCheckpoint)
}

// MatchCurrentJustifiedCheckpoint returns true if input justified checkpoint matches
// the current justified checkpoint in state.
func (b *State) MatchCurrentJustifiedCheckpoint(c *ethpb.Checkpoint) bool {
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
func (b *State) MatchPreviousJustifiedCheckpoint(c *ethpb.Checkpoint) bool {
	if b.previousJustifiedCheckpoint == nil {
		return false
	}

	if c.Epoch != b.previousJustifiedCheckpoint.Epoch {
		return false
	}
	return bytes.Equal(c.Root, b.previousJustifiedCheckpoint.Root)
}

// FinalizedCheckpoint denoting an epoch and block root.
func (b *State) FinalizedCheckpoint() *ethpb.Checkpoint {
	if b.finalizedCheckpoint == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.finalizedCheckpointVal()
}

// finalizedCheckpointVal denoting an epoch and block root.
// This assumes that a lock is already held on BeaconState.
func (b *State) finalizedCheckpointVal() *ethpb.Checkpoint {
	return ethpb.CopyCheckpoint(b.finalizedCheckpoint)
}

// FinalizedCheckpointEpoch returns the epoch value of the finalized checkpoint.
func (b *State) FinalizedCheckpointEpoch() primitives.Epoch {
	if b.finalizedCheckpoint == nil {
		return 0
	}
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.finalizedCheckpoint.Epoch
}
