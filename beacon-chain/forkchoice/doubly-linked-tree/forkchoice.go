package doublylinkedtree

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice"
	forkchoicetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/types"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/features"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	v1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// New initializes a new fork choice store.
func New() *ForkChoice {
	s := &Store{
		justifiedCheckpoint:           &forkchoicetypes.Checkpoint{},
		bestJustifiedCheckpoint:       &forkchoicetypes.Checkpoint{},
		unrealizedJustifiedCheckpoint: &forkchoicetypes.Checkpoint{},
		unrealizedFinalizedCheckpoint: &forkchoicetypes.Checkpoint{},
		prevJustifiedCheckpoint:       &forkchoicetypes.Checkpoint{},
		finalizedCheckpoint:           &forkchoicetypes.Checkpoint{},
		proposerBoostRoot:             [32]byte{},
		nodeByRoot:                    make(map[[fieldparams.RootLength]byte]*Node),
		nodeByPayload:                 make(map[[fieldparams.RootLength]byte]*Node),
		slashedIndices:                make(map[types.ValidatorIndex]bool),
		receivedBlocksLastEpoch:       [fieldparams.SlotsPerEpoch]types.Slot{},
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
	justifiedStateBalances []uint64,
) ([32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "doublyLinkedForkchoice.Head")
	defer span.End()
	f.votesLock.Lock()
	defer f.votesLock.Unlock()

	calledHeadCount.Inc()

	// Using the write lock here because subsequent calls to `updateBalances`, `applyProposerBoostScore`,
	// `applyWeightChanges`, `updateBestDescendant`, and `head` require write operations on nodes.
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

	jc := f.JustifiedCheckpoint()
	fc := f.FinalizedCheckpoint()
	currentEpoch := slots.EpochsSinceGenesis(time.Unix(int64(f.store.genesisTime), 0))
	if err := f.store.treeRootNode.updateBestDescendant(ctx, jc.Epoch, fc.Epoch, currentEpoch); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not update best descendant")
	}
	return f.store.head(ctx)
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

// InsertNode processes a new block by inserting it to the fork choice store.
func (f *ForkChoice) InsertNode(ctx context.Context, state state.BeaconState, root [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "doublyLinkedForkchoice.InsertNode")
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
	node, err := f.store.insert(ctx, slot, root, parentRoot, payloadHash, justifiedEpoch, finalizedEpoch)
	if err != nil {
		return err
	}

	if !features.Get().DisablePullTips {
		jc, fc = f.store.pullTips(state, node, jc, fc)
	}
	return f.updateCheckpoints(ctx, jc, fc)
}

// updateCheckpoints update the checkpoints when inserting a new node.
func (f *ForkChoice) updateCheckpoints(ctx context.Context, jc, fc *ethpb.Checkpoint) error {
	f.store.checkpointsLock.Lock()
	if jc.Epoch > f.store.justifiedCheckpoint.Epoch {
		if jc.Epoch > f.store.bestJustifiedCheckpoint.Epoch {
			f.store.bestJustifiedCheckpoint = &forkchoicetypes.Checkpoint{Epoch: jc.Epoch,
				Root: bytesutil.ToBytes32(jc.Root)}
		}
		currentSlot := slots.CurrentSlot(f.store.genesisTime)
		if slots.SinceEpochStarts(currentSlot) < params.BeaconConfig().SafeSlotsToUpdateJustified {
			f.store.prevJustifiedCheckpoint = f.store.justifiedCheckpoint
			f.store.justifiedCheckpoint = &forkchoicetypes.Checkpoint{Epoch: jc.Epoch,
				Root: bytesutil.ToBytes32(jc.Root)}
		} else {
			currentJcp := f.store.justifiedCheckpoint
			currentRoot := currentJcp.Root
			if currentRoot == params.BeaconConfig().ZeroHash {
				currentRoot = f.store.originRoot
			}
			jSlot, err := slots.EpochStart(currentJcp.Epoch)
			if err != nil {
				f.store.checkpointsLock.Unlock()
				return err
			}
			jcRoot := bytesutil.ToBytes32(jc.Root)
			// Releasing here the checkpoints lock because
			// AncestorRoot acquires a lock on nodes and that can
			// cause a double lock.
			f.store.checkpointsLock.Unlock()
			root, err := f.AncestorRoot(ctx, jcRoot, jSlot)
			if err != nil {
				return err
			}
			f.store.checkpointsLock.Lock()
			if root == currentRoot {
				f.store.prevJustifiedCheckpoint = f.store.justifiedCheckpoint
				f.store.justifiedCheckpoint = &forkchoicetypes.Checkpoint{Epoch: jc.Epoch,
					Root: jcRoot}
			}
		}
	}
	// Update finalization
	if fc.Epoch <= f.store.finalizedCheckpoint.Epoch {
		f.store.checkpointsLock.Unlock()
		return nil
	}
	f.store.finalizedCheckpoint = &forkchoicetypes.Checkpoint{Epoch: fc.Epoch,
		Root: bytesutil.ToBytes32(fc.Root)}
	f.store.justifiedCheckpoint = &forkchoicetypes.Checkpoint{Epoch: jc.Epoch,
		Root: bytesutil.ToBytes32(jc.Root)}
	f.store.checkpointsLock.Unlock()
	return f.store.prune(ctx)
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
		return true, ErrNilNode
	}

	return node.optimistic, nil
}

