package forkchoice

import (
	"context"
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	pbrpc "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// ForkChoicer represents the full fork choice interface composed of all the sub-interfaces.
type ForkChoicer interface {
	HeadRetriever        // to compute head.
	BlockProcessor       // to track new block for fork choice.
	AttestationProcessor // to track new attestation for fork choice.
	Pruner               // to clean old data for fork choice.
	Getter               // to retrieve fork choice information.
	Setter               // to set fork choice information.
	ProposerBooster      // ability to boost timely-proposed block roots.
}

// HeadRetriever retrieves head root and optimistic info of the current chain.
type HeadRetriever interface {
	Head(context.Context, types.Epoch, [32]byte, []uint64, types.Epoch) ([32]byte, error)
	Tips() ([][32]byte, []types.Slot)
	IsOptimistic(ctx context.Context, root [32]byte) (bool, error)
}

// BlockProcessor processes the block that's used for accounting fork choice.
type BlockProcessor interface {
	InsertOptimisticBlock(ctx context.Context,
		slot types.Slot,
		root [32]byte,
		parentRoot [32]byte,
		justifiedEpoch types.Epoch,
		finalizedEpoch types.Epoch,
	) error
}

// AttestationProcessor processes the attestation that's used for accounting fork choice.
type AttestationProcessor interface {
	ProcessAttestation(context.Context, []uint64, [32]byte, types.Epoch)
}

// Pruner prunes the fork choice upon new finalization. This is used to keep fork choice sane.
type Pruner interface {
	Prune(context.Context, [32]byte) error
}

// ProposerBooster is able to boost the proposer's root score during fork choice.
type ProposerBooster interface {
	BoostProposerRoot(ctx context.Context, blockSlot types.Slot, blockRoot [32]byte, genesisTime time.Time) error
	ResetBoostedProposerRoot(ctx context.Context) error
}

// Getter returns fork choice related information.
type Getter interface {
	HasNode([32]byte) bool
	ProposerBoost() [fieldparams.RootLength]byte
	HasParent(root [32]byte) bool
	ParentRoot(root [32]byte) ([32]byte, bool)
	AncestorRoot(ctx context.Context, root [32]byte, slot types.Slot) ([]byte, error)
	IsCanonical(root [32]byte) bool
	FinalizedEpoch() types.Epoch
	JustifiedEpoch() types.Epoch
	ForkChoiceNodes() []*pbrpc.ForkChoiceNode
	NodeCount() int
}

// Setter allows to set forkchoice information
type Setter interface {
	SetOptimisticToValid(context.Context, [fieldparams.RootLength]byte) error
	SetOptimisticToInvalid(context.Context, [fieldparams.RootLength]byte) error
}
