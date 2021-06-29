package iface

import pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"

// BeaconStateAltair has read and write access to beacon state methods.
type BeaconStateAltair interface {
	BeaconState
	CurrentSyncCommittee() (*pbp2p.SyncCommittee, error)
	NextSyncCommittee() (*pbp2p.SyncCommittee, error)
	SetCurrentSyncCommittee(val *pbp2p.SyncCommittee) error
	SetNextSyncCommittee(val *pbp2p.SyncCommittee) error
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
