package iface

// BeaconStateAltair has read and write access to beacon state Altair hard fork1 methods.
type BeaconStateAltair interface {
	BeaconState
	CurrentEpochParticipation() []byte
	PreviousEpochParticipation() []byte
	AppendCurrentParticipationBits(val byte) error
	AppendPreviousParticipationBits(val byte) error
	InactivityScores() []uint64
	AppendInactivityScore(s uint64) error
}
