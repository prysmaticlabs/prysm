package forkchoice

import (
	"context"

	forkchoicetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/types"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	v1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	eth "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// ForkChoicer represents the full fork choice interface composed of all the sub-interfaces.
type ForkChoicer interface {
	HeadRetriever        // to compute head.
	BlockProcessor       // to track new block for fork choice.
	AttestationProcessor // to track new attestation for fork choice.
	Getter               // to retrieve fork choice information.
	Setter               // to set fork choice information.
}

// HeadRetriever retrieves head root and optimistic info of the current chain.
type HeadRetriever interface {
	Head(context.Context, []uint64) ([32]byte, error)
	CachedHeadRoot() [32]byte
	Tips() ([][32]byte, []types.Slot)
	IsOptimistic(root [32]byte) (bool, error)
	AllTipsAreInvalid() bool
}

// BlockProcessor processes the block that's used for accounting fork choice.
type BlockProcessor interface {
	InsertNode(context.Context, state.BeaconState, [32]byte) error
	InsertOptimisticChain(context.Context, []*forkchoicetypes.BlockAndCheckpoints) error
}

// AttestationProcessor processes the attestation that's used for accounting fork choice.
type AttestationProcessor interface {
	ProcessAttestation(context.Context, []uint64, [32]byte, types.Epoch)
	InsertSlashedIndex(context.Context, types.ValidatorIndex)
	IsSlashed(types.ValidatorIndex) bool
	HasCurrentAggregate(*eth.AggregateAttestationAndProof) bool
	HasPreviousAggregate(*eth.AggregateAttestationAndProof) bool
	InsertCurrentAggregate(*eth.AggregateAttestationAndProof) error
	InsertPrevAggregate(*eth.AggregateAttestationAndProof) error
	MinusCurrentReferenceCount(index types.ValidatorIndex)
	MinusPrevReferenceCount(index types.ValidatorIndex)
}

// Getter returns fork choice related information.
type Getter interface {
	HasNode([32]byte) bool
	HasParent(root [32]byte) bool
	AncestorRoot(ctx context.Context, root [32]byte, slot types.Slot) ([32]byte, error)
	CommonAncestor(ctx context.Context, root1 [32]byte, root2 [32]byte) ([32]byte, types.Slot, error)
	IsCanonical(root [32]byte) bool
	FinalizedCheckpoint() *forkchoicetypes.Checkpoint
	FinalizedPayloadBlockHash() [32]byte
	JustifiedCheckpoint() *forkchoicetypes.Checkpoint
	PreviousJustifiedCheckpoint() *forkchoicetypes.Checkpoint
	JustifiedPayloadBlockHash() [32]byte
	BestJustifiedCheckpoint() *forkchoicetypes.Checkpoint
	NodeCount() int
	HighestReceivedBlockSlot() types.Slot
	HighestReceivedBlockRoot() [32]byte
	ReceivedBlocksLastEpoch() (uint64, error)
	ForkChoiceDump(context.Context) (*v1.ForkChoiceResponse, error)
	VotedFraction(root [32]byte) (uint64, error)
	CurrentLatestMessage(types.ValidatorIndex) ([32]byte, types.Epoch, error)
	PrevLatestMessage(types.ValidatorIndex) ([32]byte, types.Epoch, error)
	CurrentAttsByAggregator(types.ValidatorIndex) []*eth.AggregateAttestationAndProof
	PrevAttsByAggregator(types.ValidatorIndex) []*eth.AggregateAttestationAndProof
}

// Setter allows to set forkchoice information
type Setter interface {
	SetOptimisticToValid(context.Context, [fieldparams.RootLength]byte) error
	SetOptimisticToInvalid(context.Context, [fieldparams.RootLength]byte, [fieldparams.RootLength]byte, [fieldparams.RootLength]byte) ([][32]byte, error)
	UpdateJustifiedCheckpoint(*forkchoicetypes.Checkpoint) error
	UpdateFinalizedCheckpoint(*forkchoicetypes.Checkpoint) error
	SetGenesisTime(uint64)
	SetOriginRoot([32]byte)
	NewSlot(context.Context, types.Slot) error
}
