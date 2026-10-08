package decoupled

import (
	"slices"
	"time"

	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
)

// live does not require a viable descendant: such locks still veto confirmation.
func (st *Store) live(l lock, now time.Time) bool {
	return l.deadline.After(now) && st.inTree(l.block)
}

// conflicts excludes ancestors and descendants, including equality.
func conflicts(a, b *Node) bool { return !isAncestor(a, b) && !isAncestor(b, a) }

// vetoed checks for conflicting live locks.
func (st *Store) vetoed(b *Node, now time.Time) bool {
	for _, l := range st.locks {
		if st.live(l, now) && conflicts(l.block, b) {
			return true
		}
	}
	return false
}

// anchor starts at the deepest live lock and retreats to a viable ancestor.
// It falls back to the FG root and uses the full tree, not the frozen voter view.
func (st *Store) anchor(fr frame, now time.Time) *Node {
	var deepest *Node
	for _, l := range st.locks {
		if st.live(l, now) && (deepest == nil || l.block.Slot > deepest.Slot) {
			deepest = l.block
		}
	}
	if deepest == nil {
		return fr.root
	}
	viable := st.viableTree(fr, everyNode)
	for n := deepest; n != nil; n = n.Parent {
		if viable(n) {
			return n
		}
	}
	return fr.root
}

// unlock uses votes available before 5*delta to retreat locks with opposition
// from at least half the committee. Deadlines are preserved.
func (st *Store) unlock(slot primitives.Slot, now time.Time) {
	st.locks = slices.DeleteFunc(st.locks, func(l lock) bool { return !st.live(l, now) })
	for i := range st.locks {
		st.locks[i].block = st.backUp(st.locks[i].block, st.pool[slot], st.unlockTime(slot))
	}
}

// backUp finds the deepest ancestor with less than half-committee opposition.
// Equivocators do not count; F is compatible with every retained block.
func (st *Store) backUp(locked *Node, b *bucket, asOf time.Time) *Node {
	var voted []*Node
	if b != nil {
		votesBy := b.votesBefore(asOf)
		for _, u := range b.votes {
			if at, ok := st.processedAt(u); ok && at.Before(asOf) && votesBy[u.Validator] == 1 {
				voted = append(voted, st.block(u))
			}
		}
	}
	n := locked
	for ; n.Parent != nil; n = n.Parent {
		against := uint64(0)
		for _, r := range voted {
			if conflicts(n, r) {
				against++
			}
		}
		if 2*against < st.committeeSize {
			break
		}
	}
	return n
}
