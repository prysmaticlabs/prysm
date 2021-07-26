package attestations

import (
	"context"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations/kv"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// Pool defines the necessary methods for Prysm attestations pool to serve
// fork choice and validators. In the current design, aggregated attestations
// are used by proposer actor. Unaggregated attestations are used by
// aggregator actor.
type Pool interface {
	// For Aggregated attestations
	AggregateUnaggregatedAttestations(ctx context.Context) error
	AggregateUnaggregatedAttestationsBySlotIndex(ctx context.Context, slot types.Slot, committeeIndex types.CommitteeIndex) error
	SaveAggregatedAttestation(att *ethpb.Attestation) error
	SaveAggregatedAttestations(atts []*ethpb.Attestation) error
	AggregatedAttestations() []*ethpb.Attestation
	AggregatedAttestationsBySlotIndex(ctx context.Context, slot types.Slot, committeeIndex types.CommitteeIndex) []*ethpb.Attestation
	DeleteAggregatedAttestation(att *ethpb.Attestation) error
	HasAggregatedAttestation(att *ethpb.Attestation) (bool, error)
	AggregatedAttestationCount() int
	// For unaggregated attestations.
	SaveUnaggregatedAttestation(att *ethpb.Attestation) error
	SaveUnaggregatedAttestations(atts []*ethpb.Attestation) error
	UnaggregatedAttestations() ([]*ethpb.Attestation, error)
	UnaggregatedAttestationsBySlotIndex(ctx context.Context, slot types.Slot, committeeIndex types.CommitteeIndex) []*ethpb.Attestation
	DeleteUnaggregatedAttestation(att *ethpb.Attestation) error
	DeleteSeenUnaggregatedAttestations() (int, error)
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
	ForkchoiceAttestationCount() int
}

// NewPool initializes a new attestation pool.
func NewPool() *kv.AttCaches {
	return kv.NewAttCaches()
}
