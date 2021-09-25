package state

import ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"

// BeaconStateAltair has read and write access to beacon state methods.
type BeaconStateAltair interface {
	BeaconState
	CurrentSyncCommittee() (*ethpb.SyncCommittee, error)
	NextSyncCommittee() (*ethpb.SyncCommittee, error)
	SetCurrentSyncCommittee(val *ethpb.SyncCommittee) error
	SetNextSyncCommittee(val *ethpb.SyncCommittee) error
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
