package attestations

import (
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations/kv"
)

// Pool defines the necessary methods for Prysm attestations pool to serve
// fork choice and validators. In the current design, aggregated attestations
// are used by proposer actor. Unaggregated attestations are used by
// aggregator actor.
type Pool interface {
	// For Aggregated attestations
	AggregateUnaggregatedAttestations() error
	SaveAggregatedAttestation(att *ethpb.Attestation) error
	SaveAggregatedAttestations(atts []*ethpb.Attestation) error
	AggregatedAttestations() []*ethpb.Attestation
	AggregatedAttestationsBySlotIndex(slot uint64, committeeIndex uint64) []*ethpb.Attestation
	DeleteAggregatedAttestation(att *ethpb.Attestation) error
	HasAggregatedAttestation(att *ethpb.Attestation) (bool, error)
	AggregatedAttestationCount() int
	ClearSeenAtts()
	// For unaggregated attestations.
	SaveUnaggregatedAttestation(att *ethpb.Attestation) error
	SaveUnaggregatedAttestations(atts []*ethpb.Attestation) error
	UnaggregatedAttestations() []*ethpb.Attestation
	UnaggregatedAttestationsBySlotIndex(slot uint64, committeeIndex uint64) []*ethpb.Attestation
	DeleteUnaggregatedAttestation(att *ethpb.Attestation) error
	UnaggregatedAttestationCount() int
	// For attestations that were included in the block.
	SaveBlockAttestation(att *ethpb.Attestation) error
	SaveBlockAttestations(atts []*ethpb.Attestation) error
	BlockAttestations() []*ethpb.Attestation
	DeleteBlockAttestation(att *ethpb.Attestation) error
	// For attestations to be passed to fork choice.
	SaveForkchoiceAttestation(att *ethpb.Attestation) error
	SaveForkchoiceAttestations(atts []*ethpb.Attestation) error
	ForkchoiceAttestations() []*ethpb.Attestation
	DeleteForkchoiceAttestation(att *ethpb.Attestation) error
}

// NewPool initializes a new attestation pool.
func NewPool() *kv.AttCaches {
	return kv.NewAttCaches()
}
