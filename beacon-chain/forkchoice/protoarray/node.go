package protoarray

import (
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
)

// Slot of the fork choice node.
func (n *Node) Slot() types.Slot {
	return n.slot
}

// Root of the fork choice node.
func (n *Node) Root() [32]byte {
	return n.root
}

// Parent of the fork choice node.
func (n *Node) Parent() uint64 {
	return n.parent
}

// JustifiedEpoch of the fork choice node.
func (n *Node) JustifiedEpoch() types.Epoch {
	return n.justifiedEpoch
}

// FinalizedEpoch of the fork choice node.
func (n *Node) FinalizedEpoch() types.Epoch {
	return n.finalizedEpoch
}

// Weight of the fork choice node.
func (n *Node) Weight() uint64 {
	return n.weight
}

// BestChild of the fork choice node.
func (n *Node) BestChild() uint64 {
	return n.bestChild
}

// BestDescendant of the fork choice node.
func (n *Node) BestDescendant() uint64 {
	return n.bestDescendant
}
