package types

import (
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// ChunkKind to differentiate what kind of span we are working
// with for slashing detection, either min or max span.
type ChunkKind uint

const (
	MinSpan ChunkKind = iota
	MaxSpan
)

// IndexedAttestationWrapper contains an indexed attestation with its
// signing root to reduce duplicated computation.
type IndexedAttestationWrapper struct {
	IndexedAttestation *ethpb.IndexedAttestation
	SigningRoot        [32]byte
}

// AttesterDoubleVote represents a double vote instance
// which is a slashable event for attesters.
type AttesterDoubleVote struct {
	Target                 types.Epoch
	ValidatorIndex         types.ValidatorIndex
	PrevAttestationWrapper *IndexedAttestationWrapper
	AttestationWrapper     *IndexedAttestationWrapper
}

// DoubleBlockProposal containing an incoming and an existing proposal's signing root.
type DoubleBlockProposal struct {
	Slot                   types.Slot
	ValidatorIndex         types.ValidatorIndex
	PrevBeaconBlockWrapper *SignedBlockHeaderWrapper
	BeaconBlockWrapper     *SignedBlockHeaderWrapper
}

// SignedBlockHeaderWrapper contains an signed beacon block header with its
// signing root to reduce duplicated computation.
type SignedBlockHeaderWrapper struct {
	SignedBeaconBlockHeader *ethpb.SignedBeaconBlockHeader
	SigningRoot             [32]byte
}

// AttestedEpochForValidator encapsulates a previously attested epoch
// for a validator index.
type AttestedEpochForValidator struct {
	ValidatorIndex types.ValidatorIndex
	Epoch          types.Epoch
}
