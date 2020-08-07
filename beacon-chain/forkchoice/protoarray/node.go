package protoarray

// Slot of the fork choice node.
func (n *Node) Slot() uint64 {
	return n.slot
}

// Root of the fork choice node.
func (n *Node) Root() [32]byte {
	return n.root
}

// parent of the fork choice node.
func (n *Node) Parent() uint64 {
	return n.parent
}

// justifiedEpoch of the fork choice node.
func (n *Node) JustifiedEpoch() uint64 {
	return n.justifiedEpoch
}

// finalizedEpoch of the fork choice node.
func (n *Node) FinalizedEpoch() uint64 {
	return n.finalizedEpoch
}

// weight of the fork choice node.
func (n *Node) Weight() uint64 {
	return n.weight
}

// bestChild of the fork choice node.
func (n *Node) BestChild() uint64 {
	return n.bestChild
}

// bestDescendant of the fork choice node.
func (n *Node) BestDescendant() uint64 {
	return n.bestDescendant
}

// graffiti of the fork choice node.
func (n *Node) Graffiti() [32]byte {
	return n.graffiti
}
