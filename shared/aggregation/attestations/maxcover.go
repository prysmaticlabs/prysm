package attestations

import (
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/aggregation"
)

// NewMaxCover returns initialized Maximum Coverage problem for attestations aggregation.
func NewMaxCover(atts []*ethpb.Attestation) (*aggregation.MaxCoverProblem, error) {
	if len(atts) == 0 {
		return nil, errors.Wrap(ErrInvalidAttestationCount, "cannot create list of candidates")
	}

	// Assert that all attestations have the same bitlist length.
	for i := 1; i < len(atts); i++ {
		if atts[i-1].AggregationBits.Len() != atts[i].AggregationBits.Len() {
			return nil, aggregation.ErrBitsDifferentLen
		}
	}

	candidates := make([]*aggregation.MaxCoverCandidate, len(atts))
	for i := 0; i < len(atts); i++ {
		candidates[i] = aggregation.NewMaxCoverCandidate(i, &atts[i].AggregationBits)
	}
	return &aggregation.MaxCoverProblem{Candidates: candidates}, nil
}
