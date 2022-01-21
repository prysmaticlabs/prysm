package v2

import (
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// currentSyncCommitteeInternal of the current sync committee in beacon chain state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) currentSyncCommitteeInternal() *ethpb.SyncCommittee {
	return CopySyncCommittee(b.currentSyncCommittee)
}

// nextSyncCommitteeInternal of the next sync committee in beacon chain state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) nextSyncCommitteeInternal() *ethpb.SyncCommittee {
	return CopySyncCommittee(b.nextSyncCommittee)
}

// CurrentSyncCommittee of the current sync committee in beacon chain state.
func (b *BeaconState) CurrentSyncCommittee() (*ethpb.SyncCommittee, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.currentSyncCommittee == nil {
		return nil, nil
	}

	return b.currentSyncCommitteeInternal(), nil
}

// NextSyncCommittee of the next sync committee in beacon chain state.
func (b *BeaconState) NextSyncCommittee() (*ethpb.SyncCommittee, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.nextSyncCommittee == nil {
		return nil, nil
	}

	return b.nextSyncCommitteeInternal(), nil
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
