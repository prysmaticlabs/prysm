package decoupled

import (
	"bytes"
	"errors"
	"time"

	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
)

// ErrUnknownParent indicates that the block's parent is not in the store.
var ErrUnknownParent = errors.New("block parent is not in the store")

// InsertNode accepts a validated block and records its votes at now, after due ticks.
// It checks only parent presence; duplicate roots are ignored.
func (st *Store) InsertNode(b Block, now time.Time) error {
	st.OnTick(now)
	if st.nodes[b.Root] != nil {
		return nil
	}
	parent := st.nodes[b.Parent]
	if parent == nil {
		return ErrUnknownParent
	}
	n := &Node{Root: b.Root, Slot: b.Slot, Parent: parent, Votes: b.Votes, Supported: b.Supported, FG: b.FG, processed: now}
	parent.Children = append(parent.Children, n)
	st.nodes[b.Root] = n
	st.updateFinality(n)
	for _, v := range b.Votes {
		st.addVote(v, now)
	}
	return nil
}

func (p Pair) less(q Pair) bool {
	if p.Height != q.Height {
		return p.Height < q.Height
	}
	return bytes.Compare(p.Root[:], q.Root[:]) < 0
}

// updateFinality applies height and justification changes before checking the
// claimed finalization's viability. Accepting finalization prunes the tree.
func (st *Store) updateFinality(n *Node) {
	st.hMax = max(st.hMax, n.FG.Target.Height)
	j := st.nodes[n.FG.Justified.Root]
	current := Pair{Height: st.hJ, Root: st.justified.Root}
	if j != nil && current.less(n.FG.Justified) {
		st.justified, st.hJ = j, n.FG.Justified.Height
	}
	f := st.nodes[n.FG.Finalized.Root]
	if f != nil && f != st.finalized && isAncestor(f, st.justified) && st.hasViableDescendant(f) {
		st.finalized = f
		st.prune()
	}
}

// Finalized returns accepted F, which may lag finalization claimed by block states.
func (st *Store) Finalized() *Node { return st.finalized }

// Justified returns J, selected by lexicographic (height, root) order.
func (st *Store) Justified() *Node { return st.justified }

// JustifiedHeight returns the FG height of J.
func (st *Store) JustifiedHeight() primitives.Height { return st.hJ }
