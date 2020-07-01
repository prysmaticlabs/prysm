package attestations

import (
	"sort"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/aggregation"
	"github.com/prysmaticlabs/prysm/shared/bls"
)

// MaxCoverAttestationAggregation relies on Maximum Coverage greedy algorithm for aggregation.
// Aggregation occurs in many rounds, up until no more aggregation is possible (all attestations
// are overlapping).
func MaxCoverAttestationAggregation(atts []*ethpb.Attestation) ([]*ethpb.Attestation, error) {
	if len(atts) < 2 {
		return atts, nil
	}

	aggregated := attList(make([]*ethpb.Attestation, 0, len(atts)))
	unaggregated := attList(atts)

	// Aggregation over n/2 rounds is enough to find all aggregatable items (exits earlier if there
	// are many items that can be aggregated).
	for i := 0; i < len(atts)/2; i++ {
		if len(unaggregated) < 2 {
			break
		}

		// Find maximum non-overlapping coverage.
		maxCover, err := NewMaxCover(unaggregated)
		if err != nil {
			if err == aggregation.ErrBitsDifferentLen {
				return atts, nil
			}
			return aggregated.merge(unaggregated), err
		}
		solution, err := maxCover.Cover(len(atts), false /* allowOverlaps */, false /* allowDuplicates */)
		if err != nil {
			return aggregated.merge(unaggregated), err
		}

		// Exit earlier, if possible cover does not allow aggregation (less than two items).
		if len(solution.Keys) < 2 {
			break
		}

		// Create aggregated attestation and update solution lists.
		if !aggregated.hasCoverage(solution.Coverage) {
			att, err := unaggregated.selectUsingKeys(solution.Keys).aggregate(solution.Coverage)
			if err != nil {
				return aggregated.merge(unaggregated), err
			}
			aggregated = append(aggregated, att)
		}
		unaggregated = unaggregated.selectComplementUsingKeys(solution.Keys)
	}

	return aggregated.merge(unaggregated.filterContained()), nil
}

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

// aggregate returns list as an aggregated attestation.
func (al attList) aggregate(coverage bitfield.Bitlist) (*ethpb.Attestation, error) {
	if len(al) < 2 {
		return nil, errors.Wrap(ErrInvalidAttestationCount, "cannot aggregate")
	}
	signs := make([]bls.Signature, len(al))
	for i := 0; i < len(al); i++ {
		sig, err := signatureFromBytes(al[i].Signature)
		if err != nil {
			return nil, err
		}
		signs[i] = sig
	}
	return &ethpb.Attestation{
		AggregationBits: coverage,
		Data:            stateTrie.CopyAttestationData(al[0].Data),
		Signature:       aggregateSignatures(signs).Marshal(),
	}, nil
}

// merge combines two attestation lists into one.
func (al attList) merge(al1 attList) attList {
	merged := make([]*ethpb.Attestation, len(al)+len(al1))
	i := 0
	for _, att := range al {
		merged[i] = att
		i++
	}
	for _, att := range al1 {
		merged[i] = att
		i++
	}
	return merged
}

// selectUsingKeys returns only items with specified keys.
func (al attList) selectUsingKeys(keys []int) attList {
	filtered := make([]*ethpb.Attestation, len(keys))
	for i, key := range keys {
		filtered[i] = al[key]
	}
	return filtered
}

// selectComplementUsingKeys returns only items with keys that are NOT specified.
func (al attList) selectComplementUsingKeys(keys []int) attList {
	filtered := make([]*ethpb.Attestation, 0, len(keys))
	foundInKeys := func(key int) bool {
		for i := 0; i < len(keys); i++ {
			if keys[i] == key {
				keys[i] = keys[len(keys)-1]
				keys = keys[:len(keys)-1]
				return true
			}
		}
		return false
	}
	for i, att := range al {
		if !foundInKeys(i) {
			filtered = append(filtered, att)
		}
	}
	return filtered
}

// hasCoverage returns true if a given coverage is found in attestations list.
func (al attList) hasCoverage(coverage bitfield.Bitlist) bool {
	for _, att := range al {
		if att.AggregationBits.Xor(coverage).Count() == 0 {
			return true
		}
	}
	return false
}

// filterContained removes attestations that are contained within other attestations.
func (al attList) filterContained() attList {
	if len(al) < 2 {
		return al
	}
	sort.Slice(al, func(i, j int) bool {
		return al[i].AggregationBits.Count() > al[j].AggregationBits.Count()
	})
	filtered := make([]*ethpb.Attestation, 0, len(al))
	filtered = append(filtered, al[0])
	for i := 1; i < len(al); i++ {
		if filtered[len(filtered)-1].AggregationBits.Contains(al[i].AggregationBits) {
			continue
		}
		filtered = append(filtered, al[i])
	}
	return filtered
}
