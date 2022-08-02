package v2

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// PreviousEpochAttestations is not supported for HF1 beacon state.
func (*BeaconState) PreviousEpochAttestations() ([]*ethpb.PendingAttestation, error) {
	return nil, errors.New("PreviousEpochAttestations is not supported for hard fork 1 beacon state")
}

// CurrentEpochAttestations is not supported for HF1 beacon state.
func (*BeaconState) CurrentEpochAttestations() ([]*ethpb.PendingAttestation, error) {
	return nil, errors.New("CurrentEpochAttestations is not supported for hard fork 1 beacon state")
}

// LatestExecutionPayloadHeader is not supported for hard fork 1 beacon state.
func (*BeaconState) LatestExecutionPayloadHeader() (interfaces.ExecutionData, error) {
	return nil, errors.New("LatestExecutionPayloadHeader is not supported for hard fork 1 beacon state")
}
