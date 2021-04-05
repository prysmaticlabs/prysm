package stateV0

import (
	"github.com/pkg/errors"
)

// CurrentEpochParticipation is not supported for HF1 beacon state.
func (b *BeaconState) CurrentEpochParticipation() ([]byte, error) {
	return nil, errors.New("CurrentEpochParticipation is not supported for hard fork 1 beacon state")
}

// PreviousEpochParticipation is not supported for HF1 beacon state.
func (b *BeaconState) PreviousEpochParticipation() ([]byte, error) {
	return nil, errors.New("PreviousEpochParticipation is not supported for hard fork 1 beacon state")
}

// InactivityScores is not supported for HF1 beacon state.
func (b *BeaconState) InactivityScores() ([]uint64, error) {
	return nil, errors.New("InactivityScores is not supported for hard fork 1 beacon state")
}
