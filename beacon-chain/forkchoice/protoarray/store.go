package protoarray

import (
	"context"
	"fmt"

	types "github.com/prysmaticlabs/eth2-types"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"go.opencensus.io/trace"
)

// This defines the minimal number of block nodes that can be in the tree
// before getting pruned upon new finalization.
const defaultPruneThreshold = 256

// This defines the largest staked power that an attacker is willing to
// spend in an attack to "safe head"
const safeHeadAttackersWeight = uint64(4096 * 32000000000)

// applyProposerBoostScore applies the current proposer boost scores to the
// relevant nodes
func (s *Store) applyProposerBoostScore(newBalances []uint64) error {
	s.proposerBoostLock.Lock()
	defer s.proposerBoostLock.Unlock()

	proposerScore := uint64(0)
	var err error
	if s.previousProposerBoostRoot != params.BeaconConfig().ZeroHash {
		previousNode, ok := s.nodeByRoot[s.previousProposerBoostRoot]
		if !ok || previousNode == nil {
			return errInvalidProposerBoostRoot
		}
		previousNode.balance -= s.previousProposerBoostScore
	}

	if s.proposerBoostRoot != params.BeaconConfig().ZeroHash {
		currentNode, ok := s.nodeByRoot[s.proposerBoostRoot]
		if !ok || currentNode == nil {
			return errInvalidProposerBoostRoot
		}
		proposerScore, err = computeProposerBoostScore(newBalances)
		if err != nil {
			return err
		}
		currentNode.balance += proposerScore
	}
	s.previousProposerBoostRoot = s.proposerBoostRoot
	s.previousProposerBoostScore = proposerScore
	return nil
}

// NodeNumber returns the current number of nodes in the Store
func (s *Store) NodeNumber() int {
	return len(s.nodeByRoot)
}

// JustifiedEpoch of fork choice store.
func (s *Store) JustifiedEpoch() types.Epoch {
	return s.justifiedEpoch
}

// FinalizedEpoch of fork choice store.
func (s *Store) FinalizedEpoch() types.Epoch {
	return s.finalizedEpoch
}

// ProposerBoost of fork choice store.
func (s *Store) ProposerBoost() [fieldparams.RootLength]byte {
	s.proposerBoostLock.RLock()
	defer s.proposerBoostLock.RUnlock()
	return s.proposerBoostRoot
}

// PruneThreshold of fork choice store.
func (s *Store) PruneThreshold() uint64 {
	return s.pruneThreshold
}

// head starts from justified root and then follows the best descendant links
// to find the best block for head. This function assumes a lock on s.nodesLock
func (s *Store) head(ctx context.Context, justifiedRoot [32]byte) ([32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "protoArrayForkChoice.head")
	defer span.End()

	// JustifiedRoot has to be known
	justifiedNode, ok := s.nodeByRoot[justifiedRoot]
	if !ok || justifiedNode == nil {
		return [32]byte{}, errUnknownJustifiedRoot
	}

	// If the justified node doesn't have a best descendant,
	// the best node is itself.
	bestDescendant := justifiedNode.bestDescendant
	if bestDescendant == nil {
		bestDescendant = justifiedNode
	}

	if !bestDescendant.viableForHead(s.justifiedEpoch, s.finalizedEpoch) {
		return [32]byte{}, fmt.Errorf("head at slot %d with weight %d is not eligible, finalizedEpoch %d != %d, justifiedEpoch %d != %d",
			bestDescendant.slot, bestDescendant.weight/10e9, bestDescendant.finalizedEpoch, s.finalizedEpoch, bestDescendant.justifiedEpoch, s.justifiedEpoch)
	}

	// Update metrics.
	if bestDescendant != s.headNode {
		headChangesCount.Inc()
		headSlotNumber.Set(float64(bestDescendant.slot))
		s.headNode = bestDescendant
	}

	return bestDescendant.root, nil
}

