package aggregation

import (
	"sort"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-bitfield"
)

// ErrInvalidMaxCoverProblem is returned when Maximum Coverage problem was initialized incorrectly.
var ErrInvalidMaxCoverProblem = errors.New("invalid max_cover problem")

// MaxCoverProblem defines Maximum Coverage problem.
//
// Problem is defined as MaxCover(U, S, k): S', where:
// U is a finite set of objects, where |U| = n. Furthermore, let S = {S_1, ..., S_m} be all
// subsets of U, that's their union is equal to U. Then, Maximum Coverage is the problem of
// finding such a collection S' of subsets from S, where |S'| <= k, and union of all subsets in S'
// covering U with maximum cardinality.
//
// The current implementation captures the original MaxCover problem, and the variant where
// additional invariant is enforced: all elements of S' must be disjoint. This comes handy when
// we need to aggregate bitsets, and overlaps are not allowed.
//
// For more details, see:
// "Analysis of the Greedy Approach in Problems of Maximum k-Coverage" by Hochbaum and Pathria.
// https://hochbaum.ieor.berkeley.edu/html/pub/HPathria-max-k-coverage-greedy.pdf
type MaxCoverProblem struct {
	Candidates MaxCoverCandidates
}

// MaxCoverCandidate represents a candidate set to be used in aggregation.
type MaxCoverCandidate struct {
	key       int
	bits      *bitfield.Bitlist
	score     uint64
	processed bool
}

// MaxCoverCandidates is defined to allow group operations (filtering, sorting) on all candidates.
type MaxCoverCandidates []*MaxCoverCandidate

// NewMaxCoverCandidate returns initialized candidate.
func NewMaxCoverCandidate(key int, bits *bitfield.Bitlist) *MaxCoverCandidate {
	return &MaxCoverCandidate{
		key:  key,
		bits: bits,
	}
}

// Cover calculates solution to Maximum k-Cover problem in O(knm), where
// n is number of candidates and m is a length of bitlist in each candidate.
func (mc *MaxCoverProblem) Cover(k int, allowOverlaps bool, allowDuplicates bool) (*Aggregation, error) {
	if len(mc.Candidates) == 0 {
		return nil, errors.Wrap(ErrInvalidMaxCoverProblem, "cannot calculate set coverage")
	}
	if len(mc.Candidates) < k {
		k = len(mc.Candidates)
	}

	if err := mc.Candidates.validate(); err != nil {
		return nil, err
	}

	if !allowDuplicates {
		mc.Candidates.dedup(allowOverlaps)
	}

	solution := &Aggregation{
		Coverage: bitfield.NewBitlist(mc.Candidates[0].bits.Len()),
		Keys:     make([]int, 0, k),
	}
	remainingBits := mc.Candidates.union()
	if remainingBits == nil {
		return nil, errors.Wrap(ErrInvalidMaxCoverProblem, "empty bitlists")
	}

	for len(solution.Keys) < k && len(mc.Candidates) > 0 {
		// Score candidates against remaining bits.
		// Filter out processed and overlapping (when disallowed).
		// Sort by score in a descending order.
		mc.Candidates.score(remainingBits).filter(solution.Coverage, allowOverlaps).sort()

		for _, candidate := range mc.Candidates {
			if len(solution.Keys) >= k {
				break
			}
			if !candidate.processed {
				if !allowOverlaps && solution.Coverage.Overlaps(*candidate.bits) {
					// Overlapping candidates violate non-intersection invariant.
					candidate.processed = true
					continue
				}
				solution.Coverage = solution.Coverage.Or(*candidate.bits)
				solution.Keys = append(solution.Keys, candidate.key)
				remainingBits = remainingBits.And(candidate.bits.Not())
				candidate.processed = true
				break
			}
		}
	}
	return solution, nil
}

// score updates scores of candidates, taking into account the uncovered elements only.
func (cl *MaxCoverCandidates) score(uncovered bitfield.Bitlist) *MaxCoverCandidates {
	for i := 0; i < len(*cl); i++ {
		(*cl)[i].score = (*cl)[i].bits.And(uncovered).Count()
	}
	return cl
}

// filter removes processed, overlapping and zero-score candidates.
func (cl *MaxCoverCandidates) filter(covered bitfield.Bitlist, allowOverlaps bool) *MaxCoverCandidates {
	overlaps := func(e bitfield.Bitlist) bool {
		return !allowOverlaps && covered.Len() == e.Len() && covered.Overlaps(e)
	}
	cur, end := 0, len(*cl)
	for cur < end {
		e := *(*cl)[cur]
		if e.processed || overlaps(*e.bits) || e.score == 0 {
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
func (cl *MaxCoverCandidates) sort() *MaxCoverCandidates {
	sort.Slice(*cl, func(i, j int) bool {
		if (*cl)[i].score == (*cl)[j].score {
			return (*cl)[i].key < (*cl)[j].key
		}
		return (*cl)[i].score > (*cl)[j].score
	})
	return cl
}

// union merges all candidate bitlists using logical OR operator.
func (cl *MaxCoverCandidates) union() bitfield.Bitlist {
	if len(*cl) == 0 {
		return nil
	}
	if (*cl)[0].bits == nil || (*cl)[0].bits.Len() == 0 {
		return nil
	}
	ret := bitfield.NewBitlist((*cl)[0].bits.Len())
	for i := 0; i < len(*cl); i++ {
		if *(*cl)[i].bits != nil {
			ret = ret.Or(*(*cl)[i].bits)
		}
	}
	return ret
}

// dedup removes duplicate candidates (ones with the same bits set on).
func (cl *MaxCoverCandidates) dedup(allowOverlaps bool) *MaxCoverCandidates {
	if len(*cl) < 2 {
		return cl
	}
	uncoveredBits := cl.union()
	if uncoveredBits == nil {
		return cl
	}
	cl.score(uncoveredBits).sort()
	for i := 1; i < len(*cl); i++ {
		nonOverlappingBits := (*cl)[i-1].bits.Xor(*(*cl)[i].bits)
		if (*cl)[i-1].score == (*cl)[i].score && nonOverlappingBits.Count() == 0 {
			(*cl)[i-1].processed = true
		}
	}
	return cl.filter(bitfield.NewBitlist((*cl)[0].bits.Len()), allowOverlaps)
}

// validate checks candidates for validity (equal bitlength etc).
func (cl *MaxCoverCandidates) validate() error {
	if len(*cl) == 0 {
		return errors.Wrap(ErrInvalidMaxCoverProblem, "empty list of candidates")
	}
	if (*cl)[0].bits == nil || (*cl)[0].bits.Len() == 0 {
		return errors.Wrap(ErrInvalidMaxCoverProblem, "bitlist cannot be nil or empty")
	}
	bitlistLen := (*cl)[0].bits.Len()
	for i := 1; i < len(*cl); i++ {
		if (*cl)[i].bits == nil || (*cl)[i].bits.Len() == 0 {
			return errors.Wrap(ErrInvalidMaxCoverProblem, "bitlist cannot be nil or empty")
		}
		if bitlistLen != (*cl)[i].bits.Len() {
			return errors.Wrap(ErrInvalidMaxCoverProblem, "bitlists of different length")
		}
	}
	return nil
}
