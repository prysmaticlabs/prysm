package types

import (
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

// ChunkKind to differentiate what kind of span we are working
// with for slashing detection, either min or max span.
type ChunkKind uint

const (
	MinSpan ChunkKind = iota
	MaxSpan
)

// IndexedAttestationWrapper contains an indexed attestation with its
// data root to reduce duplicated computation.
type IndexedAttestationWrapper struct {
	IndexedAttestation *ethpb.IndexedAttestation
	DataRoot           [32]byte
}

// AttesterDoubleVote represents a double vote instance
// which is a slashable event for attesters.
type AttesterDoubleVote struct {
	Target         primitives.Epoch
	ValidatorIndex primitives.ValidatorIndex
	Wrapper_1      *IndexedAttestationWrapper
	Wrapper_2      *IndexedAttestationWrapper
}

// DoubleBlockProposal containing an incoming and an existing proposal's signing root.
type DoubleBlockProposal struct {
	Slot                   primitives.Slot
	ValidatorIndex         primitives.ValidatorIndex
	PrevBeaconBlockWrapper *SignedBlockHeaderWrapper
	BeaconBlockWrapper     *SignedBlockHeaderWrapper
}

// SignedBlockHeaderWrapper contains an signed beacon block header with its
// header root to reduce duplicated computation.
type SignedBlockHeaderWrapper struct {
	SignedBeaconBlockHeader *ethpb.SignedBeaconBlockHeader
	HeaderRoot              [32]byte
}

// AttestedEpochForValidator encapsulates a previously attested epoch
// for a validator index.
type AttestedEpochForValidator struct {
	ValidatorIndex primitives.ValidatorIndex
	Epoch          primitives.Epoch
}
