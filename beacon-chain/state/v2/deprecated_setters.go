package v2

import (
	"github.com/pkg/errors"
	v1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	statepb "github.com/prysmaticlabs/prysm/proto/prysm/v2/state"
)

// SetPreviousEpochAttestations is not supported for HF1 beacon state.
func (b *BeaconState) SetPreviousEpochAttestations(val []*statepb.PendingAttestation) error {
	return errors.New("SetPreviousEpochAttestations is not supported for hard fork 1 beacon state")
}

// SetCurrentEpochAttestations is not supported for HF1 beacon state.
func (b *BeaconState) SetCurrentEpochAttestations(val []*statepb.PendingAttestation) error {
	return errors.New("SetCurrentEpochAttestations is not supported for hard fork 1 beacon state")
}

// AppendCurrentEpochAttestations is not supported for HF1 beacon state.
func (b *BeaconState) AppendCurrentEpochAttestations(val *statepb.PendingAttestation) error {
	return errors.New("AppendCurrentEpochAttestations is not supported for hard fork 1 beacon state")
}

// AppendPreviousEpochAttestations is not supported for HF1 beacon state.
func (b *BeaconState) AppendPreviousEpochAttestations(val *statepb.PendingAttestation) error {
	return errors.New("AppendPreviousEpochAttestations is not supported for hard fork 1 beacon state")
}

// RotateAttestations is not supported for HF1 beacon state.
func (b *BeaconState) RotateAttestations() error {
	return errors.New("RotateAttestations is not supported for hard fork 1 beacon state")
}

// ToProto is not supported for HF1 beacon state.
func (b *BeaconState) ToProto() (*v1.BeaconState, error) {
	return nil, errors.New("ToProto is not yet supported for hard fork 1 beacon state")
}
