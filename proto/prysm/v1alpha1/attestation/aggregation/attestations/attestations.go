package attestations

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1/attestation/aggregation"
	"github.com/sirupsen/logrus"
)

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
	return MaxCoverAttestationAggregation(atts)
}

// AggregatePair aggregates pair of attestations a1 and a2 together.
func AggregatePair(a1, a2 *ethpb.Attestation) (*ethpb.Attestation, error) {
	o, err := a1.AggregationBits.Overlaps(a2.AggregationBits)
	if err != nil {
		return nil, err
	}
	if o {
		return nil, aggregation.ErrBitsOverlap
	}

	baseAtt := ethpb.CopyAttestation(a1)
	newAtt := ethpb.CopyAttestation(a2)
	if newAtt.AggregationBits.Count() > baseAtt.AggregationBits.Count() {
		baseAtt, newAtt = newAtt, baseAtt
	}

	c, err := baseAtt.AggregationBits.Contains(newAtt.AggregationBits)
	if err != nil {
		return nil, err
	}
	if c {
		return baseAtt, nil
	}

	newBits, err := baseAtt.AggregationBits.Or(newAtt.AggregationBits)
	if err != nil {
		return nil, err
	}
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
