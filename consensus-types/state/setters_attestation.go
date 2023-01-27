package state

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/state/types"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
)

// RotateAttestations sets the previous epoch attestations to the current epoch attestations and
// then clears the current epoch attestations.
func (b *State) RotateAttestations() error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.version != version.Phase0 {
		return errNotSupported("RotateAttestations", b.version)
	}

	b.setPreviousEpochAttestations(b.currentEpochAttestationsVal())
	b.setCurrentEpochAttestations([]*ethpb.PendingAttestation{})
	return nil
}

func (b *State) setPreviousEpochAttestations(val []*ethpb.PendingAttestation) {
	b.sharedFieldReferences[types.PreviousEpochAttestations].MinusRef()
	b.sharedFieldReferences[types.PreviousEpochAttestations] = stateutil.NewRef(1)

	b.previousEpochAttestations = val
	b.markFieldAsDirty(types.PreviousEpochAttestations)
	b.rebuildTrie[types.PreviousEpochAttestations] = true
}

func (b *State) setCurrentEpochAttestations(val []*ethpb.PendingAttestation) {
	b.sharedFieldReferences[types.CurrentEpochAttestations].MinusRef()
	b.sharedFieldReferences[types.CurrentEpochAttestations] = stateutil.NewRef(1)

	b.currentEpochAttestations = val
	b.markFieldAsDirty(types.CurrentEpochAttestations)
	b.rebuildTrie[types.CurrentEpochAttestations] = true
}

// AppendCurrentEpochAttestations for the beacon state. Appends the new value
// to the the end of list.
func (b *State) AppendCurrentEpochAttestations(val *ethpb.PendingAttestation) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.version != version.Phase0 {
		return errNotSupported("AppendCurrentEpochAttestations", b.version)
	}

	atts := b.currentEpochAttestations
	max := uint64(params.BeaconConfig().CurrentEpochAttestationsLength())
	if uint64(len(atts)) >= max {
		return fmt.Errorf("current pending attestation exceeds max length %d", max)
	}

	if b.sharedFieldReferences[types.CurrentEpochAttestations].Refs() > 1 {
		// Copy elements in underlying array by reference.
		atts = make([]*ethpb.PendingAttestation, len(b.currentEpochAttestations))
		copy(atts, b.currentEpochAttestations)
		b.sharedFieldReferences[types.CurrentEpochAttestations].MinusRef()
		b.sharedFieldReferences[types.CurrentEpochAttestations] = stateutil.NewRef(1)
	}

	b.currentEpochAttestations = append(atts, val)
	b.markFieldAsDirty(types.CurrentEpochAttestations)
	b.addDirtyIndices(types.CurrentEpochAttestations, []uint64{uint64(len(b.currentEpochAttestations) - 1)})
	return nil
}

// AppendPreviousEpochAttestations for the beacon state. Appends the new value
// to the the end of list.
func (b *State) AppendPreviousEpochAttestations(val *ethpb.PendingAttestation) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.version != version.Phase0 {
		return errNotSupported("AppendPreviousEpochAttestations", b.version)
	}

	atts := b.previousEpochAttestations
	max := uint64(params.BeaconConfig().PreviousEpochAttestationsLength())
	if uint64(len(atts)) >= max {
		return fmt.Errorf("previous pending attestation exceeds max length %d", max)
	}

	if b.sharedFieldReferences[types.PreviousEpochAttestations].Refs() > 1 {
		atts = make([]*ethpb.PendingAttestation, len(b.previousEpochAttestations))
		copy(atts, b.previousEpochAttestations)
		b.sharedFieldReferences[types.PreviousEpochAttestations].MinusRef()
		b.sharedFieldReferences[types.PreviousEpochAttestations] = stateutil.NewRef(1)
	}

	b.previousEpochAttestations = append(atts, val)
	b.markFieldAsDirty(types.PreviousEpochAttestations)
	b.addDirtyIndices(types.PreviousEpochAttestations, []uint64{uint64(len(b.previousEpochAttestations) - 1)})
	return nil
}

func (b *State) SetPreviousEpochAttestations(a []*ethpb.PendingAttestation) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.version != version.Phase0 {
		return errNotSupported("SetPreviousEpochAttestations", b.version)
	}
	b.setPreviousEpochAttestations(a)
	return nil
}

func (b *State) SetCurrentEpochAttestations(a []*ethpb.PendingAttestation) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.version != version.Phase0 {
		return errNotSupported("SetCurrentEpochAttestations", b.version)
	}
	b.setCurrentEpochAttestations(a)
	return nil
}
