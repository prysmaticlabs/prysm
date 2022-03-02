package doubly_linked_tree

import (
	"bytes"
	"context"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	pbrpc "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// Slot of the fork choice node.
func (n *Node) Slot() types.Slot {
	return n.slot
}

// Balance returns the current balance of the Node
func (n *Node) Balance() uint64 {
	return n.balance
}

// Optimistic returns the optimistic status of the node
func (n *Node) Optimistic() bool {
	return n.optimistic
}

// Root of the fork choice node.
func (n *Node) Root() [32]byte {
	return n.root
}

// Parent of the fork choice node.
func (n *Node) Parent() *Node {
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

// Children returns the children of this node
func (n *Node) Children() []*Node {
	return n.children
}

// depth returns the length of the path to the root of Fork Choice
func (n *Node) depth() uint64 {
	ret := uint64(0)
	for node := n.parent; node != nil; node = node.parent {
		ret += 1
	}
	return ret
}

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

// updateBestDescendant updates the best descendant of this node and its children.
func (n *Node) updateBestDescendant(ctx context.Context, justifiedEpoch, finalizedEpoch types.Epoch) error {
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
			return errNilNode
		}
		if err := child.updateBestDescendant(ctx, justifiedEpoch, finalizedEpoch); err != nil {
			return err
		}
		childLeadsToViableHead := child.leadsToViableHead(justifiedEpoch, finalizedEpoch)
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
func (n *Node) viableForHead(justifiedEpoch, finalizedEpoch types.Epoch) bool {
	justified := justifiedEpoch == n.justifiedEpoch || justifiedEpoch == 0
	finalized := finalizedEpoch == n.finalizedEpoch || finalizedEpoch == 0

	return justified && finalized
}

func (n *Node) leadsToViableHead(justifiedEpoch, finalizedEpoch types.Epoch) bool {
	if n.bestDescendant == nil {
		return n.viableForHead(justifiedEpoch, finalizedEpoch)
	}
	return n.bestDescendant.viableForHead(justifiedEpoch, finalizedEpoch)
}

// BestDescendant of the fork choice node.
func (n *Node) BestDescendant() *Node {
	return n.bestDescendant
}

// setNodeAndParentValidated sets the current node and the parent as validated (i.e. non-optimistic).
func (n *Node) setNodeAndParentValidated(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if !n.optimistic || n.parent == nil {
		return nil
	}

	n.optimistic = false
	return n.parent.setNodeAndParentValidated(ctx)
}

// rpcNodes is used by the RPC Debug endpoint to return information
// about all nodes in the fork choice store
func (n *Node) rpcNodes(ret []*pbrpc.ForkChoiceNode) []*pbrpc.ForkChoiceNode {
	for _, child := range n.children {
		ret = child.rpcNodes(ret)
	}
	r := n.root
	p := [32]byte{}
	if n.parent != nil {
		copy(p[:], n.parent.root[:])
	}
	b := [32]byte{}
	if n.bestDescendant != nil {
		copy(b[:], n.bestDescendant.root[:])
	}
	node := &pbrpc.ForkChoiceNode{
		Slot:           n.slot,
		Root:           r[:],
		Parent:         p[:],
		JustifiedEpoch: n.justifiedEpoch,
		FinalizedEpoch: n.finalizedEpoch,
		Weight:         n.weight,
		BestDescendant: b[:],
	}
	ret = append(ret, node)
	return ret
}
