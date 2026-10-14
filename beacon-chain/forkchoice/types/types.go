package types

import (
	fieldparams "github.com/OffchainLabs/prysm/v7/config/fieldparams"
	consensus_blocks "github.com/OffchainLabs/prysm/v7/consensus-types/blocks"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
)

// Checkpoint is an array version of ethpb.Checkpoint. It is used internally in
// forkchoice, while the slice version is used in the interface to legacy code
// in other packages
type Checkpoint struct {
	Epoch primitives.Epoch
	Root  [fieldparams.RootLength]byte
}

// VoteData is the public subset of a validator's vote needed by the fast confirmation rule.
type VoteData struct {
	Root          [fieldparams.RootLength]byte
	Slot          primitives.Slot
	PayloadStatus bool
}

// BlockAndCheckpoints to call the InsertOptimisticChain function
type BlockAndCheckpoints struct {
	Block               consensus_blocks.ROBlock
	JustifiedCheckpoint *ethpb.Checkpoint
	FinalizedCheckpoint *ethpb.Checkpoint
	HasPayload          bool
	// FG is set for Decoupled blocks only, a batch can't reload their post-states.
	FG *FGSummary
}

// Pair identifies a block at an FG height.
type Pair struct {
	Height primitives.Height
	Root   [32]byte
}

// FGSummary contains a block state's claims, not the store's accepted J and F.
type FGSummary struct {
	Target    Pair
	Justified Pair
	Finalized Pair
}
