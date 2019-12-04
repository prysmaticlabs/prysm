package attestations

import (
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations/kv"
)

// AttestationPool defines the necessary methods for Prysm attestations pool to serve
// fork choice and validators. In the current design, aggregated attestations
// are used by proposer actor. Unaggregated attestations are used by
// for aggregator actor.
type AttestationPool interface {
	// For Aggregated attestations
	SaveAggregatedAttestation(att *ethpb.Attestation) error
	AggregatedAttestation() []*ethpb.Attestation
	DeleteAggregatedAttestation(att *ethpb.Attestation) error
	// For unaggregated attestations
	SaveUnaggregatedAttestation(att *ethpb.Attestation) error
	UnaggregatedAttestation(slot uint64, committeeIndex uint64) []*ethpb.Attestation
	DeleteUnaggregatedAttestation(att *ethpb.Attestation) error
	// For forkchoice attestations

}

// NewAttestationPool initializes a new attestation pool.
func NewAttestationPool(dirPath string) *kv.AttCaches {
	return kv.NewAttCaches()
}
