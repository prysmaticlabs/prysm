package attaggregation

import (
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bls"
)

// MaxCoverAggregation is a strategy based on Maximum Coverage greedy algorithm.
const MaxCoverAggregation AttestationAggregationStrategy = "max_cover"

// MaxCoverAttestationAggregation relies on Maximum Coverage greedy algorithm for aggregation.
//
// For full analysis or running time, see "Analysis of the Greedy Approach in Problems of
// Maximum k-Coverage" by Hochbaum and Pathria.
func MaxCoverAttestationAggregation(atts []*ethpb.Attestation) ([]*ethpb.Attestation, error) {
	if len(atts) < 2 {
		return atts, nil
	}
	mc, err := newMaxCoverage(atts, len(atts))
	if err != nil {
		return atts, err
	}
	return mc.solve()
}

// maxCoverage defines Maximum k-Coverage problem.
type maxCoverage struct {
	k          int
	candidates map[int]*bitfield.Bitlist
	coverage   bitfield.Bitlist
	solution   map[int]struct{}
	input      struct {
		atts  *[]*ethpb.Attestation
		data  *ethpb.AttestationData
		signs map[int]*[]byte
	}
}

// newMaxCoverage returns initialized Maximum k-Coverage problem, with all necessary pre-tests done.
func newMaxCoverage(atts []*ethpb.Attestation, k int) (*maxCoverage, error) {
	if len(atts) == 0 {
		return &maxCoverage{}, nil
	}
	if len(atts) < k {
		k = len(atts)
	}

	// Assert that all attestations have the same bitlist length.
	for i := 1; i < len(atts); i++ {
		if atts[i-1].AggregationBits.Len() != atts[i].AggregationBits.Len() {
			return nil, ErrBitsDifferentLen
		}
	}

	// Populate candidates to select cover from.
	candidateSets := make(map[int]*bitfield.Bitlist, len(atts))
	candidateSigns := make(map[int]*[]byte, len(atts))
	for key, att := range atts {
		candidateSets[key] = &att.AggregationBits
		candidateSigns[key] = &att.Signature
	}

	return &maxCoverage{
		k:          k,
		coverage:   bitfield.NewBitlist(atts[0].AggregationBits.Len()),
		candidates: candidateSets,
		solution:   make(map[int]struct{}, k),
		input: struct {
			atts  *[]*ethpb.Attestation
			data  *ethpb.AttestationData
			signs map[int]*[]byte
		}{atts: &atts, data: atts[0].Data, signs: candidateSigns},
	}, nil
}

func (mc *maxCoverage) findCoverage() {
	if len(mc.candidates) == 0 {
		return
	}

	for len(mc.solution) < mc.k && len(mc.candidates) > 0 {
		// Select candidate that maximizes score.
		bestCandidateIndex := -1
		for ind, candidate := range mc.candidates {
			if mc.coverage.Overlaps(*candidate) {
				// Overlapping candidates violate non-intersection invariant.
				delete(mc.candidates, ind)
				continue
			}
			if bestCandidateIndex == -1 || candidate.Len() > mc.candidates[bestCandidateIndex].Len() {
				bestCandidateIndex = ind
			}
		}
		// Update partial solution.
		if bestCandidateIndex >= 0 {
			mc.coverage = mc.coverage.Or(*mc.candidates[bestCandidateIndex])
			mc.solution[bestCandidateIndex] = struct{}{}
			delete(mc.candidates, bestCandidateIndex)
		}
	}
}

func (mc *maxCoverage) solve() ([]*ethpb.Attestation, error) {
	mc.findCoverage()

	// Start with aggregated attestation.
	atts := make([]*ethpb.Attestation, 0, len(mc.solution))
	if len(mc.solution) > 0 {
		// Collect selected signatures.
		var signs []*bls.Signature
		for key := range mc.solution {
			sig, err := bls.SignatureFromBytes(*mc.input.signs[key])
			if err != nil {
				return nil, err
			}
			signs = append(signs, sig)
		}
		atts = append(atts, &ethpb.Attestation{
			AggregationBits: mc.coverage,
			Data:            stateTrie.CopyAttestationData(mc.input.data),
			Signature:       bls.AggregateSignatures(signs).Marshal(),
		})
	}

	// Add unaggregated attestations.
	for key, att := range *mc.input.atts {
		if _, ok := mc.solution[key]; !ok {
			// Consider att := stateTrie.CopyAttestation(att)
			atts = append(atts, att)
		}
	}

	lst := make([]bitfield.Bitlist, 0)
	for _, att := range atts {
		lst = append(lst, att.AggregationBits)
	}
	//log.WithFields(logrus.Fields{
	//	"list": fmt.Sprintf("%#v\n", lst),
	//}).Debug("-----> Collected")

	return atts, nil
}
