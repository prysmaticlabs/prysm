package state_native

import (
	nativetypes "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/types"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// SetCurrentSyncCommittee for the beacon state.
func (b *BeaconState) SetCurrentSyncCommittee(val *ethpb.SyncCommittee) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.currentSyncCommittee = val
	b.markFieldAsDirty(nativetypes.CurrentSyncCommittee)
	return nil
}

// SetNextSyncCommittee for the beacon state.
func (b *BeaconState) SetNextSyncCommittee(val *ethpb.SyncCommittee) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.nextSyncCommittee = val
	b.markFieldAsDirty(nativetypes.NextSyncCommittee)
	return nil
}
