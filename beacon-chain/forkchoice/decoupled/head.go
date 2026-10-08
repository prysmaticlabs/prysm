package decoupled

import (
	"slices"
	"time"

	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
)

// selectHead applies the vote and block views after choosing an anchor from
// the full tree. The anchor need not belong to the supplied block view.
func (st *Store) selectHead(in func(*Node) bool, v view, now time.Time) *Node {
	fr := st.frame()
	counts := st.countSupport(v)
	return ghost(st.anchor(fr, now), st.viableTree(fr, in), counts.score, fr.eligible(counts, st.slotAt(now)))
}

// Proposal ticks to now, then returns the parent and previous-slot vote lists from one view.
// The slices are caller-owned. The parent is nil until Joined(slot).
func (st *Store) Proposal(slot primitives.Slot, now time.Time) (parent *Node, votes, support []Vote) {
	st.OnTick(now)
	if !st.joined(slot) {
		return nil, nil, nil
	}
	v := st.heldView(slot - 1)
	return st.selectHead(everyNode, v, now), slices.Clone(v.participants), v.supported
}

// The frozen view plus current proposals, nil until joined. Called by OnTick at Δ after unlocking.
func (st *Store) voteHead(slot primitives.Slot, now time.Time) *Node {
	if !st.joined(slot) {
		return nil
	}
	return st.selectHead(st.voterBlocks(slot), st.voterView(slot), now)
}

// Head ticks to now, then returns the public head. Previous-slot votes before 2Δ, current-slot votes after. No joined check.
func (st *Store) Head(now time.Time) *Node {
	st.OnTick(now)
	return st.head(now)
}

func (st *Store) head(now time.Time) *Node {
	s := st.slotAt(now)
	if s > 0 && now.Before(st.confirmationCutoff(s)) {
		s--
	}
	return st.selectHead(everyNode, st.heldView(s), now)
}

// Source ticks to now, then returns the deepest viable fast-confirmed block and the head. Nil source if none qualifies, a slow lock alone never supplies one.
func (st *Store) Source(now time.Time) (source, head *Node) {
	st.OnTick(now)
	return st.source(now)
}

func (st *Store) source(now time.Time) (source, head *Node) {
	fr := st.frame()
	viable := st.viableTree(fr, everyNode)
	// updateConfirmation stores candidates deepest first.
	for _, n := range st.fastConfirmed {
		if st.inTree(n) && viable(n) {
			source = n
			break
		}
	}
	s := st.slotAt(now)
	return source, st.selectHead(everyNode, st.heldView(s), now)
}

// Vote ticks to now, then returns the head captured for slot's AC vote at Δ. Nil if none or the block left the tree.
func (st *Store) Vote(slot primitives.Slot, now time.Time) *Node {
	st.OnTick(now)
	if st.duty.voteSlot != slot || !st.inTree(st.duty.vote) {
		return nil
	}
	return st.duty.vote
}

// FGInputs ticks to now, then returns the source and head captured at slot's action time. Nil without a source or once either left the tree.
func (st *Store) FGInputs(slot primitives.Slot, now time.Time) (source, head *Node) {
	st.OnTick(now)
	d := st.duty
	if d.fgSlot != slot || !st.inTree(d.source) || !st.inTree(d.head) {
		return nil, nil
	}
	return d.source, d.head
}
