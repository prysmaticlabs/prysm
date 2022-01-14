package v1

import (
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// AppendCurrentParticipationBits is not supported for phase 0 beacon state.
func (*BeaconState) AppendCurrentParticipationBits(_ byte) error {
	return errors.New("AppendCurrentParticipationBits is not supported for phase 0 beacon state")
}

// AppendPreviousParticipationBits is not supported for phase 0 beacon state.
func (*BeaconState) AppendPreviousParticipationBits(_ byte) error {
	return errors.New("AppendPreviousParticipationBits is not supported for phase 0 beacon state")
}

// AppendInactivityScore is not supported for phase 0 beacon state.
func (*BeaconState) AppendInactivityScore(_ uint64) error {
	return errors.New("AppendInactivityScore is not supported for phase 0 beacon state")
}

// SetCurrentSyncCommittee is not supported for phase 0 beacon state.
func (*BeaconState) SetCurrentSyncCommittee(_ *ethpb.SyncCommittee) error {
	return errors.New("SetCurrentSyncCommittee is not supported for phase 0 beacon state")
}

// SetNextSyncCommittee is not supported for phase 0 beacon state.
func (*BeaconState) SetNextSyncCommittee(_ *ethpb.SyncCommittee) error {
	return errors.New("SetNextSyncCommittee is not supported for phase 0 beacon state")
}

// SetPreviousParticipationBits is not supported for phase 0 beacon state.
func (*BeaconState) SetPreviousParticipationBits(_ []byte) error {
	return errors.New("SetPreviousParticipationBits is not supported for phase 0 beacon state")
}

// SetCurrentParticipationBits is not supported for phase 0 beacon state.
func (*BeaconState) SetCurrentParticipationBits(_ []byte) error {
	return errors.New("SetCurrentParticipationBits is not supported for phase 0 beacon state")
}

// SetInactivityScores is not supported for phase 0 beacon state.
func (*BeaconState) SetInactivityScores(_ []uint64) error {
	return errors.New("SetInactivityScores is not supported for phase 0 beacon state")
}

// SetLatestExecutionPayloadHeader is not supported for phase 0 beacon state.
func (*BeaconState) SetLatestExecutionPayloadHeader(val *ethpb.ExecutionPayloadHeader) error {
	return errors.New("SetLatestExecutionPayloadHeader is not supported for phase 0 beacon state")
}
