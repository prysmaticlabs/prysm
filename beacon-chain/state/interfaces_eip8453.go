package state

import (
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
)

// Returned values are shared with state copies, replace them through the setters instead of mutating.
type readOnlyEIP8453Fields interface {
	TargetCheckpoint() (*ethpb.HeightCheckpoint, error)
	JustifiedCheckpointEIP8453() (*ethpb.HeightCheckpoint, error)
	FinalizedCheckpointEIP8453() (*ethpb.HeightCheckpoint, error)
	TargetSlot() (primitives.Slot, error)
	FinalizedSlot() (primitives.Slot, error)
	TargetParticipationEIP8453() ([]byte, error)
	ProgressParticipation() ([]byte, error)
	RoundParticipation() ([]byte, error)
	PendingFinalities() ([]*ethpb.PendingFinalityEIP8453, error)
	HeadVoteParticipation() ([][]byte, error)
}

type writeOnlyEIP8453Fields interface {
	SetTargetCheckpoint(*ethpb.HeightCheckpoint) error
	SetJustifiedCheckpointEIP8453(*ethpb.HeightCheckpoint) error
	SetFinalizedCheckpointEIP8453(*ethpb.HeightCheckpoint) error
	SetTargetSlot(primitives.Slot) error
	SetFinalizedSlot(primitives.Slot) error
	SetTargetParticipationEIP8453([]byte) error
	SetProgressParticipation([]byte) error
	SetRoundParticipation([]byte) error
	SetPendingFinalities([]*ethpb.PendingFinalityEIP8453) error
	SetHeadVoteParticipation([][]byte) error
}
