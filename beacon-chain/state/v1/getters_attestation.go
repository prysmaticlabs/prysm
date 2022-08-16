package v1

import (
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// PreviousEpochAttestations corresponding to blocks on the beacon chain.
func (b *BeaconState) PreviousEpochAttestations() ([]*ethpb.PendingAttestation, error) {
	if !b.hasInnerState() {
		return nil, ErrNilInnerState
	}
	if b.state.PreviousEpochAttestations == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.previousEpochAttestations(), nil
}

// previousEpochAttestations corresponding to blocks on the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) previousEpochAttestations() []*ethpb.PendingAttestation {
	if !b.hasInnerState() {
		return nil
	}

	return ethpb.CopyPendingAttestationSlice(b.state.PreviousEpochAttestations)
}

// CurrentEpochAttestations corresponding to blocks on the beacon chain.
func (b *BeaconState) CurrentEpochAttestations() ([]*ethpb.PendingAttestation, error) {
	if !b.hasInnerState() {
		return nil, ErrNilInnerState
	}
	if b.state.CurrentEpochAttestations == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.currentEpochAttestations(), nil
}

// currentEpochAttestations corresponding to blocks on the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) currentEpochAttestations() []*ethpb.PendingAttestation {
	if !b.hasInnerState() {
		return nil
	}

	return ethpb.CopyPendingAttestationSlice(b.state.CurrentEpochAttestations)
}
