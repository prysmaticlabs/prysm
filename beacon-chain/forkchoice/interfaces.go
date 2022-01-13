package forkchoice

import (
	"context"
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
)

// ForkChoicer represents the full fork choice interface composed of all of the sub-interfaces.
type ForkChoicer interface {
	HeadRetriever        // to compute head.
	BlockProcessor       // to track new block for fork choice.
	AttestationProcessor // to track new attestation for fork choice.
	Pruner               // to clean old data for fork choice.
	Getter               // to retrieve fork choice information.
	ProposerBooster      // ability to boost timely-proposed block roots.
}

// HeadRetriever retrieves head root of the current chain.
type HeadRetriever interface {
	Head(context.Context, types.Epoch, [32]byte, []uint64, types.Epoch) ([32]byte, error)
}

// BlockProcessor processes the block that's used for accounting fork choice.
type BlockProcessor interface {
	ProcessBlock(context.Context, types.Slot, [32]byte, [32]byte, [32]byte, types.Epoch, types.Epoch) error
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
}

// Getter returns fork choice related information.
type Getter interface {
	Nodes() []*protoarray.Node
	Node([32]byte) *protoarray.Node
	HasNode([32]byte) bool
	Store() *protoarray.Store
	HasParent(root [32]byte) bool
	AncestorRoot(ctx context.Context, root [32]byte, slot types.Slot) ([]byte, error)
	IsCanonical(root [32]byte) bool
}
