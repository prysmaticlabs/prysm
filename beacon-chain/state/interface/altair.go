package iface

// BeaconStateAltair has read and write access to beacon state methods.
type BeaconStateAltair interface {
	BeaconState
	AppendCurrentParticipationBits(val byte) error
	AppendPreviousParticipationBits(val byte) error
	AppendInactivityScore(s uint64) error
}
