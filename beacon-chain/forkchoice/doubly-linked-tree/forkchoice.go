package doublylinkedtree

import (
	"context"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	pbrpc "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"go.opencensus.io/trace"
)

// New initializes a new fork choice store.
func New(justifiedEpoch, finalizedEpoch types.Epoch) *ForkChoice {
	s := &Store{
		justifiedEpoch:    justifiedEpoch,
		finalizedEpoch:    finalizedEpoch,
		proposerBoostRoot: [32]byte{},
		nodeByRoot:        make(map[[fieldparams.RootLength]byte]*Node),
		pruneThreshold:    defaultPruneThreshold,
	}

	b := make([]uint64, 0)
	v := make([]Vote, 0)
	return &ForkChoice{store: s, balances: b, votes: v}
}

// NodeCount returns the current number of nodes in the Store.
func (f *ForkChoice) NodeCount() int {
	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()
	return len(f.store.nodeByRoot)
}

// Head returns the head root from fork choice store.
// It firsts computes validator's balance changes then recalculates block tree from leaves to root.
func (f *ForkChoice) Head(
	ctx context.Context,
	justifiedEpoch types.Epoch,
	justifiedRoot [32]byte,
	justifiedStateBalances []uint64,
	finalizedEpoch types.Epoch,
) ([32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "doublyLinkedForkchoice.Head")
	defer span.End()
	f.votesLock.Lock()
	defer f.votesLock.Unlock()

	calledHeadCount.Inc()

	// Using the write lock here because `applyWeightChanges` that gets called subsequently requires a write operation.
	f.store.nodesLock.Lock()
	defer f.store.nodesLock.Unlock()

	f.store.updateCheckpoints(justifiedEpoch, finalizedEpoch)

	if err := f.updateBalances(justifiedStateBalances); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not update balances")
	}

	if err := f.store.applyProposerBoostScore(justifiedStateBalances); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not apply proposer boost score")
	}

	if err := f.store.treeRootNode.applyWeightChanges(ctx); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not apply weight changes")
	}

	if err := f.store.treeRootNode.updateBestDescendant(ctx, justifiedEpoch, finalizedEpoch); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not update best descendant")
	}

	return f.store.head(ctx, justifiedRoot)
}

// ProcessAttestation processes attestation for vote accounting, it iterates around validator indices
// and update their votes accordingly.
func (f *ForkChoice) ProcessAttestation(ctx context.Context, validatorIndices []uint64, blockRoot [32]byte, targetEpoch types.Epoch) {
	_, span := trace.StartSpan(ctx, "doublyLinkedForkchoice.ProcessAttestation")
	defer span.End()
	f.votesLock.Lock()
	defer f.votesLock.Unlock()

	for _, index := range validatorIndices {
		// Validator indices will grow the vote cache.
		for index >= uint64(len(f.votes)) {
			f.votes = append(f.votes, Vote{currentRoot: params.BeaconConfig().ZeroHash, nextRoot: params.BeaconConfig().ZeroHash})
		}

		// Newly allocated vote if the root fields are untouched.
		newVote := f.votes[index].nextRoot == params.BeaconConfig().ZeroHash &&
			f.votes[index].currentRoot == params.BeaconConfig().ZeroHash

		// Vote gets updated if it's newly allocated or high target epoch.
		if newVote || targetEpoch > f.votes[index].nextEpoch {
			f.votes[index].nextEpoch = targetEpoch
			f.votes[index].nextRoot = blockRoot
		}
	}

	processedAttestationCount.Inc()
}

// InsertOptimisticBlock processes a new block by inserting it to the fork choice store.
func (f *ForkChoice) InsertOptimisticBlock(
	ctx context.Context,
	slot types.Slot,
	blockRoot, parentRoot [fieldparams.RootLength]byte,
	justifiedEpoch, finalizedEpoch types.Epoch,
) error {
	ctx, span := trace.StartSpan(ctx, "doublyLinkedForkchoice.InsertOptimisticBlock")
	defer span.End()

	return f.store.insert(ctx, slot, blockRoot, parentRoot, justifiedEpoch, finalizedEpoch)
}

// Prune prunes the fork choice store with the new finalized root. The store is only pruned if the input
// root is different than the current store finalized root, and the number of the store has met prune threshold.
func (f *ForkChoice) Prune(ctx context.Context, finalizedRoot [32]byte) error {
	return f.store.prune(ctx, finalizedRoot)
}

// HasNode returns true if the node exists in fork choice store,
// false else wise.
func (f *ForkChoice) HasNode(root [32]byte) bool {
	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()

	_, ok := f.store.nodeByRoot[root]
	return ok
}

// HasParent returns true if the node parent exists in fork choice store,
// false else wise.
func (f *ForkChoice) HasParent(root [32]byte) bool {
	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()

	node, ok := f.store.nodeByRoot[root]
	if !ok || node == nil {
		return false
	}

	return node.parent != nil
}

// ParentRoot returns the parent root of the node.
func (f *ForkChoice) ParentRoot(root [32]byte) ([32]byte, bool) {
	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()

	n, ok := f.store.nodeByRoot[root]
	if !ok {
		return [32]byte{}, false
	}
	if n.parent == nil {
		return [32]byte{}, false
	}

	return n.parent.root, true
}

