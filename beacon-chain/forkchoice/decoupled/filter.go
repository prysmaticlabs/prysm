package decoupled

import "github.com/OffchainLabs/prysm/v7/consensus-types/primitives"

// Spec: h_max - 1 truncated at zero.
func (st *Store) viabilityBound() primitives.Height {
	if st.hMax == 0 {
		return 0
	}
	return st.hMax - 1
}

// Spec: viable, some processed descendant inside the view has height at least h_max - 1. Views are ancestor closed because a block is processed only after its parent, so an excluded subtree holds no witness.
func (st *Store) viable(b *Node, in func(*Node) bool) bool {
	bound := st.viabilityBound()
	stack := []*Node{b}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if !in(n) {
			continue
		}
		if n.FG.Target.Height >= bound {
			return true
		}
		stack = append(stack, n.Children...)
	}
	return false
}

// Spec: get_fg_root, J when h_max = h_j + 1, otherwise F.
func (st *Store) fgRoot() *Node {
	if st.hMax == st.hJ+1 {
		return st.justified
	}
	return st.finalized
}

// Spec: get_filtered_block_tree_from. Membership and viability are both recomputed inside the supplied view.
func (st *Store) candidates(in func(*Node) bool) func(*Node) bool {
	fgRoot := st.fgRoot()
	memo := make(map[*Node]bool)
	return func(b *Node) bool {
		if v, ok := memo[b]; ok {
			return v
		}
		v := in(b) && isAncestor(st.finalized, b) && isAncestor(fgRoot, b) && st.viable(b, in)
		memo[b] = v
		return v
	}
}

// Spec: goldfish_eligible. The majority gate applies only when the parent sits in the band {h_max-1, h_max}, and a current-slot block is always eligible.
func (st *Store) eligible(sc *supportCounts, current primitives.Slot) func(*Node) bool {
	bound := st.viabilityBound()
	return func(b *Node) bool {
		parentHeight := b.FG.Target.Height
		if b.Parent != nil {
			parentHeight = b.Parent.FG.Target.Height
		}
		return parentHeight < bound || sc.hasMajority(b) || b.Slot == current
	}
}
