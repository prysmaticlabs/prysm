package aggregation

import (
	"fmt"
	"sort"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-bitfield"
)

// ErrInvalidMaxCoverProblem is returned when Maximum Coverage problem was initialized incorrectly.
var ErrInvalidMaxCoverProblem = errors.New("invalid max_cover problem")

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

// cover calculates solution to Maximum k-Cover problem.
func (mc *maxCoverProblem) cover(k int, allowOverlaps bool) (*maxCoverSolution, error) {
	if len(mc.candidates) == 0 {
		return nil, errors.Wrap(ErrInvalidMaxCoverProblem, "cannot calculate cover")
	}
	if len(mc.candidates) < k {
		k = len(mc.candidates)
	}

	remainingBits := mc.candidates.union()
	solution := &maxCoverSolution{
		coverage: bitfield.NewBitlist(mc.candidates[0].bits.Len()),
		keys:     make([]int, 0, k),
	}

	for len(solution.keys) < k && len(mc.candidates) > 0 {
		// Score candidates against remaining bits.
		// Filter out processed and overlapping (when disallowed).
		// Sort by score in a descending order.
		mc.candidates.score(remainingBits).filter(solution.coverage, allowOverlaps).sort()

		for _, candidate := range mc.candidates {
			if len(solution.keys) >= k {
				break
			}
			if !candidate.processed {
				if !allowOverlaps && solution.coverage.Overlaps(*candidate.bits) {
					// Overlapping candidates violate non-intersection invariant.
					candidate.processed = true
					continue
				}

				solution.coverage = solution.coverage.Or(*candidate.bits)
				remainingBits = remainingBits.And(candidate.bits.Not())
				solution.keys = append(solution.keys, candidate.key)
				candidate.processed = true
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

// filter removes processed, overlapping and zero-score candidates.
func (cl *maxCoverCandidateList) filter(covered bitfield.Bitlist, allowOverlaps bool) *maxCoverCandidateList {
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
	return fmt.Sprintf("{%v, %#b:%d, s%d, %t}",
		c.key, c.bits.Bytes(), c.bits.Len(), c.score, c.processed)
}

// String provides string representation of a Maximum Coverage problem solution.
func (s *maxCoverSolution) String() string {
	return fmt.Sprintf("{%#b:%v}", s.coverage.Bytes(), s.keys)
}