// IsCanonical returns true if the given root is part of the canonical chain.
func (f *ForkChoice) IsCanonical(root [32]byte) bool {
	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()

	node, ok := f.store.nodeByRoot[root]
	if !ok || node == nil {
		return false
	}

	if node.bestDescendant == nil {
		if f.store.headNode.bestDescendant == nil {
			return node == f.store.headNode
		}
		return node == f.store.headNode.bestDescendant
	}
	if f.store.headNode.bestDescendant == nil {
		return node.bestDescendant == f.store.headNode
	}
	return node.bestDescendant == f.store.headNode.bestDescendant
}

// IsOptimistic returns true if the given root has been optimistically synced.
func (f *ForkChoice) IsOptimistic(_ context.Context, root [32]byte) (bool, error) {
	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()

	node, ok := f.store.nodeByRoot[root]
	if !ok || node == nil {
		return false, ErrNilNode
	}

	return node.optimistic, nil
}

// AncestorRoot returns the ancestor root of input block root at a given slot.
func (f *ForkChoice) AncestorRoot(ctx context.Context, root [32]byte, slot types.Slot) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "protoArray.AncestorRoot")
	defer span.End()

	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()

	node, ok := f.store.nodeByRoot[root]
	if !ok || node == nil {
		return nil, ErrNilNode
	}

	n := node
	for n != nil && n.slot > slot {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		n = n.parent
	}

	if n == nil {
		return nil, ErrNilNode
	}

	return n.root[:], nil
}

// updateBalances updates the balances that directly voted for each block taking into account the
// validators' latest votes.
func (f *ForkChoice) updateBalances(newBalances []uint64) error {
	for index, vote := range f.votes {
		// Skip if validator has never voted for current root and next root (i.e. if the
		// votes are zero hash aka genesis block), there's nothing to compute.
		if vote.currentRoot == params.BeaconConfig().ZeroHash && vote.nextRoot == params.BeaconConfig().ZeroHash {
			continue
		}

		oldBalance := uint64(0)
		newBalance := uint64(0)
		// If the validator index did not exist in `f.balances` or
		// `newBalances` list above, the balance is just 0.
		if index < len(f.balances) {
			oldBalance = f.balances[index]
		}
		if index < len(newBalances) {
			newBalance = newBalances[index]
		}

		// Update only if the validator's balance or vote has changed.
		if vote.currentRoot != vote.nextRoot || oldBalance != newBalance {
			// Ignore the vote if the root is not in fork choice
			// store, that means we have not seen the block before.
			nextNode, ok := f.store.nodeByRoot[vote.nextRoot]
			if ok && vote.nextRoot != params.BeaconConfig().ZeroHash {
				// Protection against nil node
				if nextNode == nil {
					return ErrNilNode
				}
				nextNode.balance += newBalance
			}

			currentNode, ok := f.store.nodeByRoot[vote.currentRoot]
			if ok && vote.currentRoot != params.BeaconConfig().ZeroHash {
				// Protection against nil node
				if currentNode == nil {
					return ErrNilNode
				}
				if currentNode.balance < oldBalance {
					return errInvalidBalance
				}
				currentNode.balance -= oldBalance
			}
		}

		// Rotate the validator vote.
		f.votes[index].currentRoot = vote.nextRoot
	}
	f.balances = newBalances
	return nil
}

// Tips returns a list of possible heads from fork choice store, it returns the
// roots and the slots of the leaf nodes.
func (f *ForkChoice) Tips() ([][32]byte, []types.Slot) {
	return f.store.tips()
}

// ProposerBoost returns the proposerBoost of the store
func (f *ForkChoice) ProposerBoost() [fieldparams.RootLength]byte {
	return f.store.proposerBoost()
}

// SetOptimisticToValid sets the node with the given root as a fully validated node
func (f *ForkChoice) SetOptimisticToValid(ctx context.Context, root [fieldparams.RootLength]byte) error {
	f.store.nodesLock.Lock()
	defer f.store.nodesLock.Unlock()
	node, ok := f.store.nodeByRoot[root]
	if !ok || node == nil {
		return ErrNilNode
	}
	return node.setNodeAndParentValidated(ctx)
}

// JustifiedEpoch of fork choice store.
func (f *ForkChoice) JustifiedEpoch() types.Epoch {
	return f.store.justifiedEpoch
}

// FinalizedEpoch of fork choice store.
func (f *ForkChoice) FinalizedEpoch() types.Epoch {
	return f.store.finalizedEpoch
}

func (f *ForkChoice) ForkChoiceNodes() []*pbrpc.ForkChoiceNode {
	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()
	ret := make([]*pbrpc.ForkChoiceNode, len(f.store.nodeByRoot))
	return f.store.treeRootNode.rpcNodes(ret)
}

// SetOptimisticToInvalid removes a block with an invalid execution payload from fork choice store
func (f *ForkChoice) SetOptimisticToInvalid(ctx context.Context, root [fieldparams.RootLength]byte) error {
	return f.store.removeNode(ctx, root)
}
