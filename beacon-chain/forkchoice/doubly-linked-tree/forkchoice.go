package doublylinkedtree

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice"
	forkchoicetypes "github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	pbrpc "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// New initializes a new fork choice store.
func New(justifiedEpoch, finalizedEpoch types.Epoch) *ForkChoice {
	s := &Store{
		justifiedEpoch:    justifiedEpoch,
		finalizedEpoch:    finalizedEpoch,
		proposerBoostRoot: [32]byte{},
		nodeByRoot:        make(map[[fieldparams.RootLength]byte]*Node),
		nodeByPayload:     make(map[[fieldparams.RootLength]byte]*Node),
		slashedIndices:    make(map[types.ValidatorIndex]bool),
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
	justifiedRoot [32]byte,
	justifiedStateBalances []uint64,
) ([32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "doublyLinkedForkchoice.Head")
	defer span.End()
	f.votesLock.Lock()
	defer f.votesLock.Unlock()

	calledHeadCount.Inc()

	// Using the write lock here because `applyWeightChanges` that gets called subsequently requires a write operation.
	f.store.nodesLock.Lock()
	defer f.store.nodesLock.Unlock()

	if err := f.updateBalances(justifiedStateBalances); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not update balances")
	}

	if err := f.store.applyProposerBoostScore(justifiedStateBalances); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not apply proposer boost score")
	}

	if err := f.store.treeRootNode.applyWeightChanges(ctx); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not apply weight changes")
	}

	if err := f.store.treeRootNode.updateBestDescendant(ctx, f.store.justifiedEpoch, f.store.finalizedEpoch); err != nil {
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
func (f *ForkChoice) InsertOptimisticBlock(ctx context.Context, state state.ReadOnlyBeaconState, root [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "doublyLinkedForkchoice.InsertOptimisticBlock")
	defer span.End()

	slot := state.Slot()
	bh := state.LatestBlockHeader()
	if bh == nil {
		return errNilBlockHeader
	}
	parentRoot := bytesutil.ToBytes32(bh.ParentRoot)
	payloadHash := [32]byte{}
	if state.Version() >= version.Bellatrix {
		ph, err := state.LatestExecutionPayloadHeader()
		if err != nil {
			return err
		}
		if ph != nil {
			copy(payloadHash[:], ph.BlockHash)
		}
	}
	jc := state.CurrentJustifiedCheckpoint()
	if jc == nil {
		return errInvalidNilCheckpoint
	}
	justifiedEpoch := jc.Epoch
	fc := state.FinalizedCheckpoint()
	if fc == nil {
		return errInvalidNilCheckpoint
	}
	finalizedEpoch := fc.Epoch
	return f.store.insert(ctx, slot, root, parentRoot, payloadHash, justifiedEpoch, finalizedEpoch)
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
func (f *ForkChoice) IsOptimistic(root [32]byte) (bool, error) {
	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()

	node, ok := f.store.nodeByRoot[root]
	if !ok || node == nil {
		return false, errors.Wrap(ErrNilNode, "could not determine optimistic status")
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
		return nil, errors.Wrap(ErrNilNode, "could not determine ancestor root")
	}

	n := node
	for n != nil && n.slot > slot {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		n = n.parent
	}

	if n == nil {
		return nil, errors.Wrap(ErrNilNode, "could not determine ancestor root")
	}

	return n.root[:], nil
}

// updateBalances updates the balances that directly voted for each block taking into account the
// validators' latest votes.
func (f *ForkChoice) updateBalances(newBalances []uint64) error {
	for index, vote := range f.votes {
		// Skip if validator has been slashed
		if f.store.slashedIndices[types.ValidatorIndex(index)] {
			continue
		}
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
					return errors.Wrap(ErrNilNode, "could not update balances")
				}
				nextNode.balance += newBalance
			}

			currentNode, ok := f.store.nodeByRoot[vote.currentRoot]
			if ok && vote.currentRoot != params.BeaconConfig().ZeroHash {
				// Protection against nil node
				if currentNode == nil {
					return errors.Wrap(ErrNilNode, "could not update balances")
				}
				if currentNode.balance < oldBalance {
					f.store.proposerBoostLock.RLock()
					log.WithFields(logrus.Fields{
						"nodeRoot":                   fmt.Sprintf("%#x", bytesutil.Trunc(vote.currentRoot[:])),
						"oldBalance":                 oldBalance,
						"nodeBalance":                currentNode.balance,
						"nodeWeight":                 currentNode.weight,
						"proposerBoostRoot":          fmt.Sprintf("%#x", bytesutil.Trunc(f.store.proposerBoostRoot[:])),
						"previousProposerBoostRoot":  fmt.Sprintf("%#x", bytesutil.Trunc(f.store.previousProposerBoostRoot[:])),
						"previousProposerBoostScore": f.store.previousProposerBoostScore,
					}).Warning("node with invalid balance, setting it to zero")
					f.store.proposerBoostLock.RUnlock()
					currentNode.balance = 0
				} else {
					currentNode.balance -= oldBalance
				}
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
		return errors.Wrap(ErrNilNode, "could not set node to valid")
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
func (f *ForkChoice) SetOptimisticToInvalid(ctx context.Context, root, parentRoot, payloadHash [fieldparams.RootLength]byte) ([][32]byte, error) {
	return f.store.setOptimisticToInvalid(ctx, root, parentRoot, payloadHash)
}

// InsertSlashedIndex adds the given slashed validator index to the
// store-tracked list. Votes from these validators are not accounted for
// in forkchoice.
func (f *ForkChoice) InsertSlashedIndex(_ context.Context, index types.ValidatorIndex) {
	f.store.nodesLock.Lock()
	defer f.store.nodesLock.Unlock()
	// return early if the index was already included:
	if f.store.slashedIndices[index] {
		return
	}
	f.store.slashedIndices[index] = true

	// Subtract last vote from this equivocating validator
	f.votesLock.RLock()
	defer f.votesLock.RUnlock()

	if index >= types.ValidatorIndex(len(f.balances)) {
		return
	}

	if index >= types.ValidatorIndex(len(f.votes)) {
		return
	}

	node, ok := f.store.nodeByRoot[f.votes[index].currentRoot]
	if !ok || node == nil {
		return
	}

	if node.balance < f.balances[index] {
		node.balance = 0
	} else {
		node.balance -= f.balances[index]
	}
}

// UpdateJustifiedCheckpoint sets the justified epoch to the given one
func (f *ForkChoice) UpdateJustifiedCheckpoint(jc *pbrpc.Checkpoint) error {
	if jc == nil {
		return errInvalidNilCheckpoint
	}
	f.store.nodesLock.Lock()
	defer f.store.nodesLock.Unlock()
	f.store.justifiedEpoch = jc.Epoch
	return nil
}

// UpdateFinalizedCheckpoint sets the finalized epoch to the given one
func (f *ForkChoice) UpdateFinalizedCheckpoint(fc *pbrpc.Checkpoint) error {
	if fc == nil {
		return errInvalidNilCheckpoint
	}
	f.store.nodesLock.Lock()
	defer f.store.nodesLock.Unlock()
	f.store.finalizedEpoch = fc.Epoch
	return nil
}

// CommonAncestorRoot returns the common ancestor root between the two block roots r1 and r2.
func (f *ForkChoice) CommonAncestorRoot(ctx context.Context, r1 [32]byte, r2 [32]byte) ([32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "doublelinkedtree.CommonAncestorRoot")
	defer span.End()

	// Do nothing if the input roots are the same.
	if r1 == r2 {
		return r1, nil
	}

	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()

	n1, ok := f.store.nodeByRoot[r1]
	if !ok || n1 == nil {
		return [32]byte{}, errors.Wrap(ErrNilNode, "could not determine common ancestor root")
	}
	n2, ok := f.store.nodeByRoot[r2]
	if !ok || n2 == nil {
		return [32]byte{}, errors.Wrap(ErrNilNode, "could not determine common ancestor root")
	}

	for {
		if ctx.Err() != nil {
			return [32]byte{}, ctx.Err()
		}
		if n1.slot > n2.slot {
			n1 = n1.parent
			// Reaches the end of the tree and unable to find common ancestor.
			if n1 == nil {
				return [32]byte{}, forkchoice.ErrUnknownCommonAncestor
			}
		} else {
			n2 = n2.parent
			// Reaches the end of the tree and unable to find common ancestor.
			if n2 == nil {
				return [32]byte{}, forkchoice.ErrUnknownCommonAncestor
			}
		}
		if n1 == n2 {
			return n1.root, nil
		}
	}
}

// InsertOptimisticChain inserts all nodes corresponding to blocks in the slice
// `blocks`. This slice must be ordered from child to parent. It includes all
// blocks **except** the first one (that is the one with the highest slot
// number). All blocks are assumed to be a strict chain
// where blocks[i].Parent = blocks[i+1]. Also we assume that the parent of the
// last block in this list is already included in forkchoice store.
func (f *ForkChoice) InsertOptimisticChain(ctx context.Context, chain []*forkchoicetypes.BlockAndCheckpoints) error {
	if len(chain) == 0 {
		return nil
	}
	for i := len(chain) - 1; i > 0; i-- {
		b := chain[i].Block
		r := bytesutil.ToBytes32(chain[i-1].Block.ParentRoot())
		parentRoot := bytesutil.ToBytes32(b.ParentRoot())
		payloadHash, err := blocks.GetBlockPayloadHash(b)
		if err != nil {
			return err
		}
		if err := f.store.insert(ctx,
			b.Slot(), r, parentRoot, payloadHash,
			chain[i].JustifiedEpoch, chain[i].FinalizedEpoch); err != nil {
			return err
		}
	}
	return nil
}
