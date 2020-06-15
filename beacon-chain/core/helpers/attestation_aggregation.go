package helpers

import (
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

const (
	// NaiveAggregation is an aggregation strategy without any optimizations.
	NaiveAggregation AttestationAggregationStrategy = "naive"
	// MaxCoverAggregation is a strategy based on Maximum Coverage greedy algorithm.
	MaxCoverAggregation AttestationAggregationStrategy = "max_cover"
)

// AttestationAggregationStrategy defines attestation aggregation strategy.
type AttestationAggregationStrategy string

var (
	// ErrAttestationAggregationBitsOverlap is returned when two attestations aggregation
	// bits overlap with each other.
	ErrAttestationAggregationBitsOverlap = errors.New("overlapping aggregation bits")

	// ErrAttestationAggregationBitsDifferentLen is returned when two attestation aggregation bits
	// have different lengths.
	ErrAttestationAggregationBitsDifferentLen = errors.New("different bitlist lengths")

	// ErrAttestationAggregationInvalidStrategy is returned when invalid aggregation strategy
	// is selected.
	ErrAttestationAggregationInvalidStrategy = errors.New("invalid aggregation strategy")
)

// AggregateAttestations aggregates attestations. The minimal number of attestations are returned.
func AggregateAttestations(atts []*ethpb.Attestation) ([]*ethpb.Attestation, error) {
	strategy := AttestationAggregationStrategy(featureconfig.Get().AttestationAggregationStrategy)
	switch strategy {
	case "", NaiveAggregation:
		return NaiveAttestationAggregation(atts)
	case MaxCoverAggregation:
		return MaxCoverAttestationAggregation(atts)
	}
	return nil, errors.Wrapf(ErrAttestationAggregationInvalidStrategy, "%q", strategy)
}

// NaiveAttestationAggregation aggregates naively, without any complex algorithms or optimizations.
// Note: this is currently a naive implementation to the order of O(mn^2).
func NaiveAttestationAggregation(atts []*ethpb.Attestation) ([]*ethpb.Attestation, error) {
	if len(atts) <= 1 {
		return atts, nil
	}

	// Naive aggregation. O(n^2) time.
	for i, a := range atts {
		if i >= len(atts) {
			break
		}
		for j := i + 1; j < len(atts); j++ {
			b := atts[j]
			if a.AggregationBits.Len() == b.AggregationBits.Len() && !a.AggregationBits.Overlaps(b.AggregationBits) {
				var err error
				a, err = AggregateAttestation(a, b)
				if err != nil {
					return nil, err
				}
				// Delete b
				atts = append(atts[:j], atts[j+1:]...)
				j--
				atts[i] = a
			}
		}
	}

	// Naive deduplication of identical aggregations. O(n^2) time.
	for i, a := range atts {
		for j := i + 1; j < len(atts); j++ {
			b := atts[j]

			if a.AggregationBits.Len() != b.AggregationBits.Len() {
				continue
			}

			if a.AggregationBits.Contains(b.AggregationBits) {
				// If b is fully contained in a, then b can be removed.
				atts = append(atts[:j], atts[j+1:]...)
				j--
			} else if b.AggregationBits.Contains(a.AggregationBits) {
				// if a is fully contained in b, then a can be removed.
				atts = append(atts[:i], atts[i+1:]...)
				i--
				break // Stop the inner loop, advance a.
			}
		}
	}

	return atts, nil
}

// MaxCoverAttestationAggregation relies on Maximum Coverage greedy algorithm for aggregation.
func MaxCoverAttestationAggregation(atts []*ethpb.Attestation) ([]*ethpb.Attestation, error) {
	return atts, nil
}

// AggregateAttestation aggregates attestations a1 and a2 together.
func AggregateAttestation(a1 *ethpb.Attestation, a2 *ethpb.Attestation) (*ethpb.Attestation, error) {
	if a1.AggregationBits.Len() != a2.AggregationBits.Len() {
		return nil, ErrAttestationAggregationBitsDifferentLen
	}
	if a1.AggregationBits.Overlaps(a2.AggregationBits) {
		return nil, ErrAttestationAggregationBitsOverlap
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
	newSig, err := bls.SignatureFromBytes(newAtt.Signature)
	if err != nil {
		return nil, err
	}
	baseSig, err := bls.SignatureFromBytes(baseAtt.Signature)
	if err != nil {
		return nil, err
	}

	aggregatedSig := bls.AggregateSignatures([]*bls.Signature{baseSig, newSig})
	baseAtt.Signature = aggregatedSig.Marshal()
	baseAtt.AggregationBits = newBits

	return baseAtt, nil
}
