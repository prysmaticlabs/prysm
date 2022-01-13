package v3

import (
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// SetPreviousEpochAttestations is not supported for HF1 beacon state.
func (*BeaconState) SetPreviousEpochAttestations(_ []*ethpb.PendingAttestation) error {
	return errors.New("SetPreviousEpochAttestations is not supported for version Bellatrix beacon state")
}

// SetCurrentEpochAttestations is not supported for HF1 beacon state.
func (*BeaconState) SetCurrentEpochAttestations(_ []*ethpb.PendingAttestation) error {
	return errors.New("SetCurrentEpochAttestations is not supported for version Bellatrix beacon state")
}

// AppendCurrentEpochAttestations is not supported for HF1 beacon state.
func (*BeaconState) AppendCurrentEpochAttestations(_ *ethpb.PendingAttestation) error {
	return errors.New("AppendCurrentEpochAttestations is not supported for version Bellatrix beacon state")
}

// AppendPreviousEpochAttestations is not supported for HF1 beacon state.
func (*BeaconState) AppendPreviousEpochAttestations(_ *ethpb.PendingAttestation) error {
	return errors.New("AppendPreviousEpochAttestations is not supported for version Bellatrix beacon state")
}

// RotateAttestations is not supported for HF1 beacon state.
func (*BeaconState) RotateAttestations() error {
	return errors.New("RotateAttestations is not supported for version Bellatrix beacon state")
}
