package decoupled

import (
	"bytes"

	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
)

// countSupport counts each participant once and excludes equivocators from
// subtree support. Each supporting vote is propagated to all ancestors.
func (st *Store) countSupport(v view) *supportCounts {
	c := &supportCounts{subtree: make(map[*Node]int)}
	votesBy := make(map[primitives.ValidatorIndex]int)
	for _, u := range v.participants {
		votesBy[u.Validator]++
	}
	c.participants = len(votesBy)
	for _, n := range votesBy {
		if n >= 2 {
			c.equivocators++
		}
	}
	for _, u := range v.supported {
		if votesBy[u.Validator] >= 2 {
			continue
		}
		for n := st.block(u); n != nil; n = n.Parent {
			c.subtree[n]++
		}
	}
	return c
}

// score gives every branch the same equivocation bonus for head selection.
// Confirmation leaves c.equivocators zero, but keeps them in the denominator.
func (c *supportCounts) score(b *Node) int { return c.equivocators + c.subtree[b] }

// hasMajority requires a strict majority of observed participants.
func (c *supportCounts) hasMajority(b *Node) bool { return c.participants < 2*c.score(b) }

func greaterRoot(a, b *Node) bool { return bytes.Compare(a.Root[:], b.Root[:]) > 0 }

// ghost follows the highest-scoring eligible child, breaking ties by greater root.
// The anchor itself need not pass either predicate.
func ghost(anchor *Node, in func(*Node) bool, score func(*Node) int, eligible func(*Node) bool) *Node {
	h := anchor
	for {
		var best *Node
		for _, c := range h.Children {
			if !in(c) || !eligible(c) {
				continue
			}
			if best == nil || score(c) > score(best) || (score(c) == score(best) && greaterRoot(c, best)) {
				best = c
			}
		}
		if best == nil {
			return h
		}
		h = best
	}
}
