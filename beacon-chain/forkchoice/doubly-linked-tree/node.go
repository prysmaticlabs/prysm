package doublylinkedtree

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/config/features"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	v1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
)

// applyWeightChanges recomputes the weight of the node passed as an argument and all of its descendants,
// using the current balance stored in each node. This function requires a lock
// in Store.nodesLock
func (n *Node) applyWeightChanges(ctx context.Context) error {
	// Recursively calling the children to sum their weights.
	childrenWeight := uint64(0)
	for _, child := range n.children {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := child.applyWeightChanges(ctx); err != nil {
			return err
		}
		childrenWeight += child.weight
	}
	if n.root == params.BeaconConfig().ZeroHash {
		return nil
	}
	n.weight = n.balance + childrenWeight
	return nil
}

// updateBestDescendant updates the best descendant of this node and its
// children. This function assumes the caller has a lock on Store.nodesLock
func (n *Node) updateBestDescendant(ctx context.Context, justifiedEpoch, finalizedEpoch, currentEpoch primitives.Epoch) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if len(n.children) == 0 {
		n.bestDescendant = nil
		return nil
	}

	var bestChild *Node
	bestWeight := uint64(0)
	hasViableDescendant := false
	for _, child := range n.children {
		if child == nil {
			return errors.Wrap(ErrNilNode, "could not update best descendant")
		}
		if err := child.updateBestDescendant(ctx, justifiedEpoch, finalizedEpoch, currentEpoch); err != nil {
			return err
		}
		childLeadsToViableHead := child.leadsToViableHead(justifiedEpoch, currentEpoch)
		if childLeadsToViableHead && !hasViableDescendant {
			// The child leads to a viable head, but the current
			// parent's best child doesn't.
			bestWeight = child.weight
			bestChild = child
			hasViableDescendant = true
		} else if childLeadsToViableHead {
			// If both are viable, compare their weights.
			if child.weight == bestWeight {
				// Tie-breaker of equal weights by root.
				if bytes.Compare(child.root[:], bestChild.root[:]) > 0 {
					bestChild = child
				}
			} else if child.weight > bestWeight {
				bestChild = child
				bestWeight = child.weight
			}
		}
	}
	if hasViableDescendant {
		if bestChild.bestDescendant == nil {
			n.bestDescendant = bestChild
		} else {
			n.bestDescendant = bestChild.bestDescendant
		}
	} else {
		n.bestDescendant = nil
	}
	return nil
}

// viableForHead returns true if the node is viable to head.
// Any node with different finalized or justified epoch than
// the ones in fork choice store should not be viable to head.
func (n *Node) viableForHead(justifiedEpoch, currentEpoch primitives.Epoch) bool {
	justified := justifiedEpoch == n.justifiedEpoch || justifiedEpoch == 0
	if features.Get().EnableDefensivePull && !justified && justifiedEpoch+1 == currentEpoch {
		if n.unrealizedJustifiedEpoch+1 >= currentEpoch && n.justifiedEpoch+2 >= currentEpoch {
			justified = true
		}
	}
	return justified
}

func (n *Node) leadsToViableHead(justifiedEpoch, currentEpoch primitives.Epoch) bool {
	if n.bestDescendant == nil {
		return n.viableForHead(justifiedEpoch, currentEpoch)
	}
	return n.bestDescendant.viableForHead(justifiedEpoch, currentEpoch)
}

// setNodeAndParentValidated sets the current node and all the ancestors as validated (i.e. non-optimistic).
func (n *Node) setNodeAndParentValidated(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if !n.optimistic {
		return nil
	}
	n.optimistic = false

	if n.parent == nil {
		return nil
	}
	return n.parent.setNodeAndParentValidated(ctx)
}

// nodeTreeDump appends to the given list all the nodes descending from this one
func (n *Node) nodeTreeDump(ctx context.Context, nodes []*v1.ForkChoiceNode) ([]*v1.ForkChoiceNode, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	var parentRoot [32]byte
	if n.parent != nil {
		parentRoot = n.parent.root
	}
	thisNode := &v1.ForkChoiceNode{
		Slot:                     n.slot,
		BlockRoot:                n.root[:],
		ParentRoot:               parentRoot[:],
		JustifiedEpoch:           n.justifiedEpoch,
		FinalizedEpoch:           n.finalizedEpoch,
		UnrealizedJustifiedEpoch: n.unrealizedJustifiedEpoch,
		UnrealizedFinalizedEpoch: n.unrealizedFinalizedEpoch,
		Balance:                  n.balance,
		Weight:                   n.weight,
		ExecutionOptimistic:      n.optimistic,
		ExecutionBlockHash:       n.payloadHash[:],
		Timestamp:                n.timestamp,
	}
	if n.optimistic {
		thisNode.Validity = v1.ForkChoiceNodeValidity_OPTIMISTIC
	} else {
		thisNode.Validity = v1.ForkChoiceNodeValidity_VALID
	}

	nodes = append(nodes, thisNode)
	var err error
	for _, child := range n.children {
		nodes, err = child.nodeTreeDump(ctx, nodes)
		if err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// VotedFraction returns the fraction of the committee that voted directly for
// this node.
func (f *ForkChoice) VotedFraction(root [32]byte) (uint64, error) {
	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()

	// Avoid division by zero before a block is inserted.
	if f.store.committeeBalance == 0 {
		return 0, nil
	}

	node, ok := f.store.nodeByRoot[root]
	if !ok || node == nil {
		return 0, ErrNilNode
	}
	return node.balance * 100 / f.store.committeeBalance, nil
}
