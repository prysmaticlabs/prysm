package v3

import (
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// SetCurrentSyncCommittee for the beacon state.
func (b *BeaconState) SetCurrentSyncCommittee(val *ethpb.SyncCommittee) error {
	if !b.hasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.state.CurrentSyncCommittee = val
	b.markFieldAsDirty(currentSyncCommittee)
	return nil
}

// SetNextSyncCommittee for the beacon state.
func (b *BeaconState) SetNextSyncCommittee(val *ethpb.SyncCommittee) error {
	if !b.hasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.state.NextSyncCommittee = val
	b.markFieldAsDirty(nextSyncCommittee)
	return nil
}
