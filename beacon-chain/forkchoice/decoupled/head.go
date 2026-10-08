package decoupled

import "github.com/OffchainLabs/prysm/v7/consensus-types/primitives"

// Spec: goldfish_fork_choice, the composed head over the candidate tree inside the given block view.
func (st *Store) selectHead(in func(*Node) bool, v view, voteSlot primitives.Slot) *Node {
	counts := st.countSupport(v, voteSlot)
	return ghost(st.anchor(), st.candidates(in), counts.score, st.eligible(counts, st.currentSlot()))
}

// Spec: the proposal's head, the proposer view over the full tree.
func (st *Store) ProposalHead(slot primitives.Slot) *Node {
	return st.selectHead(everyNode, st.proposerView(slot), slot-1)
}

// Spec: B.votes, every previous-slot vote the proposer holds, with no cutoff applied.
func (st *Store) BlockVotes(slot primitives.Slot) []Vote {
	if slot == 0 {
		return nil
	}
	return st.poolVotes(slot - 1)
}

// Spec: goldfish_vote. Only the vote duty restricts the walk to the voter's block view.
func (st *Store) VoteHead(slot primitives.Slot) *Node {
	in := st.voterBlocks(slot)
	return st.selectHead(func(n *Node) bool { return in[n] }, st.voterView(slot), slot-1)
}

// Before the vote time nobody has voted in this slot, so the previous slot's full bucket is the freshest evidence. After it, this is the head read of get_fg_vote.
func (st *Store) Head() *Node {
	slot := st.currentSlot()
	if st.now().Before(st.voteTime(slot)) {
		return st.ProposalHead(slot)
	}
	v := view{participants: st.poolVotes(slot)}
	for _, u := range v.participants {
		if st.resolve(u) != nil {
			v.supported = append(v.supported, u)
		}
	}
	return st.selectHead(everyNode, v, slot)
}
