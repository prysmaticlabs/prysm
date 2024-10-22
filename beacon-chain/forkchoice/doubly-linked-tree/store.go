package doublylinkedtree

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing/trace"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
)

// head starts from justified root and then follows the best descendant links
// to find the best block for head.
func (s *Store) head(ctx context.Context) ([32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "doublyLinkedForkchoice.head")
	defer span.End()

	if err := ctx.Err(); err != nil {
		return [32]byte{}, err
	}

	// JustifiedRoot has to be known
	justifiedNode, ok := s.emptyNodeByRoot[s.justifiedCheckpoint.Root]
	if !ok || justifiedNode == nil {
		// If the justifiedCheckpoint is from genesis, then the root is
		// zeroHash. In this case it should be the root of forkchoice
		// tree.
		if s.justifiedCheckpoint.Epoch == params.BeaconConfig().GenesisEpoch {
			justifiedNode = s.treeRootNode
		} else {
			return [32]byte{}, errors.WithMessage(errUnknownJustifiedRoot, fmt.Sprintf("%#x", s.justifiedCheckpoint.Root))
		}
	}

	// If the justified node doesn't have a best descendant,
	// the best node is itself.
	bestDescendant := justifiedNode.bestDescendant
	if bestDescendant == nil {
		bestDescendant = justifiedNode
	}
	currentEpoch := slots.EpochsSinceGenesis(time.Unix(int64(s.genesisTime), 0))
	if !bestDescendant.viableForHead(s.justifiedCheckpoint.Epoch, currentEpoch) {
		s.allTipsAreInvalid = true
		return [32]byte{}, fmt.Errorf("head at slot %d with weight %d is not eligible, finalizedEpoch, justified Epoch %d, %d != %d, %d",
			bestDescendant.block.slot, bestDescendant.weight/10e9, bestDescendant.block.finalizedEpoch, bestDescendant.block.justifiedEpoch, s.finalizedCheckpoint.Epoch, s.justifiedCheckpoint.Epoch)
	}
	s.allTipsAreInvalid = false

	// Update metrics.
	if bestDescendant != s.headNode {
		headChangesCount.Inc()
		headSlotNumber.Set(float64(bestDescendant.block.slot))
		s.headNode = bestDescendant
	}

	return bestDescendant.block.root, nil
}

// insert registers a new block node to the fork choice store's node list.
// It then updates the new node's parent with the best child and descendant node.
func (s *Store) insert(
	ctx context.Context,
	slot primitives.Slot,
	root, parentRoot, payloadHash, parentHash [fieldparams.RootLength]byte,
	justifiedEpoch, finalizedEpoch primitives.Epoch,
) (*BlockNode, error) {
	ctx, span := trace.StartSpan(ctx, "doublyLinkedForkchoice.insert")
	defer span.End()

	// Return if the block has been inserted into Store before.
	n, rootPresent := s.emptyNodeByRoot[root]
	if rootPresent {
		return n.block, nil
	}
	parent := s.emptyNodeByRoot[parentRoot]
	fullParent := s.fullNodeByPayload[parentHash]
	if fullParent != nil && parent != nil && fullParent.block == parent.block {
		parent = fullParent
	}
	innerBlock := &BlockNode{
		slot:                     slot,
		root:                     root,
		payloadHash:              payloadHash,
		parent:                   parent,
		fullParent:               fullParent,
		justifiedEpoch:           justifiedEpoch,
		unrealizedJustifiedEpoch: justifiedEpoch,
		finalizedEpoch:           finalizedEpoch,
		unrealizedFinalizedEpoch: finalizedEpoch,
		timestamp:                uint64(time.Now().Unix()),
		ptcVote:                  make(map[primitives.PTCStatus]uint64),
	}
	emptyNode := &Node{
		block:      innerBlock,
		optimistic: true,
		children:   make([]*Node, 0),
		full:       false,
	}
	fullNode := &Node{
		block:      innerBlock,
		optimistic: true,
		children:   make([]*Node, 0),
		full:       true,
	}
	// Set the node's target checkpoint
	if slot%params.BeaconConfig().SlotsPerEpoch == 0 {
		innerBlock.target = fullNode
	} else if parent != nil {
		if slots.ToEpoch(slot) == slots.ToEpoch(parent.block.slot) {
			innerBlock.target = parent.block.target
		} else {
			innerBlock.target = parent
		}
	}

	if parent == nil {
		if s.treeRootNode == nil {
			s.treeRootNode = fullNode
			s.headNode = fullNode
			s.highestReceivedNode = fullNode
		} else if s.treeRootNode.block.root != fullNode.block.root {
			delete(s.emptyNodeByRoot, root)
			delete(s.fullNodeByPayload, payloadHash)
			return innerBlock, errInvalidParentRoot
		}
	}
	s.emptyNodeByRoot[root] = emptyNode
	s.fullNodeByPayload[payloadHash] = fullNode
	if parent != nil {
		parent.children = append(parent.children, emptyNode, fullNode)
		// Apply proposer boost
		timeNow := uint64(time.Now().Unix())
		if timeNow < s.genesisTime {
			return innerBlock, nil
		}
		secondsIntoSlot := (timeNow - s.genesisTime) % params.BeaconConfig().SecondsPerSlot
		currentSlot := slots.CurrentSlot(s.genesisTime)
		boostThreshold := params.BeaconConfig().SecondsPerSlot / params.BeaconConfig().IntervalsPerSlot
		isFirstBlock := s.proposerBoostRoot == [32]byte{}
		if currentSlot == slot && secondsIntoSlot < boostThreshold && isFirstBlock {
			s.proposerBoostRoot = root
		}

		// Update best descendants
		jEpoch := s.justifiedCheckpoint.Epoch
		fEpoch := s.finalizedCheckpoint.Epoch
		if err := s.treeRootNode.updateBestDescendant(ctx, jEpoch, fEpoch, slots.ToEpoch(currentSlot)); err != nil {
			_, remErr := s.removeNode(ctx, n)
			if remErr != nil {
				log.WithError(remErr).Error("could not remove node")
			}
			return nil, errors.Wrap(err, "could not update best descendants")
		}
	}
	// Update metrics.
	processedBlockCount.Inc()
	nodeCount.Set(float64(len(s.emptyNodeByRoot)))

	// Only update received block slot if it's within epoch from current time.
	if slot+params.BeaconConfig().SlotsPerEpoch > slots.CurrentSlot(s.genesisTime) {
		s.receivedBlocksLastEpoch[slot%params.BeaconConfig().SlotsPerEpoch] = slot
	}
	// Update highest slot tracking.
	if slot > s.highestReceivedNode.block.slot {
		s.highestReceivedNode = fullNode
	}
	return innerBlock, nil
}

