package attestations

import (
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations/kv"
)

// Pool defines the necessary methods for Prysm attestations pool to serve
// fork choice and validators. In the current design, aggregated attestations
// are used by proposer actor. Unaggregated attestations are used by
// for aggregator actor.
type Pool interface {
	SaveAggregatedAttestation(att *ethpb.Attestation) error
	AggregatedAttestation() []*ethpb.Attestation
	SaveUnaggregatedAttestation(att *ethpb.Attestation) error
	UnaggregatedAttestation() []*ethpb.Attestation
}

// NewPool initializes a new attestation pool.
func NewPool(dirPath string) *kv.AttCaches {
	return kv.NewAttCaches()
}
