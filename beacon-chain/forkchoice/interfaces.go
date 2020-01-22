package forkchoice

import (
	"context"
)

// ForkChoice represents the full fork choice interface composed of all of the sub-interfaces.
type ForkChoice interface {
	HeadRetriever        // to compute head.
	BlockProcessor       // to track new block for fork choice.
	AttestationProcessor // to track new attestation for fork choice.
	Pruner               // to clean old data for fork choice.
}

// HeadRetriever retrieves head root of the current chain.
type HeadRetriever interface {
	Head(context.Context, uint64, [32]byte, uint64, []uint64) ([32]byte, error)
}

// BlockProcessor processes the block that's used for accounting fork choice.
type BlockProcessor interface {
	ProcessBlock(context.Context, uint64, [32]byte, [32]byte, uint64, uint64) error
}

// AttestationProcessor processes the attestation that's used for accounting fork choice.
type AttestationProcessor interface {
	ProcessAttestation(context.Context, []uint64, [32]byte, uint64)
}

// Pruner prunes the fork choice upon new finalization. This is used to keep fork choice sane.
type Pruner interface {
	Prune(context.Context, [32]byte)
}
