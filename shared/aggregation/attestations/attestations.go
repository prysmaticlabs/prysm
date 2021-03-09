package attestations

import (
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateV0"
	"github.com/prysmaticlabs/prysm/shared/aggregation"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/sirupsen/logrus"
)

const (
	// NaiveAggregation is an aggregation strategy without any optimizations.
	NaiveAggregation AttestationAggregationStrategy = "naive"

	// MaxCoverAggregation is a strategy based on Maximum Coverage greedy algorithm.
	MaxCoverAggregation AttestationAggregationStrategy = "max_cover"

	// OptMaxCoverAggregation is a strategy based on Maximum Coverage greedy algorithm.
	// This new variant is optimized and relies on Bitlist64 (once fully tested, `max_cover`
	// strategy will be replaced with this one).
	OptMaxCoverAggregation AttestationAggregationStrategy = "opt_max_cover"
)

// AttestationAggregationStrategy defines attestation aggregation strategy.
type AttestationAggregationStrategy string

// attList represents list of attestations, defined for easier en masse operations (filtering, sorting).
type attList []*ethpb.Attestation

// BLS aggregate signature aliases for testing / benchmark substitution. These methods are
// significantly more expensive than the inner logic of AggregateAttestations so they must be
// substituted for benchmarks which analyze AggregateAttestations.
var aggregateSignatures = bls.AggregateSignatures
var signatureFromBytes = bls.SignatureFromBytes

var _ = logrus.WithField("prefix", "aggregation.attestations")

// ErrInvalidAttestationCount is returned when insufficient number
// of attestations is provided for aggregation.
var ErrInvalidAttestationCount = errors.New("invalid number of attestations")

// Aggregate aggregates attestations. The minimal number of attestations is returned.
// Aggregation occurs in-place i.e. contents of input array will be modified. Should you need to
// preserve input attestations, clone them before aggregating:
//
//   clonedAtts := make([]*ethpb.Attestation, len(atts))
//   for i, a := range atts {
//       clonedAtts[i] = stateTrie.CopyAttestation(a)
//   }
//   aggregatedAtts, err := attaggregation.Aggregate(clonedAtts)
func Aggregate(atts []*ethpb.Attestation) ([]*ethpb.Attestation, error) {
	strategy := AttestationAggregationStrategy(featureconfig.Get().AttestationAggregationStrategy)
	switch strategy {
	case "", NaiveAggregation:
		return NaiveAttestationAggregation(atts)
	case MaxCoverAggregation:
		return MaxCoverAttestationAggregation(atts)
	case OptMaxCoverAggregation:
		return optMaxCoverAttestationAggregation(atts)
	default:
		return nil, errors.Wrapf(aggregation.ErrInvalidStrategy, "%q", strategy)
	}
}

// AggregatePair aggregates pair of attestations a1 and a2 together.
func AggregatePair(a1, a2 *ethpb.Attestation) (*ethpb.Attestation, error) {
	if a1.AggregationBits.Len() != a2.AggregationBits.Len() {
		return nil, aggregation.ErrBitsDifferentLen
	}
	if a1.AggregationBits.Overlaps(a2.AggregationBits) {
		return nil, aggregation.ErrBitsOverlap
	}

	baseAtt := stateV0.CopyAttestation(a1)
	newAtt := stateV0.CopyAttestation(a2)
	if newAtt.AggregationBits.Count() > baseAtt.AggregationBits.Count() {
		baseAtt, newAtt = newAtt, baseAtt
	}

	if baseAtt.AggregationBits.Contains(newAtt.AggregationBits) {
		return baseAtt, nil
	}

	newBits := baseAtt.AggregationBits.Or(newAtt.AggregationBits)
	newSig, err := signatureFromBytes(newAtt.Signature)
	if err != nil {
		return nil, err
	}
	baseSig, err := signatureFromBytes(baseAtt.Signature)
	if err != nil {
		return nil, err
	}

	aggregatedSig := aggregateSignatures([]bls.Signature{baseSig, newSig})
	baseAtt.Signature = aggregatedSig.Marshal()
	baseAtt.AggregationBits = newBits

	return baseAtt, nil
}
