package protoarray

import (
	"bytes"
	"errors"
	"fmt"
	"math"

	"github.com/prysmaticlabs/prysm/shared/params"
)

// insert registers a new block node to the fork choice store.
// It updates the new node's parent with best child and descendant node.
func (s *Store) insert(slot uint64, root [32]byte, parent [32]byte, justifiedEpoch uint64, finalizedEpoch uint64) error {
	s.nodeIndicesLock.Lock()
	defer s.nodeIndicesLock.Unlock()

	// Return if the block has previously been inserted.
	if _, ok := s.nodeIndices[root]; ok {
		return nil
	}

	index := len(s.nodes)
	parentIndex, ok := s.nodeIndices[parent]
	// Mark genesis block's parent as non existent.
	if !ok {
		parentIndex = nonExistentNode
	}

	n := Node{
		slot:           slot,
		root:           root,
		parent:         parentIndex,
		justifiedEpoch: justifiedEpoch,
		finalizedEpoch: finalizedEpoch,
		bestChild:      nonExistentNode,
		bestDescendant: nonExistentNode,
		weight:         0,
	}

	s.nodeIndices[root] = uint64(index)
	s.nodes = append(s.nodes, n)

	if n.parent != nonExistentNode {
		if err := s.updateBestChildAndDescendant(parentIndex, uint64(index)); err != nil {
			return err
		}
	}

	return nil
}

// head starts from justifiedRoot and then follows the best descendant links to find the best block for head.
func (s *Store) head(justifiedRoot [32]byte) ([32]byte, error) {
	justifiedIndex, ok := s.nodeIndices[justifiedRoot]
	if !ok {
		return [32]byte{}, errors.New("unknown justified root")
	}
	if int(justifiedIndex) >= len(s.nodes) {
		return [32]byte{}, errors.New("invalid justified index")
	}

	justifiedNode := s.nodes[justifiedIndex]
	bestDescendantIndex := justifiedNode.bestDescendant
	if bestDescendantIndex == nonExistentNode {
		bestDescendantIndex = justifiedIndex
	}
	if bestDescendantIndex >= uint64(len(s.nodes)) {
		return [32]byte{}, errors.New("invalid best descendant index")
	}

	bestNode := s.nodes[bestDescendantIndex]
	if !s.viableForHead(&bestNode) {
		return [32]byte{}, errors.New("best node not viable for head")
	}

	return bestNode.root, nil
}

// updateBestChildAndDescendant updates parent node's best child and descendent.
// It looks at parent node and child node and potentially modifies parent's best
// child and best descendent values.
// There are four outcomes:
// - The child is already the best child but it's now invalid due to a FFG change and should be removed.
// - The child is already the best child and the parent is updated with the new best descendant.
// - The child is not the best child but becomes the best child.
// - The child is not the best child and does not become best child.
func (s *Store) updateBestChildAndDescendant(parentIndex uint64, childIndex uint64) error {
	parent := s.nodes[parentIndex]
	child := s.nodes[childIndex]
	childLeadsToViableHead, err := s.leadsToViableHead(&child)
	if err != nil {
		return err
	}

	// Define 3 variables for the 3 options mentioned above. This is to
	// set `parent.bestChild` and `parent.bestDescendent` to. These
	// aliases are to assist readability.
	changeToNone := []uint64{nonExistentNode, nonExistentNode}
	bestDescendant := child.bestDescendant
	if bestDescendant == nonExistentNode {
		bestDescendant = childIndex
	}
	changeToChild := []uint64{childIndex, bestDescendant}
	noChange := []uint64{parent.bestChild, parent.bestDescendant}
	newParentChild := make([]uint64, 0)

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
			bestChild := &s.nodes[parent.bestChild]
			bestChildLeadsToViableHead, err := s.leadsToViableHead(bestChild)
			if err != nil {
				return err
			}

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

	return nil
}

