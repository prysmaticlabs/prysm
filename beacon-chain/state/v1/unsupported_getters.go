package v1

import (
	"github.com/pkg/errors"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// CurrentEpochParticipation is not supported for phase 0 beacon state.
func (*BeaconState) CurrentEpochParticipation() ([]byte, error) {
	return nil, errors.New("CurrentEpochParticipation is not supported for phase 0 beacon state")
}

// PreviousEpochParticipation is not supported for phase 0 beacon state.
func (*BeaconState) PreviousEpochParticipation() ([]byte, error) {
	return nil, errors.New("PreviousEpochParticipation is not supported for phase 0 beacon state")
}

// UnrealizedCheckpointBalances is not supported for phase 0 beacon state.
func (*BeaconState) UnrealizedCheckpointBalances() (uint64, uint64, uint64, error) {
	return 0, 0, 0, errors.New("UnrealizedCheckpointBalances is not supported for phase0 beacon state")
}

// InactivityScores is not supported for phase 0 beacon state.
func (*BeaconState) InactivityScores() ([]uint64, error) {
	return nil, errors.New("InactivityScores is not supported for phase 0 beacon state")
}

// CurrentSyncCommittee is not supported for phase 0 beacon state.
func (*BeaconState) CurrentSyncCommittee() (*ethpb.SyncCommittee, error) {
	return nil, errors.New("CurrentSyncCommittee is not supported for phase 0 beacon state")
}

// NextSyncCommittee is not supported for phase 0 beacon state.
func (*BeaconState) NextSyncCommittee() (*ethpb.SyncCommittee, error) {
	return nil, errors.New("NextSyncCommittee is not supported for phase 0 beacon state")
}

// LatestExecutionPayloadHeader is not supported for phase 0 beacon state.
func (*BeaconState) LatestExecutionPayloadHeader() (*enginev1.ExecutionPayloadHeader, error) {
	return nil, errors.New("LatestExecutionPayloadHeader is not supported for phase 0 beacon state")
}
