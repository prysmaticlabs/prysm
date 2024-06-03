package attestations

import (
	"context"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/operations/attestations/kv"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

// Pool defines the necessary methods for Prysm attestations pool to serve
// fork choice and validators. In the current design, aggregated attestations
// are used by proposer actor. Unaggregated attestations are used by
// aggregator actor.
type Pool interface {
	// For Aggregated attestations
	AggregateUnaggregatedAttestations(ctx context.Context) error
	SaveAggregatedAttestation(att blocks.ROAttestation) error
	SaveAggregatedAttestations(atts []blocks.ROAttestation) error
	AggregatedAttestations() []blocks.ROAttestation
	AggregatedAttestationsBySlotIndex(ctx context.Context, slot primitives.Slot, committeeIndex primitives.CommitteeIndex) []ethpb.Att
	DeleteAggregatedAttestation(att blocks.ROAttestation) error
	HasAggregatedAttestation(att blocks.ROAttestation) (bool, error)
	AggregatedAttestationCount() int
	// For unaggregated attestations.
	SaveUnaggregatedAttestation(att blocks.ROAttestation) error
	SaveUnaggregatedAttestations(atts []blocks.ROAttestation) error
	UnaggregatedAttestations() ([]blocks.ROAttestation, error)
	UnaggregatedAttestationsBySlotIndex(ctx context.Context, slot primitives.Slot, committeeIndex primitives.CommitteeIndex) []ethpb.Att
	DeleteUnaggregatedAttestation(att blocks.ROAttestation) error
	DeleteSeenUnaggregatedAttestations() (int, error)
	UnaggregatedAttestationCount() int
	// For attestations that were included in the block.
	SaveBlockAttestation(att blocks.ROAttestation) error
	BlockAttestations() []blocks.ROAttestation
	DeleteBlockAttestation(att blocks.ROAttestation) error
	// For attestations to be passed to fork choice.
	SaveForkchoiceAttestation(att blocks.ROAttestation) error
	SaveForkchoiceAttestations(atts []blocks.ROAttestation) error
	ForkchoiceAttestations() []blocks.ROAttestation
	DeleteForkchoiceAttestation(att blocks.ROAttestation) error
	ForkchoiceAttestationCount() int
}

// NewPool initializes a new attestation pool.
func NewPool() *kv.AttCaches {
	return kv.NewAttCaches()
}
