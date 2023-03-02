package forkchoice

import (
	"context"

	forkchoicetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/types"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	v1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
)

// BalancesByRooter is a handler to obtain the effective balances of the state
// with the given block root
type BalancesByRooter func(context.Context, [32]byte) ([]uint64, error)

// ForkChoicer represents the full fork choice interface composed of all the sub-interfaces.
type ForkChoicer interface {
	Lock()
	Unlock()
	RLock()
	RUnlock()
	HeadRetriever        // to compute head.
	BlockProcessor       // to track new block for fork choice.
	AttestationProcessor // to track new attestation for fork choice.
	Getter               // to retrieve fork choice information.
	Setter               // to set fork choice information.
}

// HeadRetriever retrieves head root and optimistic info of the current chain.
type HeadRetriever interface {
	Head(context.Context) ([32]byte, error)
	CachedHeadRoot() [32]byte
	Tips() ([][32]byte, []primitives.Slot)
	IsOptimistic(root [32]byte) (bool, error)
}

// BlockProcessor processes the block that's used for accounting fork choice.
type BlockProcessor interface {
	InsertNode(context.Context, state.BeaconState, [32]byte) error
	InsertChain(context.Context, []*forkchoicetypes.BlockAndCheckpoints) error
}

// AttestationProcessor processes the attestation that's used for accounting fork choice.
type AttestationProcessor interface {
	ProcessAttestation(context.Context, []uint64, [32]byte, primitives.Epoch)
	InsertSlashedIndex(context.Context, primitives.ValidatorIndex)
}

// Getter returns fork choice related information.
type Getter interface {
	HasNode([32]byte) bool
	ProposerBoost() [fieldparams.RootLength]byte
	AncestorRoot(ctx context.Context, root [32]byte, slot primitives.Slot) ([32]byte, error)
	CommonAncestor(ctx context.Context, root1 [32]byte, root2 [32]byte) ([32]byte, primitives.Slot, error)
	IsCanonical(root [32]byte) bool
	FinalizedCheckpoint() *forkchoicetypes.Checkpoint
	FinalizedPayloadBlockHash() [32]byte
	JustifiedCheckpoint() *forkchoicetypes.Checkpoint
	PreviousJustifiedCheckpoint() *forkchoicetypes.Checkpoint
	JustifiedPayloadBlockHash() [32]byte
	BestJustifiedCheckpoint() *forkchoicetypes.Checkpoint
	NodeCount() int
	HighestReceivedBlockSlot() primitives.Slot
	HighestReceivedBlockRoot() [32]byte
	ReceivedBlocksLastEpoch() (uint64, error)
	ForkChoiceDump(context.Context) (*v1.ForkChoiceDump, error)
	Weight(root [32]byte) (uint64, error)
}

// Setter allows to set forkchoice information
type Setter interface {
	SetOptimisticToValid(context.Context, [fieldparams.RootLength]byte) error
	SetOptimisticToInvalid(context.Context, [fieldparams.RootLength]byte, [fieldparams.RootLength]byte, [fieldparams.RootLength]byte) ([][32]byte, error)
	UpdateJustifiedCheckpoint(context.Context, *forkchoicetypes.Checkpoint) error
	UpdateFinalizedCheckpoint(*forkchoicetypes.Checkpoint) error
	SetGenesisTime(uint64)
	SetOriginRoot([32]byte)
	NewSlot(context.Context, primitives.Slot) error
	SetBalancesByRooter(BalancesByRooter)
}
