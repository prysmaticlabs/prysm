package state_native

import (
	"fmt"

	nativetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stateutil"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
)

// RotateAttestations sets the previous epoch attestations to the current epoch attestations and
// then clears the current epoch attestations.
func (b *BeaconState) RotateAttestations() error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.version != version.Phase0 {
		return errNotSupported("RotateAttestations", b.version)
	}

	b.setPreviousEpochAttestations(b.currentEpochAttestationsVal())
	b.setCurrentEpochAttestations([]*ethpb.PendingAttestation{})
	return nil
}

func (b *BeaconState) setPreviousEpochAttestations(val []*ethpb.PendingAttestation) {
	b.sharedFieldReferences[nativetypes.PreviousEpochAttestations].MinusRef()
	b.sharedFieldReferences[nativetypes.PreviousEpochAttestations] = stateutil.NewRef(1)

	b.previousEpochAttestations = val
	b.markFieldAsDirty(nativetypes.PreviousEpochAttestations)
	b.rebuildTrie[nativetypes.PreviousEpochAttestations] = true
}

func (b *BeaconState) setCurrentEpochAttestations(val []*ethpb.PendingAttestation) {
	b.sharedFieldReferences[nativetypes.CurrentEpochAttestations].MinusRef()
	b.sharedFieldReferences[nativetypes.CurrentEpochAttestations] = stateutil.NewRef(1)

	b.currentEpochAttestations = val
	b.markFieldAsDirty(nativetypes.CurrentEpochAttestations)
	b.rebuildTrie[nativetypes.CurrentEpochAttestations] = true
}

// AppendCurrentEpochAttestations for the beacon state. Appends the new value
// to the the end of list.
func (b *BeaconState) AppendCurrentEpochAttestations(val *ethpb.PendingAttestation) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.version != version.Phase0 {
		return errNotSupported("AppendCurrentEpochAttestations", b.version)
	}

	atts := b.currentEpochAttestations
	max := uint64(fieldparams.CurrentEpochAttestationsLength)
	if uint64(len(atts)) >= max {
		return fmt.Errorf("current pending attestation exceeds max length %d", max)
	}

	if b.sharedFieldReferences[nativetypes.CurrentEpochAttestations].Refs() > 1 {
		// Copy elements in underlying array by reference.
		atts = make([]*ethpb.PendingAttestation, len(b.currentEpochAttestations))
		copy(atts, b.currentEpochAttestations)
		b.sharedFieldReferences[nativetypes.CurrentEpochAttestations].MinusRef()
		b.sharedFieldReferences[nativetypes.CurrentEpochAttestations] = stateutil.NewRef(1)
	}

	b.currentEpochAttestations = append(atts, val)
	b.markFieldAsDirty(nativetypes.CurrentEpochAttestations)
	b.addDirtyIndices(nativetypes.CurrentEpochAttestations, []uint64{uint64(len(b.currentEpochAttestations) - 1)})
	return nil
}

// AppendPreviousEpochAttestations for the beacon state. Appends the new value
// to the the end of list.
func (b *BeaconState) AppendPreviousEpochAttestations(val *ethpb.PendingAttestation) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.version != version.Phase0 {
		return errNotSupported("AppendPreviousEpochAttestations", b.version)
	}

	atts := b.previousEpochAttestations
	max := uint64(fieldparams.PreviousEpochAttestationsLength)
	if uint64(len(atts)) >= max {
		return fmt.Errorf("previous pending attestation exceeds max length %d", max)
	}

	if b.sharedFieldReferences[nativetypes.PreviousEpochAttestations].Refs() > 1 {
		atts = make([]*ethpb.PendingAttestation, len(b.previousEpochAttestations))
		copy(atts, b.previousEpochAttestations)
		b.sharedFieldReferences[nativetypes.PreviousEpochAttestations].MinusRef()
		b.sharedFieldReferences[nativetypes.PreviousEpochAttestations] = stateutil.NewRef(1)
	}

	b.previousEpochAttestations = append(atts, val)
	b.markFieldAsDirty(nativetypes.PreviousEpochAttestations)
	b.addDirtyIndices(nativetypes.PreviousEpochAttestations, []uint64{uint64(len(b.previousEpochAttestations) - 1)})
	return nil
}
