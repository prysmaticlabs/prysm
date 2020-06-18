package aggregation

import (
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

// ErrInvalidAttestationCount is returned when insufficient number
// of attestations is provided for aggregation.
var ErrInvalidAttestationCount = errors.New("invalid number of attestations")

// NewAttestationsMaxCover returns initialized Maximum Coverage problem.
func NewAttestationsMaxCover(atts []*ethpb.Attestation) (*MaxCoverProblem, error) {
	candidates, err := candidateListFromAttestations(atts)
	if err != nil {
		return nil, err
	}
	return &MaxCoverProblem{candidates}, nil
}

// candidateListFromAttestations transforms list of attestations into candidate sets.
func candidateListFromAttestations(atts []*ethpb.Attestation) (MaxCoverCandidates, error) {
	if len(atts) == 0 {
		return nil, errors.Wrap(ErrInvalidAttestationCount, "cannot create list of candidates")
	}
	// Assert that all attestations have the same bitlist length.
	for i := 1; i < len(atts); i++ {
		if atts[i-1].AggregationBits.Len() != atts[i].AggregationBits.Len() {
			return nil, ErrBitsDifferentLen
		}
	}
	candidates := make([]*MaxCoverCandidate, len(atts))
	for i := 0; i < len(atts); i++ {
		candidates[i] = &MaxCoverCandidate{
			key:  i,
			bits: &atts[i].AggregationBits,
		}
	}
	return candidates, nil
}
