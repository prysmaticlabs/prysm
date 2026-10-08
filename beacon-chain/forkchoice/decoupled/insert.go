package decoupled

import (
	"bytes"
	"errors"

	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
)

var ErrUnknownParent = errors.New("block parent is not in the store")

// The fields InsertNode reads until the Decoupled state and block getters exist.
type Block struct {
	Root       [32]byte
	Parent     [32]byte
	Slot       primitives.Slot
	BlockVotes []Vote
	FG         FGSummary
}

// Spec: on_block. Sync and the state transition already validated the block, only the parent must be a node.
func (st *Store) InsertNode(b Block) error {
	if st.nodes[b.Root] != nil {
		return nil
	}
	parent := st.nodes[b.Parent]
	if parent == nil {
		return ErrUnknownParent
	}
	now := st.now()
	n := st.addChild(b.Root, parent, b.Slot)
	n.processed = now
	n.BlockVotes = b.BlockVotes
	n.FG = b.FG
	for _, v := range b.BlockVotes {
		st.addVote(v, now)
	}
	st.updateFinality(n)
	return nil
}

// Spec: the (h_j, J.root) lexicographic order, greatest root wins a tie.
func (p Pair) less(q Pair) bool {
	if p.Height != q.Height {
		return p.Height < q.Height
	}
	return bytes.Compare(p.Root[:], q.Root[:]) < 0
}

// Spec: update_finality. Three statements in order, each reading the store the previous one wrote.
func (st *Store) updateFinality(n *Node) {
	st.hMax = max(st.hMax, n.FG.Target.Height)
	j := st.nodes[n.FG.Justified.Root]
	current := Pair{Height: st.hJ, Root: st.justified.Root}
	if j != nil && isAncestor(st.finalized, j) && current.less(n.FG.Justified) {
		st.justified, st.hJ = j, n.FG.Justified.Height
	}
	f := st.nodes[n.FG.Finalized.Root]
	if f != nil && f != st.finalized && isAncestor(st.finalized, f) && isAncestor(f, st.justified) && st.viable(f, everyNode) {
		st.finalized = f
		st.prune()
	}
}

func (st *Store) prune() {}

func (st *Store) Finalized() *Node                   { return st.finalized }
func (st *Store) Justified() *Node                   { return st.justified }
func (st *Store) JustifiedHeight() primitives.Height { return st.hJ }
func (st *Store) MaxHeight() primitives.Height       { return st.hMax }