// AncestorRoot returns the ancestor root of input block root at a given slot.
func (f *ForkChoice) AncestorRoot(ctx context.Context, root [32]byte, slot types.Slot) ([32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "protoArray.AncestorRoot")
	defer span.End()

	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()

	node, ok := f.store.nodeByRoot[root]
	if !ok || node == nil {
		return [32]byte{}, errors.Wrap(ErrNilNode, "could not determine ancestor root")
	}

	n := node
	for n != nil && n.slot > slot {
		if ctx.Err() != nil {
			return [32]byte{}, ctx.Err()
		}
		n = n.parent
	}

	if n == nil {
		return [32]byte{}, errors.Wrap(ErrNilNode, "could not determine ancestor root")
	}

	return n.root, nil
}

// updateBalances updates the balances that directly voted for each block taking into account the
// validators' latest votes. This function requires a lock in Store.nodesLock
// and votesLock
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

// BestJustifiedCheckpoint of fork choice store.
func (f *ForkChoice) BestJustifiedCheckpoint() *forkchoicetypes.Checkpoint {
	f.store.checkpointsLock.RLock()
	defer f.store.checkpointsLock.RUnlock()
	return f.store.bestJustifiedCheckpoint
}

// PreviousJustifiedCheckpoint of fork choice store.
func (f *ForkChoice) PreviousJustifiedCheckpoint() *forkchoicetypes.Checkpoint {
	f.store.checkpointsLock.RLock()
	defer f.store.checkpointsLock.RUnlock()
	return f.store.prevJustifiedCheckpoint
}

// JustifiedCheckpoint of fork choice store.
func (f *ForkChoice) JustifiedCheckpoint() *forkchoicetypes.Checkpoint {
	f.store.checkpointsLock.RLock()
	defer f.store.checkpointsLock.RUnlock()
	return f.store.justifiedCheckpoint
}

// FinalizedCheckpoint of fork choice store.
func (f *ForkChoice) FinalizedCheckpoint() *forkchoicetypes.Checkpoint {
	f.store.checkpointsLock.RLock()
	defer f.store.checkpointsLock.RUnlock()
	return f.store.finalizedCheckpoint
}

// SetOptimisticToInvalid removes a block with an invalid execution payload from fork choice store
func (f *ForkChoice) SetOptimisticToInvalid(ctx context.Context, root, parentRoot, payloadHash [fieldparams.RootLength]byte) ([][32]byte, error) {
	return f.store.setOptimisticToInvalid(ctx, root, parentRoot, payloadHash)
}

