package decoupled

import (
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/time/slots"
)

// frame normally uses F with floor hMax-1. At hMax == hJ+1, use J instead,
// unless a competing branch has also reached hMax. That contest uses F and
// raises the floor to hMax.
func (st *Store) frame() frame {
	floor := st.hMax
	if floor > 0 {
		floor--
	}
	if st.hMax != st.hJ+1 {
		return frame{root: st.finalized, floor: floor}
	}
	for _, n := range st.nodes {
		// A withheld justification must not displace an already advanced branch.
		if n.FG.Target.Height == st.hMax && !isAncestor(st.justified, n) {
			return frame{root: st.finalized, floor: st.hMax}
		}
	}
	// The floor alone cannot protect a hidden finalization at this height.
	return frame{root: st.justified, floor: floor}
}

// viableHead checks the height floor; ancestry is checked by viableTree.
func (fr frame) viableHead(n *Node) bool { return n.FG.Target.Height >= fr.floor }

// hasViableDescendant checks the height requirement for accepting finalization.
// The block itself counts as a descendant.
func (st *Store) hasViableDescendant(b *Node) bool {
	floor := st.frame().floor
	blocks := []*Node{b}
	for i := 0; i < len(blocks); i++ {
		if blocks[i].FG.Target.Height >= floor {
			return true
		}
		blocks = append(blocks, blocks[i].Children...)
	}
	return false
}

// viableTree keeps paths from the FG root to heads meeting the height floor.
// The input view must be ancestor-closed; descendants outside it do not count.
func (st *Store) viableTree(fr frame, in func(*Node) bool) func(*Node) bool {
	viable := make(map[*Node]bool, len(st.nodes))
	var walk func(n *Node, under bool) bool
	walk = func(n *Node, under bool) bool {
		if !in(n) {
			return false
		}
		under = under || n == fr.root
		v := under && fr.viableHead(n)
		for _, c := range n.Children {
			v = walk(c, under) || v
		}
		viable[n] = v && under
		return v
	}
	walk(st.finalized, false)
	return func(n *Node) bool { return viable[n] }
}

func everyNode(*Node) bool { return true }

// eligible waives the majority requirement when the parent is below the height
// floor, for current-slot proposals, and for pre-fork blocks.
func (fr frame) eligible(c *supportCounts, current primitives.Slot) func(*Node) bool {
	return func(b *Node) bool {
		return b.Parent == nil || !fr.viableHead(b.Parent) || b.Slot == current || slots.ToEpoch(b.Slot) < params.BeaconConfig().DecoupledForkEpoch || c.hasMajority(b)
	}
}
