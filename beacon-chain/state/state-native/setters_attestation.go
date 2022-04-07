package state_native

import (
	"fmt"

	v0types "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// RotateAttestations sets the previous epoch attestations to the current epoch attestations and
// then clears the current epoch attestations.
func (b *BeaconState) RotateAttestations() error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.setPreviousEpochAttestations(b.currentEpochAttestationsVal())
	b.setCurrentEpochAttestations([]*ethpb.PendingAttestation{})
	return nil
}

func (b *BeaconState) setPreviousEpochAttestations(val []*ethpb.PendingAttestation) {
	b.sharedFieldReferences[v0types.PreviousEpochAttestations].MinusRef()
	b.sharedFieldReferences[v0types.PreviousEpochAttestations] = stateutil.NewRef(1)

	b.previousEpochAttestations = val
	b.markFieldAsDirty(v0types.PreviousEpochAttestations)
	b.rebuildTrie[v0types.PreviousEpochAttestations] = true
}

func (b *BeaconState) setCurrentEpochAttestations(val []*ethpb.PendingAttestation) {
	b.sharedFieldReferences[v0types.CurrentEpochAttestations].MinusRef()
	b.sharedFieldReferences[v0types.CurrentEpochAttestations] = stateutil.NewRef(1)

	b.currentEpochAttestations = val
	b.markFieldAsDirty(v0types.PreviousEpochAttestations)
	b.rebuildTrie[v0types.CurrentEpochAttestations] = true
}

// AppendCurrentEpochAttestations for the beacon state. Appends the new value
// to the the end of list.
func (b *BeaconState) AppendCurrentEpochAttestations(val *ethpb.PendingAttestation) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	atts := b.currentEpochAttestations
	max := uint64(fieldparams.CurrentEpochAttestationsLength)
	if uint64(len(atts)) >= max {
		return fmt.Errorf("current pending attestation exceeds max length %d", max)
	}

	if b.sharedFieldReferences[v0types.CurrentEpochAttestations].Refs() > 1 {
		// Copy elements in underlying array by reference.
		atts = make([]*ethpb.PendingAttestation, len(b.currentEpochAttestations))
		copy(atts, b.currentEpochAttestations)
		b.sharedFieldReferences[v0types.CurrentEpochAttestations].MinusRef()
		b.sharedFieldReferences[v0types.CurrentEpochAttestations] = stateutil.NewRef(1)
	}

	b.currentEpochAttestations = append(atts, val)
	b.markFieldAsDirty(v0types.CurrentEpochAttestations)
	b.addDirtyIndices(v0types.CurrentEpochAttestations, []uint64{uint64(len(b.currentEpochAttestations) - 1)})
	return nil
}

// AppendPreviousEpochAttestations for the beacon state. Appends the new value
// to the the end of list.
func (b *BeaconState) AppendPreviousEpochAttestations(val *ethpb.PendingAttestation) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	atts := b.previousEpochAttestations
	max := uint64(fieldparams.PreviousEpochAttestationsLength)
	if uint64(len(atts)) >= max {
		return fmt.Errorf("previous pending attestation exceeds max length %d", max)
	}

	if b.sharedFieldReferences[v0types.PreviousEpochAttestations].Refs() > 1 {
		atts = make([]*ethpb.PendingAttestation, len(b.previousEpochAttestations))
		copy(atts, b.previousEpochAttestations)
		b.sharedFieldReferences[v0types.PreviousEpochAttestations].MinusRef()
		b.sharedFieldReferences[v0types.PreviousEpochAttestations] = stateutil.NewRef(1)
	}

	b.previousEpochAttestations = append(atts, val)
	b.markFieldAsDirty(v0types.PreviousEpochAttestations)
	b.addDirtyIndices(v0types.PreviousEpochAttestations, []uint64{uint64(len(b.previousEpochAttestations) - 1)})
	return nil
}
