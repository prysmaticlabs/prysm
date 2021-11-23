package v3

import (
	"github.com/pkg/errors"
	v1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// SetPreviousEpochAttestations is not supported for HF1 beacon state.
func (b *BeaconState) SetPreviousEpochAttestations(val []*ethpb.PendingAttestation) error {
	return errors.New("SetPreviousEpochAttestations is not supported for version Merge beacon state")
}

// SetCurrentEpochAttestations is not supported for HF1 beacon state.
func (b *BeaconState) SetCurrentEpochAttestations(val []*ethpb.PendingAttestation) error {
	return errors.New("SetCurrentEpochAttestations is not supported for version Merge beacon state")
}

// AppendCurrentEpochAttestations is not supported for HF1 beacon state.
func (b *BeaconState) AppendCurrentEpochAttestations(val *ethpb.PendingAttestation) error {
	return errors.New("AppendCurrentEpochAttestations is not supported for version Merge beacon state")
}

// AppendPreviousEpochAttestations is not supported for HF1 beacon state.
func (b *BeaconState) AppendPreviousEpochAttestations(val *ethpb.PendingAttestation) error {
	return errors.New("AppendPreviousEpochAttestations is not supported for version Merge beacon state")
}

// RotateAttestations is not supported for HF1 beacon state.
func (b *BeaconState) RotateAttestations() error {
	return errors.New("RotateAttestations is not supported for version Merge beacon state")
}

// ToProto is not supported for HF1 beacon state.
func (b *BeaconState) ToProto() (*v1.BeaconState, error) {
	return nil, errors.New("ToProto is not yet supported for Merge beacon state")
}
