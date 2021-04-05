package stateV0

import (
	"github.com/pkg/errors"
)

// AppendCurrentParticipationBits is not supported for phase 0 beacon state.
func (b *BeaconState) AppendCurrentParticipationBits(val byte) error {
	return errors.New("AppendCurrentParticipationBits is not supported for phase 0 beacon state")
}

// AppendPreviousParticipationBits is not supported for phase 0 beacon state.
func (b *BeaconState) AppendPreviousParticipationBits(val byte) error {
	return errors.New("AppendPreviousParticipationBits is not supported for phase 0 beacon state")
}

// AppendInactivityScore is not supported for phase 0 beacon state.
func (b *BeaconState) AppendInactivityScore(s uint64) error {
	return errors.New("AppendInactivityScore is not supported for phase 0 beacon state")
}
