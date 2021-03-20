package iface

// BeaconStateV1 has read and write access to beacon state V1/HF1 methods.
type BeaconStateV1 interface {
	BeaconState
	CurrentEpochParticipation() []byte
	PreviousEpochParticipation() []byte
	AppendCurrentParticipationBits(val byte) error
	AppendPreviousParticipationBits(val byte) error
}
