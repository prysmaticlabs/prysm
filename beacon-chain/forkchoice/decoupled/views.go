package decoupled

import "github.com/OffchainLabs/prysm/v7/consensus-types/primitives"

// dedupe preserves first-seen order and retains equivocations.
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

// voterView merges the 3*delta frozen view with current proposals.
// All carried votes prove participation; only Supported votes add support.
func (st *Store) voterView(s primitives.Slot) view {
	if s == 0 {
		return view{}
	}
	prev := s - 1
	cutoff := st.supportCutoff(prev)
	var v view
	if b := st.pool[prev]; b != nil {
		for i, u := range b.votes {
			if b.at[i].Before(cutoff) {
				v.participants = append(v.participants, u)
			}
			if at, ok := st.processedAt(u); ok && at.Before(cutoff) {
				v.supported = append(v.supported, u)
			}
		}
	}
	for _, p := range st.nodes {
		if p.Slot != s {
			continue
		}
		v.participants = append(v.participants, p.Votes...)
		for _, u := range p.Supported {
			if st.block(u) != nil {
				v.supported = append(v.supported, u)
			}
		}
	}
	v.participants = dedupe(v.participants)
	v.supported = dedupe(v.supported)
	return v
}

// voterBlocks adds current proposals and their ancestors to the frozen tree.
// Parent processing precedes child processing, so the first frozen ancestor ends the walk.
func (st *Store) voterBlocks(s primitives.Slot) func(*Node) bool {
	if s == 0 {
		return func(*Node) bool { return false }
	}
	cutoff := st.supportCutoff(s - 1)
	late := map[*Node]bool{}
	for _, p := range st.nodes {
		if p.Slot != s {
			continue
		}
		for n := p; n != nil && !n.processed.Before(cutoff) && !late[n]; n = n.Parent {
			late[n] = true
		}
	}
	return func(n *Node) bool { return n.processed.Before(cutoff) || late[n] }
}

// heldView uses all stored votes for s, without the voter's freeze cutoff.
func (st *Store) heldView(s primitives.Slot) view {
	v := view{participants: st.votes(s)}
	for _, u := range v.participants {
		if st.block(u) != nil {
			v.supported = append(v.supported, u)
		}
	}
	return v
}