// pruneFinalizedNodeByRootMap prunes the `nodeByRoot` map
// starting from `node` down to the finalized Node or to a leaf of the Fork
// choice store.
func (s *Store) pruneFinalizedNodeByRootMap(ctx context.Context, node, finalizedNode *Node) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if node == finalizedNode {
		if node.block.target != node {
			node.block.target = nil
		}
		node.block.parent = nil
		node.block.fullParent = nil
		return nil
	}
	for _, child := range node.children {
		if err := s.pruneFinalizedNodeByRootMap(ctx, child, finalizedNode); err != nil {
			return err
		}
	}

	node.children = nil
	delete(s.emptyNodeByRoot, node.block.root)
	delete(s.fullNodeByPayload, node.block.payloadHash)
	return nil
}

// prune prunes the fork choice store. It removes all nodes that compete with the finalized root.
// This function does not prune for invalid optimistically synced nodes, it deals only with pruning upon finalization
func (s *Store) prune(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "doublyLinkedForkchoice.Prune")
	defer span.End()

	finalizedRoot := s.finalizedCheckpoint.Root
	finalizedEpoch := s.finalizedCheckpoint.Epoch
	finalizedNode, ok := s.emptyNodeByRoot[finalizedRoot]
	if !ok || finalizedNode == nil {
		return errors.WithMessage(errUnknownFinalizedRoot, fmt.Sprintf("%#x", finalizedRoot))
	}
	// return early if we haven't changed the finalized checkpoint
	if finalizedNode.block.parent == nil {
		return nil
	}
	fullFinalizedNode := s.fullNodeByPayload[finalizedNode.block.payloadHash]
	if fullFinalizedNode != nil {
		finalizedNode = fullFinalizedNode
	}
	// Prune nodeByRoot starting from root
	if err := s.pruneFinalizedNodeByRootMap(ctx, s.treeRootNode, finalizedNode); err != nil {
		return err
	}

	s.treeRootNode = finalizedNode
	prunedCount.Inc()
	// Prune all children of the finalized checkpoint block that are incompatible with it
	checkpointMaxSlot, err := slots.EpochStart(finalizedEpoch)
	if err != nil {
		return errors.Wrap(err, "could not compute epoch start")
	}
	if finalizedNode.block.slot == checkpointMaxSlot {
		return nil
	}

	for _, child := range finalizedNode.children {
		if child != nil && child.block.slot <= checkpointMaxSlot {
			if err := s.pruneFinalizedNodeByRootMap(ctx, child, finalizedNode); err != nil {
				return errors.Wrap(err, "could not prune incompatible finalized child")
			}
		}
	}
	return nil
}

// tips returns a list of possible heads from fork choice store, it returns the
// roots and the slots of the leaf nodes.
func (s *Store) tips() ([][32]byte, []primitives.Slot) {
	var roots [][32]byte
	var slots []primitives.Slot

	for _, node := range s.fullNodeByPayload {
		if len(node.children) == 0 {
			roots = append(roots, node.block.root)
			slots = append(slots, node.block.slot)
		}
	}
	return roots, slots
}

// HighestReceivedBlockSlotRoot returns the highest slot and root received by the forkchoice
func (f *ForkChoice) HighestReceivedBlockSlotRoot() (primitives.Slot, [32]byte) {
	if f.store.highestReceivedNode == nil {
		return 0, [32]byte{}
	}
	return f.store.highestReceivedNode.block.slot, f.store.highestReceivedNode.block.root
}

// HighestReceivedBlockSlotDelay returns the number of slots that the highest
// received block was late when receiving it
func (f *ForkChoice) HighestReceivedBlockDelay() primitives.Slot {
	n := f.store.highestReceivedNode
	if n == nil {
		return 0
	}
	secs, err := slots.SecondsSinceSlotStart(n.block.slot, f.store.genesisTime, n.block.timestamp)
	if err != nil {
		return 0
	}
	return primitives.Slot(secs / params.BeaconConfig().SecondsPerSlot)
}

// ReceivedBlocksLastEpoch returns the number of blocks received in the last epoch
func (f *ForkChoice) ReceivedBlocksLastEpoch() (uint64, error) {
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
