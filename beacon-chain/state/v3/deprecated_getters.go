package v3

import (
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// PreviousEpochAttestations is not supported for HF1 beacon state.
func (*BeaconState) PreviousEpochAttestations() ([]*ethpb.PendingAttestation, error) {
	return nil, errors.New("PreviousEpochAttestations is not supported for version Bellatrix beacon state")
}

// CurrentEpochAttestations is not supported for HF1 beacon state.
func (*BeaconState) CurrentEpochAttestations() ([]*ethpb.PendingAttestation, error) {
	return nil, errors.New("CurrentEpochAttestations is not supported for version Bellatrix beacon state")
}
