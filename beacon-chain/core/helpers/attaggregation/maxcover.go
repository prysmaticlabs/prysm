package attaggregation

import (
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

// MaxCoverAggregation is a strategy based on Maximum Coverage greedy algorithm.
const MaxCoverAggregation AttestationAggregationStrategy = "max_cover"

// MaxCoverAttestationAggregation relies on Maximum Coverage greedy algorithm for aggregation.
func MaxCoverAttestationAggregation(atts []*ethpb.Attestation) ([]*ethpb.Attestation, error) {
	return atts, nil
}
