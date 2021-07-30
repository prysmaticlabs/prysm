package v1

import (
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// AppendCurrentParticipationBits is not supported for phase 0 beacon state.
func (b *BeaconState) AppendCurrentParticipationBits(val byte) error {
	return errors.New("AppendCurrentParticipationBits is not supported for phase 0 beacon state")
}

// AppendPreviousParticipationBits is not supported for phase 0 beacon state.
func (b *BeaconState) AppendPreviousParticipationBits(val byte) error {
	return errors.New("AppendPreviousParticipationBits is not supported for phase 0 beacon state")
}

// AppendInactivityScore is not supported for phase 0 beacon state.
func (b *BeaconState) AppendInactivityScore(s uint64) error {
	return errors.New("AppendInactivityScore is not supported for phase 0 beacon state")
}

// SetCurrentSyncCommittee is not supported for phase 0 beacon state.
func (b *BeaconState) SetCurrentSyncCommittee(val *ethpb.SyncCommittee) error {
	return errors.New("SetCurrentSyncCommittee is not supported for phase 0 beacon state")
}

// SetNextSyncCommittee is not supported for phase 0 beacon state.
func (b *BeaconState) SetNextSyncCommittee(val *ethpb.SyncCommittee) error {
	return errors.New("SetNextSyncCommittee is not supported for phase 0 beacon state")
}

// SetPreviousParticipationBits is not supported for phase 0 beacon state.
func (b *BeaconState) SetPreviousParticipationBits(val []byte) error {
	return errors.New("SetPreviousParticipationBits is not supported for phase 0 beacon state")
}

// SetCurrentParticipationBits is not supported for phase 0 beacon state.
func (b *BeaconState) SetCurrentParticipationBits(val []byte) error {
	return errors.New("SetCurrentParticipationBits is not supported for phase 0 beacon state")
}

// SetInactivityScores is not supported for phase 0 beacon state.
func (b *BeaconState) SetInactivityScores(val []uint64) error {
	return errors.New("SetInactivityScores is not supported for phase 0 beacon state")
}
