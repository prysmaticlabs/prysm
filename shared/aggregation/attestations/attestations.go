package attestations

import (
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

// ErrInvalidAttestationCount is returned when insufficient number
// of attestations is provided for aggregation.
var ErrInvalidAttestationCount = errors.New("invalid number of attestations")

// AttestationAggregationStrategy defines attestation aggregation strategy.
type AttestationAggregationStrategy string

// selectedAttestationAggregateFn holds reference to the selected attestation aggregation strategy.
var selectedAttestationAggregateFn func(atts []*ethpb.Attestation) ([]*ethpb.Attestation, error)

