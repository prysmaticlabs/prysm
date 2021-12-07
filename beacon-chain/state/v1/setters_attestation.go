package v1

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// RotateAttestations sets the previous epoch attestations to the current epoch attestations and
// then clears the current epoch attestations.
func (b *BeaconState) RotateAttestations() error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.setPreviousEpochAttestations(b.currentEpochAttestationsInternal())
	b.setCurrentEpochAttestations([]*ethpb.PendingAttestation{})
	return nil
}

func (b *BeaconState) setPreviousEpochAttestations(val []*ethpb.PendingAttestation) {
	b.sharedFieldReferences[previousEpochAttestations].MinusRef()
	b.sharedFieldReferences[previousEpochAttestations] = stateutil.NewRef(1)

	b.previousEpochAttestations = val
	b.markFieldAsDirty(previousEpochAttestations)
	b.rebuildTrie[previousEpochAttestations] = true
}

func (b *BeaconState) setCurrentEpochAttestations(val []*ethpb.PendingAttestation) {
	b.sharedFieldReferences[currentEpochAttestations].MinusRef()
	b.sharedFieldReferences[currentEpochAttestations] = stateutil.NewRef(1)

	b.currentEpochAttestations = val
	b.markFieldAsDirty(currentEpochAttestations)
	b.rebuildTrie[currentEpochAttestations] = true
}

// AppendCurrentEpochAttestations for the beacon state. Appends the new value
// to the the end of list.
func (b *BeaconState) AppendCurrentEpochAttestations(val *ethpb.PendingAttestation) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	atts := b.currentEpochAttestations
	max := uint64(params.BeaconConfig().SlotsPerEpoch) * params.BeaconConfig().MaxAttestations
	if uint64(len(atts)) >= max {
		return fmt.Errorf("current pending attestation exceeds max length %d", max)
	}

	if b.sharedFieldReferences[currentEpochAttestations].Refs() > 1 {
		// Copy elements in underlying array by reference.
		atts = make([]*ethpb.PendingAttestation, len(b.currentEpochAttestations))
		copy(atts, b.currentEpochAttestations)
		b.sharedFieldReferences[currentEpochAttestations].MinusRef()
		b.sharedFieldReferences[currentEpochAttestations] = stateutil.NewRef(1)
	}

	b.currentEpochAttestations = append(atts, val)
	b.markFieldAsDirty(currentEpochAttestations)
	b.addDirtyIndices(currentEpochAttestations, []uint64{uint64(len(b.currentEpochAttestations) - 1)})
	return nil
}

// AppendPreviousEpochAttestations for the beacon state. Appends the new value
// to the the end of list.
func (b *BeaconState) AppendPreviousEpochAttestations(val *ethpb.PendingAttestation) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	atts := b.previousEpochAttestations
	max := uint64(params.BeaconConfig().SlotsPerEpoch) * params.BeaconConfig().MaxAttestations
	if uint64(len(atts)) >= max {
		return fmt.Errorf("previous pending attestation exceeds max length %d", max)
	}

	if b.sharedFieldReferences[previousEpochAttestations].Refs() > 1 {
		atts = make([]*ethpb.PendingAttestation, len(b.previousEpochAttestations))
		copy(atts, b.previousEpochAttestations)
		b.sharedFieldReferences[previousEpochAttestations].MinusRef()
		b.sharedFieldReferences[previousEpochAttestations] = stateutil.NewRef(1)
	}

	b.previousEpochAttestations = append(atts, val)
	b.markFieldAsDirty(previousEpochAttestations)
	b.addDirtyIndices(previousEpochAttestations, []uint64{uint64(len(b.previousEpochAttestations) - 1)})
	return nil
}
