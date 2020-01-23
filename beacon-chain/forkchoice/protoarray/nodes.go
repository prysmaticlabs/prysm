package protoarray

import (
	"bytes"
	"context"
	"math"

	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// applyScoreChanges iterates backwards through the nodes in store. It checks all nodes parent
// and its best child. For each node, it updates the weight with input delta and
// back propagate the nodes delta to its parents delta. After scoring changes,
// the best child is then updated along with best descendant.
func (s *Store) applyScoreChanges(ctx context.Context, justifiedEpoch uint64, finalizedEpoch uint64, delta []int) error {
	ctx, span := trace.StartSpan(ctx, "protoArrayForkChoice.applyScoreChanges")
	defer span.End()

	// The length of the node indices can not be different than length of the delta.
	if len(s.nodeIndices) != len(delta) {
		return errInvalidDeltaLength
	}

	// Update the justified / finalized epochs in store if necessary.
	if s.justifiedEpoch != justifiedEpoch || s.finalizedEpoch != finalizedEpoch {
		s.justifiedEpoch = justifiedEpoch
		s.finalizedEpoch = finalizedEpoch
	}

	// Iterate backwards through all index to node in store.
	for i := len(s.nodes) - 1; i >= 0; i-- {
		n := s.nodes[i]

		// There is no need to adjust the balances or manage parent of the zero hash, it
		// is an alias to the genesis block.
		if n.root == params.BeaconConfig().ZeroHash {
			continue
		}

		// Protection against node index out of bound. This should not happen.
		if i >= len(delta) {
			return errInvalidNodeDelta
		}
		nodeDelta := delta[i]

		if nodeDelta < 0 {
			// A node's weight can not be negative but the delta can be negative.
			if int(n.weight)+nodeDelta < 0 {
				n.weight = 0
			} else {
				// Subtract node's weight.
				n.weight -= uint64(math.Abs(float64(nodeDelta)))
			}
		} else {
			// Add node's weight.
			n.weight += uint64(nodeDelta)
		}
		s.nodes[i] = n

		// Update parent's best child and descendent if the node has a known parent.
		if n.parent != nonExistentNode {
			// Protection against node parent index out of bound. This should not happen.
			if int(n.parent) >= len(delta) {
				return errInvalidParentDelta
			}
			// Back propagate the nodes delta to its parent.
			delta[n.parent] += nodeDelta
			if err := s.updateBestChildAndDescendant(ctx, n.parent, uint64(i)); err != nil {
				return err
			}
		}
	}

	return nil
}

// updateBestChildAndDescendant updates parent node's best child and descendent.
// It looks at input parent node and input child node and potentially modifies parent's best
// child and best descendent indices.
// There are four outcomes:
// 1.)  The child is already the best child but it's now invalid due to a FFG change and should be removed.
// 2.)  The child is already the best child and the parent is updated with the new best descendant.
// 3.)  The child is not the best child but becomes the best child.
// 4.)  The child is not the best child and does not become best child.
func (s *Store) updateBestChildAndDescendant(ctx context.Context, parentIndex uint64, childIndex uint64) error {
	ctx, span := trace.StartSpan(ctx, "protoArrayForkChoice.updateBestChildAndDescendant")
	defer span.End()

	// Protection against parent index out of bound, this should not happen.
	if parentIndex >= uint64(len(s.nodes)) {
		return errInvalidNodeIndex
	}
	parent := s.nodes[parentIndex]

	// Protection against child index out of bound, again this should not happen.
	if childIndex >= uint64(len(s.nodes)) {
		return errInvalidNodeIndex
	}
	child := s.nodes[childIndex]

	// Is the child viable to become head? Based on justification and finalization rules.
	childLeadsToViableHead, err := s.leadsToViableHead(ctx, child)
	if err != nil {
		return err
	}

	// Define 3 variables for the 3 outcomes mentioned above. This is to
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
			// we should remove it. (Outcome 1)
			newParentChild = changeToNone
		} else if parent.bestChild == childIndex {
			// If the child is already the best child of the parent, set it again to ensure best
			// descendent of the parent is updated. (Outcome 2)
			newParentChild = changeToChild
		} else {
			// Protection against parent's best child going out of bound.
			if parent.bestChild > uint64(len(s.nodes)) {
				return errInvalidBestDescendantIndex
			}
			bestChild := s.nodes[parent.bestChild]
			// Is current parent's best child viable to be head? Based on justification and finalization rules.
			bestChildLeadsToViableHead, err := s.leadsToViableHead(ctx, bestChild)
			if err != nil {
				return err
			}

			if childLeadsToViableHead && !bestChildLeadsToViableHead {
				// The child leads to a viable head, but the current parent's best child doesnt.
				newParentChild = changeToChild
			} else if !childLeadsToViableHead && bestChildLeadsToViableHead {
				// The child doesn't lead to a viable head, the current parent's best child does.
				newParentChild = noChange
			} else if child.weight == bestChild.weight {
				// If both are viable, compare their weights.
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
			// If parent doesn't have a best child and the child is viable.
			newParentChild = changeToChild
		} else {
			// If parent doesn't have a best child and the child is not viable.
			newParentChild = noChange
		}
	}

	// Update parent with the outcome.
	parent.bestChild = newParentChild[0]
	parent.bestDescendant = newParentChild[1]
	s.nodes[parentIndex] = parent

	return nil
}

// leadsToViableHead returns true if the node or the best descendent of the node is viable for head.
// Any node with diff finalized or justified epoch than the ones in fork choice store
// should not be viable to head.
func (s *Store) leadsToViableHead(ctx context.Context, node *Node) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "protoArrayForkChoice.leadsToViableHead")
	defer span.End()

	var bestDescendentViable bool
	bestDescendentIndex := node.bestDescendant

	// If the best descendant is not part of the leaves.
	if bestDescendentIndex != nonExistentNode {
		// Protection against out of bound, best descendent index can not be
		// exceeds length of nodes list.
		if bestDescendentIndex >= uint64(len(s.nodes)) {
			return false, errInvalidBestDescendantIndex
		}

		bestDescendentNode := s.nodes[bestDescendentIndex]
		bestDescendentViable = s.viableForHead(ctx, bestDescendentNode)
	}

	// The node is viable as long as the best descendent is viable.
	return bestDescendentViable || s.viableForHead(ctx, node), nil
}

// viableForHead returns true if the node is viable to head.
// Any node with diff finalized or justified epoch than the ones in fork choice store
// should not be viable to head.
func (s *Store) viableForHead(ctx context.Context, node *Node) bool {
	ctx, span := trace.StartSpan(ctx, "protoArrayForkChoice.viableForHead")
	defer span.End()

	// `node` is viable if its justified epoch and finalized epoch are the same as the one in `Store`.
	// It's also viable if we are in genesis epoch.
	justified := s.justifiedEpoch == node.justifiedEpoch || s.justifiedEpoch == 0
	finalized := s.finalizedEpoch == node.finalizedEpoch || s.finalizedEpoch == 0

	return justified && finalized
}
