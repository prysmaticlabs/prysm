package v2

import (
	"github.com/pkg/errors"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
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
func (*BeaconState) LatestExecutionPayloadHeader() (*enginev1.ExecutionPayloadHeader, error) {
	return nil, errors.New("LatestExecutionPayloadHeader is not supported for hard fork 1 beacon state")
}
