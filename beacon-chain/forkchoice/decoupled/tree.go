package decoupled

import (
	"time"

	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
)

// Spec: Store.init. Genesis carries the initial chain state, so h_max starts at 1.
func New(genesisRoot [32]byte) *Store {
	g := &Node{
		Root: genesisRoot,
		FG: FGSummary{
			Target:    Pair{Height: 1, Root: genesisRoot},
			Justified: Pair{Root: genesisRoot},
			Finalized: Pair{Root: genesisRoot},
		},
	}
	return &Store{
		now:             time.Now,
		root:            g,
		nodes:           map[[32]byte]*Node{genesisRoot: g},
		pool:            make(map[primitives.Slot][]poolEntry),
		finalized:       g,
		justified:       g,
		hMax:            1,
		liveConfirmed:   g,
		latestConfirmed: g,
	}
}

func (st *Store) SetGenesisTime(genesis time.Time) { st.genesis = genesis }

func (st *Store) Node(root [32]byte) *Node { return st.nodes[root] }

func (st *Store) HasNode(root [32]byte) bool { return st.nodes[root] != nil }

func (st *Store) addChild(root [32]byte, parent *Node, slot primitives.Slot) *Node {
	n := &Node{Root: root, Slot: slot, Parent: parent}
	parent.Children = append(parent.Children, n)
	st.nodes[root] = n
	return n
}

// Spec: Block.preceq, a ⪯ b. Parents have strictly lower slots, so the walk stops at a's slot.
func isAncestor(a, b *Node) bool {
	for b != nil && b.Slot > a.Slot {
		b = b.Parent
	}
	return a == b
}

func (st *Store) IsAncestor(ancestor, descendant [32]byte) bool {
	a, b := st.nodes[ancestor], st.nodes[descendant]
	return a != nil && b != nil && isAncestor(a, b)
}
