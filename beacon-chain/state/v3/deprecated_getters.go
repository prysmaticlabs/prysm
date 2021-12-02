package v3

import (
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// PreviousEpochAttestations is not supported for HF1 beacon state.
func (b *BeaconState) PreviousEpochAttestations() ([]*ethpb.PendingAttestation, error) {
	return nil, errors.New("PreviousEpochAttestations is not supported for version Merge beacon state")
}

// CurrentEpochAttestations is not supported for HF1 beacon state.
func (b *BeaconState) CurrentEpochAttestations() ([]*ethpb.PendingAttestation, error) {
	return nil, errors.New("CurrentEpochAttestations is not supported for version Merge beacon state")
}