// insert registers a new block node to the fork choice store's node list.
// It then updates the new node's parent with best child and descendant node.
func (s *Store) insert(ctx context.Context,
	slot types.Slot,
	root, parentRoot [fieldparams.RootLength]byte,
	justifiedEpoch, finalizedEpoch types.Epoch, optimistic bool) error {
	_, span := trace.StartSpan(ctx, "protoArrayForkChoice.insert")
	defer span.End()

	s.nodesLock.Lock()
	defer s.nodesLock.Unlock()

	// Return if the block has been inserted into Store before.
	if _, ok := s.nodeByRoot[root]; ok {
		return nil
	}

	parent := s.nodeByRoot[parentRoot]

	n := &Node{
		slot:           slot,
		root:           root,
		parent:         parent,
		justifiedEpoch: justifiedEpoch,
		finalizedEpoch: finalizedEpoch,
		optimistic:     optimistic,
	}

	s.nodeByRoot[root] = n
	if parent != nil {
		parent.children = append(parent.children, n)
		if err := s.treeRootNode.updateBestDescendant(ctx, s.justifiedEpoch, s.finalizedEpoch); err != nil {
			return err
		}
	}

	if !optimistic {
		if err := n.setNodeAndParentValidated(ctx); err != nil {
			return err
		}
	} else {
		optimisticCount.Inc()
	}

	// Set the node as root if the store was empty
	if s.treeRootNode == nil {
		s.treeRootNode = n
		s.headNode = n
	}

	// Update metrics.
	processedBlockCount.Inc()
	nodeCount.Set(float64(len(s.nodeByRoot)))

	return nil
}

// updateCheckpoints Update the justified / finalized epochs in store if necessary.
func (s *Store) updateCheckpoints(justifiedEpoch, finalizedEpoch types.Epoch) {
	s.justifiedEpoch = justifiedEpoch
	s.finalizedEpoch = finalizedEpoch
}

// pruneMaps prunes the `nodeByRoot` map
// starting from `node` down to the finalized Node or to a leaf of the Fork
// choice store. This method assumes a lock on nodesLock.
func (s *Store) pruneMaps(ctx context.Context, node, finalizedNode *Node) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if node == finalizedNode {
		return nil
	}
	for _, child := range node.children {
		if err := s.pruneMaps(ctx, child, finalizedNode); err != nil {
			return err
		}
	}

	delete(s.nodeByRoot, node.root)
	return nil
}

// prune prunes the fork choice store with the new finalized root. The store is only pruned if the input
// root is different than the current store finalized root, and the number of the store has met prune threshold.
func (s *Store) prune(ctx context.Context, finalizedRoot [32]byte) error {
	_, span := trace.StartSpan(ctx, "protoArrayForkChoice.Prune")
	defer span.End()

	s.nodesLock.Lock()
	defer s.nodesLock.Unlock()

	finalizedNode, ok := s.nodeByRoot[finalizedRoot]
	if !ok || finalizedNode == nil {
		return errUnknownFinalizedRoot
	}

	// The number of the nodes has not met the prune threshold.
	// Pruning at small numbers incurs more cost than benefit.
	if finalizedNode.depth() < s.pruneThreshold {
		return nil
	}

	// Prune nodeByRoot starting from root
	if err := s.pruneMaps(ctx, s.treeRootNode, finalizedNode); err != nil {
		return err
	}

	finalizedNode.parent = nil
	s.treeRootNode = finalizedNode

	prunedCount.Inc()
	return nil
}

// tips returns a list of possible heads from fork choice store, it returns the
// roots and the slots of the leaf nodes.
func (s *Store) tips() ([][32]byte, []types.Slot) {
	var roots [][32]byte
	var slots []types.Slot
	for root, node := range s.nodeByRoot {
		if len(node.children) == 0 {
			roots = append(roots, root)
			slots = append(slots, node.slot)
		}
	}
	return roots, slots
}

//TreeRoot returns the current root Node of the Store
func (s *Store) TreeRoot() *Node {
	s.nodesLock.RLock()
	defer s.nodesLock.RUnlock()
	return s.treeRootNode
}

// Safe returns whether the block is considered safe with respect to
// synchronicity assumptions on the network and how much an attacker is willing
// to spend
func (s *Store) Safe(ctx context.Context, root [32]byte, committeeWeight uint64) (bool, error) {
	node, ok := s.nodeByRoot[root]
	if !ok || node == nil {
		return false, errNilNode
	}

	if node.bestDescendant != nil && node.bestDescendant != s.headNode {
		return false, nil
	}

	potentialVotes := uint64(s.headNode.slot-node.slot+1) * committeeWeight
	actualVotes := node.weight

	tailWeight := uint64(0)
	for node.parent != nil {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		node = node.parent
		tailWeight += node.balance
	}

	s.proposerBoostLock.RLock()
	defer s.proposerBoostLock.RUnlock()
	return tailWeight+2*actualVotes > potentialVotes+safeHeadAttackersWeight+s.previousProposerBoostScore, nil
}
