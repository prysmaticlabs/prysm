package forkchoice

import (
	"context"

	forkchoicetypes "github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// ForkChoicer represents the full fork choice interface composed of all the sub-interfaces.
type ForkChoicer interface {
	HeadRetriever        // to compute head.
	BlockProcessor       // to track new block for fork choice.
	AttestationProcessor // to track new attestation for fork choice.
	Getter               // to retrieve fork choice information.
	Setter               // to set fork choice information.
	ProposerBooster      // ability to boost timely-proposed block roots.
}

// HeadRetriever retrieves head root and optimistic info of the current chain.
type HeadRetriever interface {
	Head(context.Context, []uint64) ([32]byte, error)
	Tips() ([][32]byte, []types.Slot)
	IsOptimistic(root [32]byte) (bool, error)
}

// BlockProcessor processes the block that's used for accounting fork choice.
type BlockProcessor interface {
	InsertNode(context.Context, state.ReadOnlyBeaconState, [32]byte) error
	InsertOptimisticChain(context.Context, []*forkchoicetypes.BlockAndCheckpoints) error
}

// AttestationProcessor processes the attestation that's used for accounting fork choice.
type AttestationProcessor interface {
	ProcessAttestation(context.Context, []uint64, [32]byte, types.Epoch)
	InsertSlashedIndex(context.Context, types.ValidatorIndex)
}

// ProposerBooster is able to boost the proposer's root score during fork choice.
type ProposerBooster interface {
	ResetBoostedProposerRoot(ctx context.Context) error
}

// Getter returns fork choice related information.
type Getter interface {
	HasNode([32]byte) bool
	ProposerBoost() [fieldparams.RootLength]byte
	HasParent(root [32]byte) bool
	AncestorRoot(ctx context.Context, root [32]byte, slot types.Slot) ([32]byte, error)
	CommonAncestorRoot(ctx context.Context, root1 [32]byte, root2 [32]byte) ([32]byte, error)
	IsCanonical(root [32]byte) bool
	FinalizedCheckpoint() *forkchoicetypes.Checkpoint
	JustifiedCheckpoint() *forkchoicetypes.Checkpoint
	ForkChoiceNodes() []*ethpb.ForkChoiceNode
	NodeCount() int
}

// Setter allows to set forkchoice information
type Setter interface {
	SetOptimisticToValid(context.Context, [fieldparams.RootLength]byte) error
	SetOptimisticToInvalid(context.Context, [fieldparams.RootLength]byte, [fieldparams.RootLength]byte, [fieldparams.RootLength]byte) ([][32]byte, error)
	UpdateJustifiedCheckpoint(*forkchoicetypes.Checkpoint) error
	UpdateFinalizedCheckpoint(*forkchoicetypes.Checkpoint) error
	SetGenesisTime(uint64)
	SetOriginRoot([32]byte)
}
