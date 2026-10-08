package decoupled

import (
	"errors"
	"slices"
	"time"

	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
)

// New initializes the store at a trusted finalized root and begins observation at now.
// root.FG.Finalized.Height supplies the initial justified and finalized heights.
func New(committeeSize uint64, genesis time.Time, root Block, now time.Time) (*Store, error) {
	if committeeSize == 0 {
		return nil, errors.New("committee size must be positive")
	}
	g := &Node{Root: root.Root, Slot: root.Slot, FG: root.FG}
	return &Store{
		committeeSize:   committeeSize,
		genesis:         genesis,
		restartTime:     now,
		nodes:           map[[32]byte]*Node{root.Root: g},
		finalized:       g,
		justified:       g,
		hJ:              root.FG.Finalized.Height,
		hF:              root.FG.Finalized.Height,
		hMax:            root.FG.Target.Height,
		pool:            make(map[primitives.Slot]*bucket),
		confirmations:   make(map[primitives.Slot]*Node),
		latestConfirmed: g,
	}, nil
}

// Node returns a retained block, or nil.
func (st *Store) Node(root [32]byte) *Node { return st.nodes[root] }

// HasNode reports whether root is retained.
func (st *Store) HasNode(root [32]byte) bool { return st.nodes[root] != nil }

// inTree rejects stale pointers, even if the same root has been reinserted.
func (st *Store) inTree(n *Node) bool { return n != nil && st.nodes[n.Root] == n }

// isAncestor includes equality and requires non-nil a.
func isAncestor(a, b *Node) bool {
	for b != nil && b.Slot > a.Slot {
		b = b.Parent
	}
	return a == b
}

// IsAncestor requires both roots to be retained and includes equality.
func (st *Store) IsAncestor(ancestor, descendant [32]byte) bool {
	a, b := st.nodes[ancestor], st.nodes[descendant]
	return a != nil && b != nil && isAncestor(a, b)
}

// commonAncestor requires non-nil nodes in the same tree.
func commonAncestor(a, b *Node) *Node {
	for a != b {
		if a.Slot > b.Slot {
			a = a.Parent
		} else {
			b = b.Parent
		}
	}
	return a
}

// prune retains F and its descendants. Cached pointers must be checked with inTree.
func (st *Store) prune() {
	f := st.finalized
	f.Parent = nil
	kept := make(map[[32]byte]*Node, len(st.nodes))
	blocks := []*Node{f}
	for i := 0; i < len(blocks); i++ {
		kept[blocks[i].Root] = blocks[i]
		blocks = append(blocks, blocks[i].Children...)
	}
	st.nodes = kept
}

// Remove ticks to now, deletes subtrees and rebuilds J and hMax without moving F.
// Removing F fails, removals already made by this call are not rolled back.
func (st *Store) Remove(roots [][32]byte, now time.Time) error {
	st.OnTick(now)
	for _, r := range roots {
		n := st.nodes[r]
		if n == nil {
			continue
		}
		if n == st.finalized {
			return errors.New("cannot remove the finalized block")
		}
		if p := n.Parent; p != nil {
			p.Children = slices.DeleteFunc(p.Children, func(c *Node) bool { return c == n })
		}
		st.drop(n)
	}
	st.justified, st.hJ, st.hMax = st.finalized, st.hF, st.finalized.FG.Target.Height
	for _, n := range st.nodes {
		st.hMax = max(st.hMax, n.FG.Target.Height)
		if j := st.nodes[n.FG.Justified.Root]; j != nil && (Pair{Height: st.hJ, Root: st.justified.Root}).less(n.FG.Justified) {
			st.justified, st.hJ = j, n.FG.Justified.Height
		}
	}
	return nil
}

func (st *Store) drop(n *Node) {
	delete(st.nodes, n.Root)
	for _, c := range n.Children {
		st.drop(c)
	}
}

// AncestorAtSlot returns the last retained ancestor at or before slot.
// It returns false for unknown roots or ancestors pruned below F.
func (st *Store) AncestorAtSlot(root [32]byte, slot primitives.Slot) ([32]byte, bool) {
	n := st.nodes[root]
	for n != nil && n.Slot > slot {
		n = n.Parent
	}
	if n == nil {
		return [32]byte{}, false
	}
	return n.Root, true
}