// applyScoreChanges iterates backwards through the stored nodes. It checks all nodes parents
// and potential best child. For each node, it updates the weight with input delta and
// back propagate each nodes delta to its parents delta. The best child is then updated
// along with best descendant.
func (s *Store) applyScoreChanges(justifiedEpoch uint64, finalizedEpoch uint64, delta []int) error {
	if len(s.nodeIndices) != len(delta) {
		return fmt.Errorf("node indices length diff than delta length, %d != %d", len(s.nodeIndices), len(delta))
	}
	if len(s.nodes) != len(delta) {
		return fmt.Errorf("nodes length diff than delta length, %d != %d", len(s.nodes), len(delta))
	}

	if s.justifiedEpoch != justifiedEpoch || s.finalizedEpoch != finalizedEpoch {
		s.justifiedEpoch = justifiedEpoch
		s.finalizedEpoch = finalizedEpoch

	}

	// Iterate backwards through all indices store nodes.
	for i := len(s.nodes) - 1; i >= 0; i-- {
		n := s.nodes[i]

		// There is no need to adjust the balances or manage parent of the zero hash since it
		// is an alias to the genesis block.
		if n.root == params.BeaconConfig().ZeroHash {
			continue
		}

		nodeDelta := delta[i]

		if nodeDelta < 0 {
			if int(n.weight)+nodeDelta < 0 {
				n.weight = 0
			} else {
				n.weight -= uint64(math.Abs(float64(nodeDelta)))
			}
		} else {
			n.weight += uint64(nodeDelta)
		}
		s.nodes[i] = n

		// Update parent's best child and descendent if the node has a known parent.
		if n.parent != nonExistentNode {
			// Back propagate the nodes delta to its parent.
			if int(n.parent) >= len(delta) {
				return errors.New("invalid parent index")
			}

			delta[n.parent] += nodeDelta
			if err := s.updateBestChildAndDescendant(n.parent, uint64(i)); err != nil {
				return err
			}
		}
	}

	return nil
}

// pruneBeforeFinalized prunes the store with the new finalization information. The tree is only
// pruned if the supplied finalized epoch and root are different than current store value and
// the number of the nodes in store has met prune threshold.
func (s *Store) pruneBeforeFinalized(finalizedRoot [32]byte, finalizedEpoch uint64) error {
	s.nodeIndicesLock.Lock()
	defer s.nodeIndicesLock.Unlock()

	if finalizedEpoch < s.finalizedEpoch {
		return fmt.Errorf("reverted finalized epoch %d <= %d", finalizedEpoch, s.finalizedEpoch)
	} else if finalizedEpoch != s.finalizedEpoch {
		s.finalizedEpoch = finalizedEpoch
	}

	finalizedIndex, ok := s.nodeIndices[finalizedRoot]
	if !ok {
		return errors.New("finalized node unknown")
	}

	// The number of the nodes has not met the prune threshold.
	// Pruning at small numbers incurs more cost than benefit.
	if finalizedIndex < s.pruneThreshold {
		return nil
	}

	// Remove the key/values from indices mapping on to be deleted nodes.
	for i := uint64(0); i < finalizedIndex; i++ {
		if int(i) >= len(s.nodes) {
			return errors.New("invalid node index")
		}
		delete(s.nodeIndices, s.nodes[i].root)
	}

	// Discard all the nodes before finalization.
	if int(finalizedIndex) >= len(s.nodes) {
		return errors.New("invalid finalized index")
	}
	s.nodes = s.nodes[finalizedIndex:]

	// Adjust indices to node mapping.
	for k, v := range s.nodeIndices {
		s.nodeIndices[k] = v - finalizedIndex
	}

	// Iterate through existing nodes and adjust its parent/child indices with the new layout.
	for i, node := range s.nodes {
		if node.parent != nonExistentNode {
			// If the node's parent is less than finalized index, set it to non existent.
			if node.parent >= finalizedIndex {
				node.parent -= finalizedIndex
			} else {
				node.parent = nonExistentNode
			}
		}
		if node.bestChild != nonExistentNode {
			if node.bestChild < finalizedIndex {
				return errors.New("invalid best child index")
			}
			node.bestChild -= finalizedIndex
		}
		if node.bestDescendant != nonExistentNode {
			if node.bestDescendant < finalizedIndex {
				return errors.New("invalid best descendant index")
			}
			node.bestDescendant -= finalizedIndex
		}

		s.nodes[i] = node
	}
	return nil
}

// leadsToViableHead returns true if the node or the best descendent of the node is viable for head.
// Any node with diff finalized or justified epoch than the ones in fork choice store
// should not be viable to head.
func (s *Store) leadsToViableHead(node *Node) (bool, error) {
	var bestDescendentViable bool
	bestDescendentIndex := node.bestDescendant
	if bestDescendentIndex != nonExistentNode {
		bestDescendentNode := &s.nodes[bestDescendentIndex]
		bestDescendentViable = s.viableForHead(bestDescendentNode)
	}

	return bestDescendentViable || s.viableForHead(node), nil
}

// viableForHead returns true if the node is viable to head.
// Any node with diff finalized or justified epoch than the ones in fork choice store
// should not be viable to head.
func (s *Store) viableForHead(node *Node) bool {
	justified := s.justifiedEpoch == node.justifiedEpoch || s.justifiedEpoch == 0
	finalized := s.finalizedEpoch == node.finalizedEpoch || s.finalizedEpoch == 0
	return justified && finalized
}
