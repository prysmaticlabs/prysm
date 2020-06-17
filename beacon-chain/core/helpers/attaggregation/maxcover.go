package attaggregation

import (
	"fmt"
	"sort"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
)

// MaxCoverAggregation is a strategy based on Maximum Coverage greedy algorithm.
const MaxCoverAggregation AttestationAggregationStrategy = "max_cover"

// ErrInvalidAttestationCount is returned when insufficient number
// of attestations is provided for aggregation.
var ErrInvalidAttestationCount = errors.New("invalid number of attestations")

// ErrInvalidMaxCoverProblem is returned when Maximum Coverage problem was initialized incorrectly.
var ErrInvalidMaxCoverProblem = errors.New("invalid max_cover problem")

// MaxCoverAttestationAggregation relies on Maximum Coverage greedy algorithm for aggregation.
//
// For full analysis or running time, see "Analysis of the Greedy Approach in Problems of
// Maximum k-Coverage" by Hochbaum and Pathria.
func MaxCoverAttestationAggregation(atts []*ethpb.Attestation) ([]*ethpb.Attestation, error) {
	if len(atts) < 2 {
		return atts, nil
	}
	_, err := newMaxCoverProblem(atts)
	if err != nil {
		return atts, err
	}
	return atts, nil
	// TODO
	//return mc.cover()
}

// maxCoverProblem defines Maximum Coverage problem.
type maxCoverProblem struct {
	candidates maxCoverCandidateList
}

// maxCoverCandidate represents a candidate set to be used in aggregation.
type maxCoverCandidate struct {
	key       int
	bits      *bitfield.Bitlist
	score     uint64
	processed bool
}

// maxCoverCandidateList is defined to allow group operations (filtering, sorting) on all candidates.
type maxCoverCandidateList []*maxCoverCandidate

// maxCoverSolution represents the solution as bitlist of resultant coverage, and indices in
// attributes array grouped by aggregations.
type maxCoverSolution struct {
	coverage bitfield.Bitlist
	keys     []int
}

// newMaxCoverProblem returns initialized Maximum Coverage problem, with all necessary pre-tests done.
func newMaxCoverProblem(atts []*ethpb.Attestation) (*maxCoverProblem, error) {
	candidates, err := candidateListFromAttestations(atts)
	if err != nil {
		return nil, err
	}
	return &maxCoverProblem{candidates}, nil
}

// candidateListFromAttestations transforms list of attestations into candidate sets.
func candidateListFromAttestations(atts []*ethpb.Attestation) (maxCoverCandidateList, error) {
	if len(atts) == 0 {
		return nil, errors.Wrap(ErrInvalidAttestationCount, "cannot create list of candidates")
	}
	// Assert that all attestations have the same bitlist length.
	for i := 1; i < len(atts); i++ {
		if atts[i-1].AggregationBits.Len() != atts[i].AggregationBits.Len() {
			return nil, ErrBitsDifferentLen
		}
	}
	candidates := make([]*maxCoverCandidate, len(atts))
	for i := 0; i < len(atts); i++ {
		candidates[i] = &maxCoverCandidate{
			key:   i,
			bits:  &atts[i].AggregationBits,
			score: atts[i].AggregationBits.Count(),
		}
	}
	return candidates, nil
}

// cover calculates solution to Maximum k-Cover problem.
func (mc *maxCoverProblem) cover(k int, allowOverlaps bool) (*maxCoverSolution, error) {
	if len(mc.candidates) == 0 {
		return nil, errors.Wrap(ErrInvalidMaxCoverProblem, "cannot calculate cover")
	}
	if len(mc.candidates) < k {
		k = len(mc.candidates)
	}

	remainingBits := mc.candidates.union()

	fmt.Printf("remaining bits: %v\n", remainingBits)

	solution := &maxCoverSolution{
		coverage: bitfield.NewBitlist(mc.candidates[0].bits.Len()),
		keys:     make([]int, 0, k),
	}

	fmt.Printf("problemset: %v\n", mc.candidates)
	for len(solution.keys) < k && len(mc.candidates) > 0 {
		// Filter out processed and overlapping, sort by score in a descending order.
		mc.candidates.filter(solution.coverage).sort()
		fmt.Printf("problemset (sorted): %v\n", mc.candidates)

		// Pick enough non-overlapping candidates.
		for _, candidate := range mc.candidates {
			if !candidate.processed {
				if solution.coverage.Overlaps(*candidate.bits) {
					fmt.Printf("del: %v\n", candidate)
					// Overlapping candidates violate non-intersection invariant.
					candidate.processed = true
				} else {
					fmt.Printf("sel: %v\n", candidate)
					solution.coverage = solution.coverage.Or(*candidate.bits)
					solution.keys = append(solution.keys, candidate.key)
					candidate.processed = true
				}
			}
			if len(solution.keys) >= k {
				break
			}
		}
	}
	return solution, nil
}

