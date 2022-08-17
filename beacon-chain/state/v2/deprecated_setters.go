package v2

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// AppendCurrentEpochAttestations is not supported for HF1 beacon state.
func (*BeaconState) AppendCurrentEpochAttestations(_ *ethpb.PendingAttestation) error {
	return errors.New("AppendCurrentEpochAttestations is not supported for hard fork 1 beacon state")
}

// AppendPreviousEpochAttestations is not supported for HF1 beacon state.
func (*BeaconState) AppendPreviousEpochAttestations(_ *ethpb.PendingAttestation) error {
	return errors.New("AppendPreviousEpochAttestations is not supported for hard fork 1 beacon state")
}

// RotateAttestations is not supported for HF1 beacon state.
func (*BeaconState) RotateAttestations() error {
	return errors.New("RotateAttestations is not supported for hard fork 1 beacon state")
}

// SetLatestExecutionPayloadHeader is not supported for hard fork 1 beacon state.
func (*BeaconState) SetLatestExecutionPayloadHeader(_ interfaces.ExecutionData) error {
	return errors.New("SetLatestExecutionPayloadHeader is not supported for hard fork 1 beacon state")
}
