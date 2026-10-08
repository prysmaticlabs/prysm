package decoupled

import "github.com/OffchainLabs/prysm/v7/consensus-types/primitives"

// Spec: the (votes, support_votes) pair a duty reads. Participants set the denominator and the equivocation test, supported votes count for a block.
type view struct {
	participants []Vote
	supported    []Vote
}

func dedupe(votes []Vote) []Vote {
	seen := make(map[Vote]struct{}, len(votes))
	out := make([]Vote, 0, len(votes))
	for _, v := range votes {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func (st *Store) blocksAtSlot(s primitives.Slot) []*Node {
	var out []*Node
	for _, n := range st.nodes {
		if n.Slot == s {
			out = append(out, n)
		}
	}
	return out
}

// Spec: voter_view and voter_support_view for the slot-s vote. A block vote is supported when its root resolves, there is no proposer-fixed subset on the wire.
func (st *Store) voterView(s primitives.Slot) view {
	if s == 0 {
		return view{}
	}
	prev := s - 1
	cutoff := st.supportCutoff(prev)
	var v view
	for _, e := range st.pool[prev] {
		if e.at.Before(cutoff) {
			v.participants = append(v.participants, e.vote)
		}
		if at, ok := st.resolutionTime(e.vote); ok && at.Before(cutoff) {
			v.supported = append(v.supported, e.vote)
		}
	}
	for _, b := range st.blocksAtSlot(s) {
		for _, u := range b.BlockVotes {
			if u.Slot != prev {
				continue
			}
			v.participants = append(v.participants, u)
			if st.resolve(u) != nil {
				v.supported = append(v.supported, u)
			}
		}
	}
	v.participants = dedupe(v.participants)
	v.supported = dedupe(v.supported)
	return v
}

// Spec: voter_processed_block_tree. Blocks processed before the previous support cutoff plus every ancestor of a slot-s proposal.
func (st *Store) voterBlocks(s primitives.Slot) map[*Node]bool {
	in := make(map[*Node]bool, len(st.nodes))
	if s == 0 {
		return in
	}
	cutoff := st.supportCutoff(s - 1)
	for _, n := range st.nodes {
		if n.processed.Before(cutoff) {
			in[n] = true
		}
	}
	for _, p := range st.blocksAtSlot(s) {
		for n := p; n != nil && !in[n]; n = n.Parent {
			in[n] = true
		}
	}
	return in
}

// Spec: proposer_view and proposer_support_view, the whole previous-slot bucket with no cutoff.
func (st *Store) proposerView(s primitives.Slot) view {
	if s == 0 {
		return view{}
	}
	var v view
	for _, e := range st.pool[s-1] {
		v.participants = append(v.participants, e.vote)
		if st.resolve(e.vote) != nil {
			v.supported = append(v.supported, e.vote)
		}
	}
	return v
}
