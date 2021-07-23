package v2

import (
	"github.com/pkg/errors"
	statepb "github.com/prysmaticlabs/prysm/proto/prysm/v2/state"
)

// PreviousEpochAttestations is not supported for HF1 beacon state.
func (b *BeaconState) PreviousEpochAttestations() ([]*statepb.PendingAttestation, error) {
	return nil, errors.New("PreviousEpochAttestations is not supported for hard fork 1 beacon state")
}

// CurrentEpochAttestations is not supported for HF1 beacon state.
func (b *BeaconState) CurrentEpochAttestations() ([]*statepb.PendingAttestation, error) {
	return nil, errors.New("CurrentEpochAttestations is not supported for hard fork 1 beacon state")
}