// score updates scores of candidates, taking into account the uncovered elements only.
func (cl *maxCoverCandidateList) score(uncovered bitfield.Bitlist) *maxCoverCandidateList {
	for i := 0; i < len(*cl); i++ {
		(*cl)[i].score = (*cl)[i].bits.And(uncovered).Count()
	}
	return cl
}

// filter removes processed and overlapping candidates.
func (cl *maxCoverCandidateList) filter(covered bitfield.Bitlist) *maxCoverCandidateList {
	cur, end := 0, len(*cl)
	for cur < end {
		if (*cl)[cur].processed || covered.Overlaps(*(*cl)[cur].bits) {
			(*cl)[cur] = (*cl)[end-1]
			end--
			continue
		}
		cur++
	}
	*cl = (*cl)[:end]
	return cl
}

// sort orders candidates by their score, starting from the candidate with the highest score.
func (cl *maxCoverCandidateList) sort() *maxCoverCandidateList {
	sort.Slice(*cl, func(i, j int) bool {
		if (*cl)[i].score == (*cl)[j].score {
			return (*cl)[i].key < (*cl)[j].key
		}
		return (*cl)[i].score > (*cl)[j].score
	})
	return cl
}

func (cl *maxCoverCandidateList) union() bitfield.Bitlist {
	if len(*cl) == 0 {
		return nil
	}
	ret := bitfield.NewBitlist((*cl)[0].bits.Len())
	for i := 0; i < len(*cl); i++ {
		ret = ret.Or(*(*cl)[i].bits)
	}
	return ret
}

// String provides string representation of candidates list.
func (cl *maxCoverCandidateList) String() string {
	return fmt.Sprintf("candidates: %v", *cl)
}

// String provides string representation of a candidate.
func (c *maxCoverCandidate) String() string {
	return fmt.Sprintf("{%v, %#b, s%d, %t}", c.key, c.bits.Bytes(), c.score, c.processed)
}

// String provides string representation of a Maximum Coverage problem solution.
func (s *maxCoverSolution) String() string {
	return fmt.Sprintf("{%#b:%v}", s.coverage.Bytes(), s.keys)
}

//// aggregatedAttestation returns aggregated attestation containing all unaggregated attestations
//// from the solution.
//func (mc *maxCoverProblem) aggregatedAttestation() (*ethpb.Attestation, error) {
//	if len(mc.aggregated) < 1 {
//		return nil, errors.Wrap(ErrInvalidAttestationCount, "cannot aggregate solution")
//	}
//
//	signs := make([]*bls.Signature, len(mc.aggregated))
//	for i := 0; i < len(mc.aggregated); i++ {
//		sig, err := signatureFromBytes(*mc.aggregated[i].signature)
//		if err != nil {
//			return nil, err
//		}
//		signs[i] = sig
//	}
//	return &ethpb.Attestation{
//		AggregationBits: mc.coverage,
//		Data:            stateTrie.CopyAttestationData(mc.atts[0].Data),
//		Signature:       aggregateSignatures(signs).Marshal(),
//	}, nil
//}
//
//func (mc *maxCoverProblem) collectAttestations() ([]*ethpb.Attestation, error) {
//	// Remove processed items. Filter by score in desc.
//	mc.candidates.filter(mc.coverage).sort()
//
//	// Return results, start with the aggregated attestation.
//	atts := make([]*ethpb.Attestation, 1+len(mc.candidates)+len(mc.unaggregated))
//	ind := 0
//	if len(mc.aggregated) > 0 {
//		att, err := mc.aggregatedAttestation()
//		if err != nil {
//			return nil, err
//		}
//		atts[ind] = att
//		ind++
//	}
//
//	// Add unaggregated attestations.
//	for _, candidate := range mc.unaggregated {
//		//atts[ind] = &ethpb.Attestation{
//		//	AggregationBits: bytesutil.SafeCopyBytes(*candidate.bits),
//		//	Data:            stateTrie.CopyAttestationData(mc.atts[0].Data),
//		//	Signature:       bytesutil.SafeCopyBytes(*candidate.signature),
//		//}
//		atts[ind] = mc.atts[candidate.ind]
//		ind++
//	}
//
//	// Add left-over candidates (their fate was undecided as we had k items already).
//	for _, candidate := range mc.candidates {
//		if candidate.processed {
//			panic("wrong solution")
//		}
//		atts[ind] = mc.atts[candidate.ind]
//		ind++
//	}
//
//	return atts, nil
//}
