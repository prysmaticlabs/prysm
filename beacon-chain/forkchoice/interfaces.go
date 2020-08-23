package forkchoice

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
)

// ForkChoicer represents the full fork choice interface composed of all of the sub-interfaces.
type ForkChoicer interface {
	HeadRetriever        // to compute head.
	BlockProcessor       // to track new block for fork choice.
	AttestationProcessor // to track new attestation for fork choice.
	Pruner               // to clean old data for fork choice.
	Getter               // to retrieve fork choice information.
}

// HeadRetriever retrieves head root of the current chain.
type HeadRetriever interface {
	Head(context.Context, uint64, [32]byte, []uint64, uint64) ([32]byte, error)
}

// BlockProcessor processes the block that's used for accounting fork choice.
type BlockProcessor interface {
	ProcessBlock(context.Context, uint64, [32]byte, [32]byte, [32]byte, uint64, uint64) error
}

// AttestationProcessor processes the attestation that's used for accounting fork choice.
type AttestationProcessor interface {
	ProcessAttestation(context.Context, []uint64, [32]byte, uint64)
}

// Pruner prunes the fork choice upon new finalization. This is used to keep fork choice sane.
type Pruner interface {
	Prune(context.Context, [32]byte) error
}

// Getter returns fork choice related information.
type Getter interface {
	Nodes() []*protoarray.Node
	Node([32]byte) *protoarray.Node
	HasNode([32]byte) bool
	Store() *protoarray.Store
	HasParent(root [32]byte) bool
	AncestorRoot(ctx context.Context, root [32]byte, slot uint64) ([]byte, error)
}
