// Package attaggregation contains implementations of attestation aggregation
// algorithms and heuristics.
package attaggregation

import (
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/sirupsen/logrus"
)

// BLS aggregate signature aliases for testing / benchmark substitution. These methods are
// significantly more expensive than the inner logic of AggregateAttestations so they must be
// substituted for benchmarks which analyze AggregateAttestations.
var aggregateSignatures = bls.AggregateSignatures
var signatureFromBytes = bls.SignatureFromBytes

var log = logrus.WithField("prefix", "attaggregation")

var (
	// ErrBitsOverlap is returned when two attestations aggregation bits overlap with each other.
	ErrBitsOverlap = errors.New("overlapping aggregation bits")

	// ErrBitsDifferentLen is returned when two attestation aggregation bits have different lengths.
	ErrBitsDifferentLen = errors.New("different bitlist lengths")

	// ErrInvalidStrategy is returned when invalid aggregation strategy is selected.
	ErrInvalidStrategy = errors.New("invalid aggregation strategy")
)

// AttestationAggregationStrategy defines attestation aggregation strategy.
type AttestationAggregationStrategy string

// selectedAggregateFn holds reference to currently selected aggregation strategy.
var selectedAggregateFn func(atts []*ethpb.Attestation) ([]*ethpb.Attestation, error)

// Aggregate aggregates attestations. The minimal number of attestations is returned.
func Aggregate(atts []*ethpb.Attestation) ([]*ethpb.Attestation, error) {
	// Setup aggregation function once.
	if selectedAggregateFn == nil {
		strategy := AttestationAggregationStrategy(featureconfig.Get().AttestationAggregationStrategy)
		switch strategy {
		case "", NaiveAggregation:
			selectedAggregateFn = NaiveAttestationAggregation
		case MaxCoverAggregation:
			selectedAggregateFn = MaxCoverAttestationAggregation
		default:
			return nil, errors.Wrapf(ErrInvalidStrategy, "%q", strategy)
		}
	}
	return selectedAggregateFn(atts)
}

// AggregatePair aggregates pair of attestations a1 and a2 together.
func AggregatePair(a1 *ethpb.Attestation, a2 *ethpb.Attestation) (*ethpb.Attestation, error) {
	if a1.AggregationBits.Len() != a2.AggregationBits.Len() {
		return nil, ErrBitsDifferentLen
	}
	if a1.AggregationBits.Overlaps(a2.AggregationBits) {
		return nil, ErrBitsOverlap
	}

	baseAtt := stateTrie.CopyAttestation(a1)
	newAtt := stateTrie.CopyAttestation(a2)
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

	aggregatedSig := aggregateSignatures([]*bls.Signature{baseSig, newSig})
	baseAtt.Signature = aggregatedSig.Marshal()
	baseAtt.AggregationBits = newBits

	return baseAtt, nil
}
