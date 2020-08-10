package protoarray

// Slot of the fork choice node.
func (n *Node) Slot() uint64 {
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
func (n *Node) JustifiedEpoch() uint64 {
	return n.justifiedEpoch
}

// FinalizedEpoch of the fork choice node.
func (n *Node) FinalizedEpoch() uint64 {
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

// Graffiti of the fork choice node.
func (n *Node) Graffiti() [32]byte {
	return n.graffiti
}
