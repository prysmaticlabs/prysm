package attestations

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1/attestation/aggregation"
	"github.com/sirupsen/logrus"
)

// attList represents list of attestations, defined for easier en masse operations (filtering, sorting).
type attList []blocks.ROAttestation

// BLS aggregate signature aliases for testing / benchmark substitution. These methods are
// significantly more expensive than the inner logic of AggregateAttestations so they must be
// substituted for benchmarks which analyze AggregateAttestations.
var aggregateSignatures = bls.AggregateSignatures
var signatureFromBytes = bls.SignatureFromBytesNoValidation

var _ = logrus.WithField("prefix", "aggregation.attestations")

// ErrInvalidAttestationCount is returned when insufficient number
// of attestations is provided for aggregation.
var ErrInvalidAttestationCount = errors.New("invalid number of attestations")

// Aggregate aggregates attestations. The minimal number of attestations is returned.
// Aggregation occurs in-place i.e. contents of input array will be modified. Should you need to
// preserve input attestations, clone them before aggregating
func Aggregate(atts []blocks.ROAttestation) ([]blocks.ROAttestation, error) {
	return MaxCoverAttestationAggregation(atts)
}

// AggregateDisjointOneBitAtts aggregates unaggregated attestations with the
// exact same attestation data.
func AggregateDisjointOneBitAtts(atts []blocks.ROAttestation) (blocks.ROAttestation, error) {
	if len(atts) == 0 {
		return blocks.ROAttestation{}, nil
	}
	if len(atts) == 1 {
		return atts[0], nil
	}
	coverage, err := atts[0].GetAggregationBits().ToBitlist64()
	if err != nil {
		return blocks.ROAttestation{}, errors.Wrap(err, "could not get aggregation bits")
	}
	for _, att := range atts[1:] {
		bits, err := att.GetAggregationBits().ToBitlist64()
		if err != nil {
			return blocks.ROAttestation{}, errors.Wrap(err, "could not get aggregation bits")
		}
		err = coverage.NoAllocOr(bits, coverage)
		if err != nil {
			return blocks.ROAttestation{}, errors.Wrap(err, "could not get aggregation bits")
		}
	}
	keys := make([]int, len(atts))
	for i := 0; i < len(atts); i++ {
		keys[i] = i
	}
	idx, err := aggregateAttestations(atts, keys, coverage)
	if err != nil {
		return blocks.ROAttestation{}, errors.Wrap(err, "could not aggregate attestations")
	}
	if idx != 0 {
		return blocks.ROAttestation{}, errors.New("could not aggregate attestations, obtained non zero index")
	}
	return atts[0], nil
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
