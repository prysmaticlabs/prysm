package aggregation

import (
	"fmt"
	"sort"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-bitfield"
)

// ErrInvalidMaxCoverProblem is returned when Maximum Coverage problem was initialized incorrectly.
var ErrInvalidMaxCoverProblem = errors.New("invalid max_cover problem")

// MaxCoverProblem defines Maximum Coverage problem.
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

// Cover calculates solution to Maximum k-Cover problem.
func (mc *MaxCoverProblem) Cover(k int, allowOverlaps bool) (*Aggregation, error) {
	if len(mc.Candidates) == 0 {
		return nil, errors.Wrap(ErrInvalidMaxCoverProblem, "cannot calculate set coverage")
	}
	if len(mc.Candidates) < k {
		k = len(mc.Candidates)
	}

	solution := &Aggregation{
		Coverage: bitfield.NewBitlist(mc.Candidates[0].bits.Len()),
		Keys:     make([]int, 0, k),
	}
	remainingBits := mc.Candidates.union()

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
				remainingBits = remainingBits.And(candidate.bits.Not())
				solution.Keys = append(solution.Keys, candidate.key)
				candidate.processed = true
				break
			}
		}
	}
	return solution, nil
}

// score updates scores of Candidates, taking into account the uncovered elements only.
func (cl *MaxCoverCandidates) score(uncovered bitfield.Bitlist) *MaxCoverCandidates {
	for i := 0; i < len(*cl); i++ {
		(*cl)[i].score = (*cl)[i].bits.And(uncovered).Count()
	}
	return cl
}

// filter removes processed, overlapping and zero-score Candidates.
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

// sort orders Candidates by their score, starting from the candidate with the highest score.
func (cl *MaxCoverCandidates) sort() *MaxCoverCandidates {
	sort.Slice(*cl, func(i, j int) bool {
		if (*cl)[i].score == (*cl)[j].score {
			return (*cl)[i].key < (*cl)[j].key
		}
		return (*cl)[i].score > (*cl)[j].score
	})
	return cl
}

func (cl *MaxCoverCandidates) union() bitfield.Bitlist {
	if len(*cl) == 0 {
		return nil
	}
	ret := bitfield.NewBitlist((*cl)[0].bits.Len())
	for i := 0; i < len(*cl); i++ {
		ret = ret.Or(*(*cl)[i].bits)
	}
	return ret
}

// String provides string representation of Candidates list.
func (cl *MaxCoverCandidates) String() string {
	return fmt.Sprintf("Candidates: %v", *cl)
}

// String provides string representation of a candidate.
func (c *MaxCoverCandidate) String() string {
	return fmt.Sprintf("{%v, %#b:%d, s%d, %t}",
		c.key, c.bits.Bytes(), c.bits.Len(), c.score, c.processed)
}
