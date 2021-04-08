package stateV0

import (
	"github.com/pkg/errors"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
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

// SetCurrentSyncCommittee is not supported for phase 0 beacon state.
func (b *BeaconState) SetCurrentSyncCommittee(val *pbp2p.SyncCommittee) error {
	return errors.New("SetCurrentSyncCommittee is not supported for phase 0 beacon state")
}

// SetPreviousParticipationBits is not supported for phase 0 beacon state.
func (b *BeaconState) SetPreviousParticipationBits(val []byte) error {
	return errors.New("SetPreviousParticipationBits is not supported for phase 0 beacon state")
}

// SetCurrentParticipationBits is not supported for phase 0 beacon state.
func (b *BeaconState) SetCurrentParticipationBits(val []byte) error {
	return errors.New("SetCurrentParticipationBits is not supported for phase 0 beacon state")
}
