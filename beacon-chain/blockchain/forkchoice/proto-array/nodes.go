package proto_array

import "bytes"

// Insert registers a new block with the fork choice.
func (s Store) Insert(root [32]byte, parent [32]byte, justifiedEpoch uint64, finalizedEpoch uint64) {
	s.nodeIndicesLock.Lock()
	defer s.nodeIndicesLock.Unlock()

	index := len(s.nodes)
	parentIndex, ok := s.nodeIndices[parent]
	// Mark genesis block's parent as non existent.
	if !ok {
		parentIndex = nonExistentNode
	}

	n := Node{
		root: root,
		parent: parentIndex,
		justifiedEpoch: justifiedEpoch,
		finalizedEpoch: finalizedEpoch,
		bestChild: nonExistentNode,
		bestDescendant: nonExistentNode,
		weight: 0,
	}

	s.nodeIndices[root] = uint64(index)
	s.nodes = append(s.nodes, n)

	if n.parent != nonExistentNode {
		s.UpdateBestChildAndDescendant(parentIndex, uint64(index))
	}
}

// UpdateBestChildAndDescendant updates parent node's best child and descendent.
// It looks at parent node and child node and potentially modifies parent's best
// child and best descendent values.
// There are four outcomes:
// - The child is already the best child but it's now invalid due to a FFG change and should be removed.
// - The child is already the best child and the parent is updated with the new best descendant.
// - The child is not the best child but becomes the best child.
// - The child is not the best child and does not become best child.
func (s Store) UpdateBestChildAndDescendant(parentIndex uint64, childIndex uint64) {
	parent := s.nodes[parentIndex]
	child := s.nodes[childIndex]

	childLeadsToViableHead := s.LeadsToViableHead(child)

	// Define 3 variables for the 3 options mentioned above. This is to
	// set `parent.bestChild` and `parent.bestDescendent` to. These
	// aliases are to assist readability.
	changeToNone := []uint64{nonExistentNode, nonExistentNode}
	changeToChild := []uint64{childIndex, child.bestDescendant}
	noChange := []uint64{parent.bestChild, parent.bestDescendant}
	newParentChild := make([]uint64,0)

	if parent.bestChild != nonExistentNode {
		if parent.bestChild == childIndex && !childLeadsToViableHead {
			// If the child is already the best child of the parent but it's not viable for head,
			// we should remove it.
			newParentChild = changeToNone
		} else if parent.bestChild == childIndex {
			// If the child is already the best child of the parent, set it again to ensure best
			// descendent of the parent is updated.
			newParentChild = changeToChild
		} else {
			bestChild := s.nodes[parent.bestChild]
			bestChildLeadsToViableHead := s.LeadsToViableHead(bestChild)

			if childLeadsToViableHead && !bestChildLeadsToViableHead {
				// The child leads to a viable head, but the current best child doesnt.
				newParentChild = changeToChild
			} else if !childLeadsToViableHead && bestChildLeadsToViableHead {
				// The best child leads to viable head, but the child doesnt.
				newParentChild = noChange
			} else if child.weight == bestChild.weight {
				// Tie-breaker of equal weights by root.
				if bytes.Compare(child.root[:], bestChild.root[:]) > 0 {
					newParentChild = changeToChild
				} else {
					newParentChild = noChange
				}
			} else {
				// Choose winner by weight.
				if child.weight > bestChild.weight {
					newParentChild = changeToChild
				} else {
					newParentChild = noChange
				}
			}
		}
	} else {
		if childLeadsToViableHead {
			// There's no current best child and the child is viable.
			newParentChild = changeToChild
		} else {
			// There's no current best child and the child is not viable.
			newParentChild = noChange
		}
	}

	parent.bestChild = newParentChild[0]
	parent.bestDescendant = newParentChild[1]
	s.nodes[parentIndex] = parent
}

func (s Store) LeadsToViableHead(node Node) bool {
	bestDescendentIndex := node.bestDescendant
	bestDescendentNode := s.nodes[bestDescendentIndex]
	return s.ViableForHead(bestDescendentNode) || s.ViableForHead(node)
}

func (s Store) ViableForHead(node Node) bool {
	justified := s.justifiedEpoch == node.justifiedEpoch || s.justifiedEpoch == 0
	finalized := s.finalizedEpoch == node.finalizedEpoch || s.finalizedEpoch == 0
	return justified && finalized
}
