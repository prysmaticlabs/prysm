package types

import (
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
)

// ProposerBoostRootArgs to call the BoostProposerRoot function.
type ProposerBoostRootArgs struct {
	BlockRoot       [32]byte
	BlockSlot       types.Slot
	CurrentSlot     types.Slot
	SecondsIntoSlot uint64
}

// BlockAndCheckpoint to call the InsertOptimisticChain function
type BlockAndCheckpoint struct {
	Block          interfaces.BeaconBlock
	JustifiedEpoch types.Epoch
	FinalizedEpoch types.Epoch
}
