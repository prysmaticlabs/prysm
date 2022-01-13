package v3

import (
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// currentSyncCommittee of the current sync committee in beacon chain state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) currentSyncCommittee() *ethpb.SyncCommittee {
	if !b.hasInnerState() {
		return nil
	}

	return CopySyncCommittee(b.state.CurrentSyncCommittee)
}

// nextSyncCommittee of the next sync committee in beacon chain state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) nextSyncCommittee() *ethpb.SyncCommittee {
	if !b.hasInnerState() {
		return nil
	}

	return CopySyncCommittee(b.state.NextSyncCommittee)
}

// CurrentSyncCommittee of the current sync committee in beacon chain state.
func (b *BeaconState) CurrentSyncCommittee() (*ethpb.SyncCommittee, error) {
	if !b.hasInnerState() {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.state.CurrentSyncCommittee == nil {
		return nil, nil
	}

	return b.currentSyncCommittee(), nil
}

// NextSyncCommittee of the next sync committee in beacon chain state.
func (b *BeaconState) NextSyncCommittee() (*ethpb.SyncCommittee, error) {
	if !b.hasInnerState() {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.state.NextSyncCommittee == nil {
		return nil, nil
	}

	return b.nextSyncCommittee(), nil
}

// CopySyncCommittee copies the provided sync committee object.
func CopySyncCommittee(data *ethpb.SyncCommittee) *ethpb.SyncCommittee {
	if data == nil {
		return nil
	}
	return &ethpb.SyncCommittee{
		Pubkeys:         bytesutil.SafeCopy2dBytes(data.Pubkeys),
		AggregatePubkey: bytesutil.SafeCopyBytes(data.AggregatePubkey),
	}
}