// InsertSlashedIndex adds the given slashed validator index to the
// store-tracked list. Votes from these validators are not accounted for
// in forkchoice.
func (f *ForkChoice) InsertSlashedIndex(_ context.Context, index types.ValidatorIndex) {
	f.votesLock.RLock()
	defer f.votesLock.RUnlock()

	f.store.nodesLock.Lock()
	defer f.store.nodesLock.Unlock()
	// return early if the index was already included:
	if f.store.slashedIndices[index] {
		return
	}
	f.store.slashedIndices[index] = true

	// Subtract last vote from this equivocating validator

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

// UpdateJustifiedCheckpoint sets the justified checkpoint to the given one
func (f *ForkChoice) UpdateJustifiedCheckpoint(jc *forkchoicetypes.Checkpoint) error {
	if jc == nil {
		return errInvalidNilCheckpoint
	}
	f.store.checkpointsLock.Lock()
	defer f.store.checkpointsLock.Unlock()
	f.store.prevJustifiedCheckpoint = f.store.justifiedCheckpoint
	f.store.justifiedCheckpoint = jc
	bj := f.store.bestJustifiedCheckpoint
	if bj == nil || bj.Root == params.BeaconConfig().ZeroHash || jc.Epoch > bj.Epoch {
		f.store.bestJustifiedCheckpoint = &forkchoicetypes.Checkpoint{Epoch: jc.Epoch, Root: jc.Root}
	}
	return nil
}

// UpdateFinalizedCheckpoint sets the finalized checkpoint to the given one
func (f *ForkChoice) UpdateFinalizedCheckpoint(fc *forkchoicetypes.Checkpoint) error {
	if fc == nil {
		return errInvalidNilCheckpoint
	}
	f.store.checkpointsLock.Lock()
	defer f.store.checkpointsLock.Unlock()
	f.store.finalizedCheckpoint = fc
	return nil
}

// CommonAncestorRoot returns the common ancestor root between the two block roots r1 and r2.
func (f *ForkChoice) CommonAncestor(ctx context.Context, r1 [32]byte, r2 [32]byte) ([32]byte, types.Slot, error) {
	ctx, span := trace.StartSpan(ctx, "doublelinkedtree.CommonAncestorRoot")
	defer span.End()

	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()

	n1, ok := f.store.nodeByRoot[r1]
	if !ok || n1 == nil {
		return [32]byte{}, 0, forkchoice.ErrUnknownCommonAncestor
	}

	// Do nothing if the input roots are the same.
	if r1 == r2 {
		return r1, n1.slot, nil
	}

	n2, ok := f.store.nodeByRoot[r2]
	if !ok || n2 == nil {
		return [32]byte{}, 0, forkchoice.ErrUnknownCommonAncestor
	}

	for {
		if ctx.Err() != nil {
			return [32]byte{}, 0, ctx.Err()
		}
		if n1.slot > n2.slot {
			n1 = n1.parent
			// Reaches the end of the tree and unable to find common ancestor.
			// This should not happen at runtime as the finalized
			// node has to be a common ancestor
			if n1 == nil {
				return [32]byte{}, 0, forkchoice.ErrUnknownCommonAncestor
			}
		} else {
			n2 = n2.parent
			// Reaches the end of the tree and unable to find common ancestor.
			if n2 == nil {
				return [32]byte{}, 0, forkchoice.ErrUnknownCommonAncestor
			}
		}
		if n1 == n2 {
			return n1.root, n1.slot, nil
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
		r := chain[i-1].Block.ParentRoot()
		parentRoot := b.ParentRoot()
		payloadHash, err := blocks.GetBlockPayloadHash(b)
		if err != nil {
			return err
		}
		if _, err := f.store.insert(ctx,
			b.Slot(), r, parentRoot, payloadHash,
			chain[i].JustifiedCheckpoint.Epoch, chain[i].FinalizedCheckpoint.Epoch); err != nil {
			return err
		}
		if err := f.updateCheckpoints(ctx, chain[i].JustifiedCheckpoint, chain[i].FinalizedCheckpoint); err != nil {
			return err
		}
	}
	return nil
}

// SetGenesisTime sets the genesisTime tracked by forkchoice
func (f *ForkChoice) SetGenesisTime(genesisTime uint64) {
	f.store.genesisTime = genesisTime
}

// SetOriginRoot sets the genesis block root
func (f *ForkChoice) SetOriginRoot(root [32]byte) {
	f.store.originRoot = root
}

// CachedHeadRoot returns the last cached head root
func (f *ForkChoice) CachedHeadRoot() [32]byte {
	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()
	node := f.store.headNode
	if node == nil {
		return [32]byte{}
	}
	return f.store.headNode.root
}

// FinalizedPayloadBlockHash returns the hash of the payload at the finalized checkpoint
func (f *ForkChoice) FinalizedPayloadBlockHash() [32]byte {
	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()
	root := f.FinalizedCheckpoint().Root
	node, ok := f.store.nodeByRoot[root]
	if !ok || node == nil {
		// This should not happen
		return [32]byte{}
	}
	return node.payloadHash
}

// JustifiedPayloadBlockHash returns the hash of the payload at the justified checkpoint
func (f *ForkChoice) JustifiedPayloadBlockHash() [32]byte {
	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()
	root := f.JustifiedCheckpoint().Root
	node, ok := f.store.nodeByRoot[root]
	if !ok || node == nil {
		// This should not happen
		return [32]byte{}
	}
	return node.payloadHash
}

// ForkChoiceDump returns a full dump of forkhoice.
func (f *ForkChoice) ForkChoiceDump(ctx context.Context) (*v1.ForkChoiceResponse, error) {
	jc := &v1.Checkpoint{
		Epoch: f.store.justifiedCheckpoint.Epoch,
		Root:  f.store.justifiedCheckpoint.Root[:],
	}
	bjc := &v1.Checkpoint{
		Epoch: f.store.bestJustifiedCheckpoint.Epoch,
		Root:  f.store.bestJustifiedCheckpoint.Root[:],
	}
	ujc := &v1.Checkpoint{
		Epoch: f.store.unrealizedJustifiedCheckpoint.Epoch,
		Root:  f.store.unrealizedJustifiedCheckpoint.Root[:],
	}
	fc := &v1.Checkpoint{
		Epoch: f.store.finalizedCheckpoint.Epoch,
		Root:  f.store.finalizedCheckpoint.Root[:],
	}
	ufc := &v1.Checkpoint{
		Epoch: f.store.unrealizedFinalizedCheckpoint.Epoch,
		Root:  f.store.unrealizedFinalizedCheckpoint.Root[:],
	}
	nodes := make([]*v1.ForkChoiceNode, 0, f.NodeCount())
	var err error
	if f.store.treeRootNode != nil {
		nodes, err = f.store.treeRootNode.nodeTreeDump(ctx, nodes)
		if err != nil {
			return nil, err
		}
	}
	var headRoot [32]byte
	if f.store.headNode != nil {
		headRoot = f.store.headNode.root
	}
	resp := &v1.ForkChoiceResponse{
		JustifiedCheckpoint:           jc,
		BestJustifiedCheckpoint:       bjc,
		UnrealizedJustifiedCheckpoint: ujc,
		FinalizedCheckpoint:           fc,
		UnrealizedFinalizedCheckpoint: ufc,
		ProposerBoostRoot:             f.store.proposerBoostRoot[:],
		PreviousProposerBoostRoot:     f.store.previousProposerBoostRoot[:],
		HeadRoot:                      headRoot[:],
		ForkchoiceNodes:               nodes,
	}
	return resp, nil

}
