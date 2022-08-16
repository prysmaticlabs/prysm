package types

import (
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// ProposerBoostRootArgs to call the BoostProposerRoot function.
type ProposerBoostRootArgs struct {
	BlockRoot       [32]byte
	BlockSlot       types.Slot
	CurrentSlot     types.Slot
	SecondsIntoSlot uint64
}

// Checkpoint is an array version of ethpb.Checkpoint. It is used internally in
// forkchoice, while the slice version is used in the interface to legagy code
// in other packages
type Checkpoint struct {
	Epoch types.Epoch
	Root  [fieldparams.RootLength]byte
}

// BlockAndCheckpoints to call the InsertOptimisticChain function
type BlockAndCheckpoints struct {
	Block               interfaces.BeaconBlock
	JustifiedCheckpoint *ethpb.Checkpoint
	FinalizedCheckpoint *ethpb.Checkpoint
}
