package iface

import statepb "github.com/prysmaticlabs/prysm/proto/prysm/v2/state"

// BeaconStateAltair has read and write access to beacon state methods.
type BeaconStateAltair interface {
	BeaconState
	CurrentSyncCommittee() (*statepb.SyncCommittee, error)
	NextSyncCommittee() (*statepb.SyncCommittee, error)
	SetCurrentSyncCommittee(val *statepb.SyncCommittee) error
	SetNextSyncCommittee(val *statepb.SyncCommittee) error
	CurrentEpochParticipation() ([]byte, error)
	PreviousEpochParticipation() ([]byte, error)
	InactivityScores() ([]uint64, error)
	AppendCurrentParticipationBits(val byte) error
	AppendPreviousParticipationBits(val byte) error
	AppendInactivityScore(s uint64) error
	SetInactivityScores(val []uint64) error
	SetPreviousParticipationBits(val []byte) error
	SetCurrentParticipationBits(val []byte) error
}
