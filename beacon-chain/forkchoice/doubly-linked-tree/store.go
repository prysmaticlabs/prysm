package doublylinkedtree

import (
	"context"
	"fmt"
	"time"

	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"go.opencensus.io/trace"
)

// This defines the minimal number of block nodes that can be in the tree
// before getting pruned upon new finalization.
const defaultPruneThreshold = 256

// applyProposerBoostScore applies the current proposer boost scores to the
// relevant nodes. This function requires a lock in Store.nodesLock.
func (s *Store) applyProposerBoostScore(newBalances []uint64) error {
	s.proposerBoostLock.Lock()
	defer s.proposerBoostLock.Unlock()

	proposerScore := uint64(0)
	var err error
	if s.previousProposerBoostRoot != params.BeaconConfig().ZeroHash {
		previousNode, ok := s.nodeByRoot[s.previousProposerBoostRoot]
		if !ok || previousNode == nil {
			s.previousProposerBoostRoot = [32]byte{}
			log.WithError(errInvalidProposerBoostRoot).Errorf(fmt.Sprintf("invalid prev root %#x", s.previousProposerBoostRoot))
			return nil
		}
		previousNode.balance -= s.previousProposerBoostScore
	}

	if s.proposerBoostRoot != params.BeaconConfig().ZeroHash {
		currentNode, ok := s.nodeByRoot[s.proposerBoostRoot]
		if !ok || currentNode == nil {
			s.proposerBoostRoot = [32]byte{}
			log.WithError(errInvalidProposerBoostRoot).Errorf(fmt.Sprintf("invalid current root %#x", s.proposerBoostRoot))
			return nil
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

// ProposerBoost of fork choice store.
func (s *Store) proposerBoost() [fieldparams.RootLength]byte {
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
func (s *Store) head(ctx context.Context) ([32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "doublyLinkedForkchoice.head")
	defer span.End()
	s.checkpointsLock.RLock()
	defer s.checkpointsLock.RUnlock()

	if err := ctx.Err(); err != nil {
		return [32]byte{}, err
	}

	// JustifiedRoot has to be known
	justifiedNode, ok := s.nodeByRoot[s.justifiedCheckpoint.Root]
	if !ok || justifiedNode == nil {
		// If the justifiedCheckpoint is from genesis, then the root is
		// zeroHash. In this case it should be the root of forkchoice
		// tree.
		if s.justifiedCheckpoint.Epoch == params.BeaconConfig().GenesisEpoch {
			justifiedNode = s.treeRootNode
		} else {
			return [32]byte{}, errUnknownJustifiedRoot
		}
	}

	// If the justified node doesn't have a best descendant,
	// the best node is itself.
	bestDescendant := justifiedNode.bestDescendant
	if bestDescendant == nil {
		bestDescendant = justifiedNode
	}
	currentEpoch := slots.EpochsSinceGenesis(time.Unix(int64(s.genesisTime), 0))
	if !bestDescendant.viableForHead(s.justifiedCheckpoint.Epoch, s.finalizedCheckpoint.Epoch, currentEpoch) {
		s.allTipsAreInvalid = true
		return [32]byte{}, fmt.Errorf("head at slot %d with weight %d is not eligible, finalizedEpoch, justified Epoch %d, %d != %d, %d",
			bestDescendant.slot, bestDescendant.weight/10e9, bestDescendant.finalizedEpoch, bestDescendant.justifiedEpoch, s.finalizedCheckpoint.Epoch, s.justifiedCheckpoint.Epoch)
	}
	s.allTipsAreInvalid = false

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
	root, parentRoot, payloadHash [fieldparams.RootLength]byte,
	justifiedEpoch, finalizedEpoch types.Epoch) (*Node, error) {
	ctx, span := trace.StartSpan(ctx, "doublyLinkedForkchoice.insert")
	defer span.End()

	s.nodesLock.Lock()
	defer s.nodesLock.Unlock()

	// Return if the block has been inserted into Store before.
	if n, ok := s.nodeByRoot[root]; ok {
		return n, nil
	}

	parent := s.nodeByRoot[parentRoot]

	n := &Node{
		slot:                     slot,
		root:                     root,
		parent:                   parent,
		justifiedEpoch:           justifiedEpoch,
		unrealizedJustifiedEpoch: justifiedEpoch,
		finalizedEpoch:           finalizedEpoch,
		unrealizedFinalizedEpoch: finalizedEpoch,
		optimistic:               true,
		payloadHash:              payloadHash,
		timestamp:                uint64(time.Now().Unix()),
	}

	s.nodeByPayload[payloadHash] = n
	s.nodeByRoot[root] = n
	if parent == nil {
		if s.treeRootNode == nil {
			s.treeRootNode = n
			s.headNode = n
		} else {
			return n, errInvalidParentRoot
		}
	} else {
		parent.children = append(parent.children, n)
		// Apply proposer boost
		timeNow := uint64(time.Now().Unix())
		if timeNow < s.genesisTime {
			return n, nil
		}
		secondsIntoSlot := (timeNow - s.genesisTime) % params.BeaconConfig().SecondsPerSlot
		currentSlot := slots.CurrentSlot(s.genesisTime)
		boostThreshold := params.BeaconConfig().SecondsPerSlot / params.BeaconConfig().IntervalsPerSlot
		if currentSlot == slot && secondsIntoSlot < boostThreshold {
			s.proposerBoostLock.Lock()
			s.proposerBoostRoot = root
			s.proposerBoostLock.Unlock()
		}

		// Update best descendants
		s.checkpointsLock.RLock()
		jEpoch := s.justifiedCheckpoint.Epoch
		fEpoch := s.finalizedCheckpoint.Epoch
		s.checkpointsLock.RUnlock()
		if err := s.treeRootNode.updateBestDescendant(ctx, jEpoch, fEpoch, slots.ToEpoch(currentSlot)); err != nil {
			return n, err
		}
	}
	// Update metrics.
	processedBlockCount.Inc()
	nodeCount.Set(float64(len(s.nodeByRoot)))

	// Only update received block slot if it's within epoch from current time.
	if slot+params.BeaconConfig().SlotsPerEpoch > slots.CurrentSlot(s.genesisTime) {
		s.receivedBlocksLastEpoch[slot%params.BeaconConfig().SlotsPerEpoch] = slot
	}
	// Update highest slot tracking.
	if slot > s.highestReceivedSlot {
		s.highestReceivedSlot = slot
	}

	return n, nil
}

// pruneFinalizedNodeByRootMap prunes the `nodeByRoot` map
// starting from `node` down to the finalized Node or to a leaf of the Fork
// choice store. This method assumes a lock on nodesLock.
func (s *Store) pruneFinalizedNodeByRootMap(ctx context.Context, node, finalizedNode *Node) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if node == finalizedNode {
		return nil
	}
	for _, child := range node.children {
		if err := s.pruneFinalizedNodeByRootMap(ctx, child, finalizedNode); err != nil {
			return err
		}
	}

	delete(s.nodeByRoot, node.root)
	return nil
}

// prune prunes the fork choice store with the new finalized root. The store is only pruned if the input
// root is different than the current store finalized root, and the number of the store has met prune threshold.
// This function does not prune for invalid optimistically synced nodes, it deals only with pruning upon finalization
func (s *Store) prune(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "doublyLinkedForkchoice.Prune")
	defer span.End()

	s.nodesLock.Lock()
	defer s.nodesLock.Unlock()
	s.checkpointsLock.RLock()
	finalizedRoot := s.finalizedCheckpoint.Root
	s.checkpointsLock.RUnlock()

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
	if err := s.pruneFinalizedNodeByRootMap(ctx, s.treeRootNode, finalizedNode); err != nil {
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

	s.nodesLock.RLock()
	defer s.nodesLock.RUnlock()

	for root, node := range s.nodeByRoot {
		if len(node.children) == 0 {
			roots = append(roots, root)
			slots = append(slots, node.slot)
		}
	}
	return roots, slots
}

// HighestReceivedBlockSlot returns the highest slot received by the forkchoice
func (f *ForkChoice) HighestReceivedBlockSlot() types.Slot {
	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()
	return f.store.highestReceivedSlot
}

// ReceivedBlocksLastEpoch returns the number of blocks received in the last epoch
func (f *ForkChoice) ReceivedBlocksLastEpoch() (uint64, error) {
	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()
	count := uint64(0)
	lowerBound := slots.CurrentSlot(f.store.genesisTime)
	var err error
	if lowerBound > fieldparams.SlotsPerEpoch {
		lowerBound, err = lowerBound.SafeSub(fieldparams.SlotsPerEpoch)
		if err != nil {
			return 0, err
		}
	}

	for _, s := range f.store.receivedBlocksLastEpoch {
		if s != 0 && lowerBound <= s {
			count++
		}
	}
	return count, nil
}
