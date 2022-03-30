package v0

import (
	"github.com/prysmaticlabs/go-bitfield"
	v0types "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/v0/types"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// SetJustificationBits for the beacon state.
func (b *BeaconState) SetJustificationBits(val bitfield.Bitvector4) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.justificationBits = val
	b.markFieldAsDirty(v0types.JustificationBits)
	return nil
}

// SetPreviousJustifiedCheckpoint for the beacon state.
func (b *BeaconState) SetPreviousJustifiedCheckpoint(val *ethpb.Checkpoint) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.previousJustifiedCheckpoint = val
	b.markFieldAsDirty(v0types.PreviousJustifiedCheckpoint)
	return nil
}

// SetCurrentJustifiedCheckpoint for the beacon state.
func (b *BeaconState) SetCurrentJustifiedCheckpoint(val *ethpb.Checkpoint) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.currentJustifiedCheckpoint = val
	b.markFieldAsDirty(v0types.CurrentJustifiedCheckpoint)
	return nil
}

// SetFinalizedCheckpoint for the beacon state.
func (b *BeaconState) SetFinalizedCheckpoint(val *ethpb.Checkpoint) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.finalizedCheckpoint = val
	b.markFieldAsDirty(v0types.FinalizedCheckpoint)
	return nil
}
