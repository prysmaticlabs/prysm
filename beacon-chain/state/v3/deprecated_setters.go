package v3

import (
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// AppendCurrentEpochAttestations is not supported for HF1 beacon state.
func (*BeaconState) AppendCurrentEpochAttestations(_ *ethpb.PendingAttestation) error {
	return errors.New("AppendCurrentEpochAttestations is not supported for Bellatrix beacon state")
}

// AppendPreviousEpochAttestations is not supported for HF1 beacon state.
func (*BeaconState) AppendPreviousEpochAttestations(_ *ethpb.PendingAttestation) error {
	return errors.New("AppendPreviousEpochAttestations is not supported for version Bellatrix beacon state")
}

// RotateAttestations is not supported for HF1 beacon state.
func (*BeaconState) RotateAttestations() error {
	return errors.New("RotateAttestations is not supported for version Bellatrix beacon state")
}
