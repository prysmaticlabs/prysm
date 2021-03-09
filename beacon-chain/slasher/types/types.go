package types

import (
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
)

// ChunkKind to differentiate what kind of span we are working
// with for slashing detection, either min or max span.
type ChunkKind uint

const (
	MinSpan ChunkKind = iota
	MaxSpan
)

const (
	AttesterSlashing feed.EventType = iota
	ProposerSlashing
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

// Slashing represents a compact format with all the information
// needed to understand a slashable offense in eth2.
type Slashing struct {
	Kind            SlashingKind
	ValidatorIndex  types.ValidatorIndex
	PrevBeaconBlock *ethpb.SignedBeaconBlockHeader
	BeaconBlock     *ethpb.SignedBeaconBlockHeader
	SigningRoot     [32]byte
	PrevSigningRoot [32]byte
	TargetEpoch     types.Epoch
	PrevAttestation *ethpb.IndexedAttestation
	Attestation     *ethpb.IndexedAttestation
}

// SlashingKind is an enum representing the type of slashable
// offense detected by slasher, useful for conditionals or for logging.
type SlashingKind int

const (
	NotSlashable SlashingKind = iota
	DoubleVote
	SurroundingVote
	SurroundedVote
	DoubleProposal
)

func (k SlashingKind) String() string {
	switch k {
	case NotSlashable:
		return "NOT_SLASHABLE"
	case DoubleVote:
		return "DOUBLE_VOTE"
	case SurroundingVote:
		return "SURROUNDING_VOTE"
	case SurroundedVote:
		return "SURROUNDED_VOTE"
	default:
		return "UNKNOWN"
	}
}
