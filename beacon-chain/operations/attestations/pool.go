package attestations

import (
	"context"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/operations/attestations/kv"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
)

// Pool defines the necessary methods for Prysm attestations pool to serve
// fork choice and validators. In the current design, aggregated attestations
// are used by proposer actor. Unaggregated attestations are used by
// aggregator actor.
type Pool interface {
	// For Aggregated attestations
	AggregateUnaggregatedAttestations(ctx context.Context) error
	SaveAggregatedAttestation(att interfaces.Attestation) error
	SaveAggregatedAttestations(atts []interfaces.Attestation) error
	AggregatedAttestations() []interfaces.Attestation
	AggregatedAttestationsBySlotIndex(ctx context.Context, slot primitives.Slot, committeeIndex primitives.CommitteeIndex) []interfaces.Attestation
	DeleteAggregatedAttestation(att interfaces.Attestation) error
	HasAggregatedAttestation(att interfaces.Attestation) (bool, error)
	AggregatedAttestationCount() int
	// For unaggregated attestations.
	SaveUnaggregatedAttestation(att interfaces.Attestation) error
	SaveUnaggregatedAttestations(atts []interfaces.Attestation) error
	UnaggregatedAttestations() ([]interfaces.Attestation, error)
	UnaggregatedAttestationsBySlotIndex(ctx context.Context, slot primitives.Slot, committeeIndex primitives.CommitteeIndex) []interfaces.Attestation
	DeleteUnaggregatedAttestation(att interfaces.Attestation) error
	DeleteSeenUnaggregatedAttestations() (int, error)
	UnaggregatedAttestationCount() int
	// For attestations that were included in the block.
	SaveBlockAttestation(att interfaces.Attestation) error
	BlockAttestations() []interfaces.Attestation
	DeleteBlockAttestation(att interfaces.Attestation) error
	// For attestations to be passed to fork choice.
	SaveForkchoiceAttestation(att interfaces.Attestation) error
	SaveForkchoiceAttestations(atts []interfaces.Attestation) error
	ForkchoiceAttestations() []interfaces.Attestation
	DeleteForkchoiceAttestation(att interfaces.Attestation) error
	ForkchoiceAttestationCount() int
}

// NewPool initializes a new attestation pool.
func NewPool() *kv.AttCaches {
	return kv.NewAttCaches()
}
