package protoarray

import (
	"bytes"
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
	pmath "github.com/prysmaticlabs/prysm/v3/math"
	v1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// This defines the minimal number of block nodes that can be in the tree
// before getting pruned upon new finalization.
const defaultPruneThreshold = 256

// New initializes a new fork choice store.
func New() *ForkChoice {
	s := &Store{
		justifiedCheckpoint:           &forkchoicetypes.Checkpoint{},
		bestJustifiedCheckpoint:       &forkchoicetypes.Checkpoint{},
		unrealizedJustifiedCheckpoint: &forkchoicetypes.Checkpoint{},
		prevJustifiedCheckpoint:       &forkchoicetypes.Checkpoint{},
		finalizedCheckpoint:           &forkchoicetypes.Checkpoint{},
		unrealizedFinalizedCheckpoint: &forkchoicetypes.Checkpoint{},
		proposerBoostRoot:             [32]byte{},
		nodes:                         make([]*Node, 0),
		nodesIndices:                  make(map[[32]byte]uint64),
		payloadIndices:                make(map[[32]byte]uint64),
		canonicalNodes:                make(map[[32]byte]bool),
		slashedIndices:                make(map[types.ValidatorIndex]bool),
		pruneThreshold:                defaultPruneThreshold,
		receivedBlocksLastEpoch:       [fieldparams.SlotsPerEpoch]types.Slot{},
	}

	b := make([]uint64, 0)
	v := make([]Vote, 0)
	return &ForkChoice{store: s, balances: b, votes: v}
}

// Head returns the head root from fork choice store.
// It firsts computes validator's balance changes then recalculates block tree from leaves to root.
func (f *ForkChoice) Head(ctx context.Context, justifiedStateBalances []uint64) ([32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "protoArrayForkChoice.Head")
	defer span.End()
	f.votesLock.Lock()
	defer f.votesLock.Unlock()

	calledHeadCount.Inc()
	newBalances := justifiedStateBalances

	// Using the write lock here because `updateCanonicalNodes` that gets called subsequently requires a write operation.
	f.store.nodesLock.Lock()
	defer f.store.nodesLock.Unlock()
	deltas, newVotes, err := computeDeltas(ctx, len(f.store.nodes), f.store.nodesIndices, f.votes, f.balances, newBalances, f.store.slashedIndices)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "Could not compute deltas")
	}
	f.votes = newVotes

	if err := f.store.applyWeightChanges(ctx, newBalances, deltas); err != nil {
		return [32]byte{}, errors.Wrap(err, "Could not apply score changes")
	}
	f.balances = newBalances

	return f.store.head(ctx)
}

// ProcessAttestation processes attestation for vote accounting, it iterates around validator indices
// and update their votes accordingly.
func (f *ForkChoice) ProcessAttestation(ctx context.Context, validatorIndices []uint64, blockRoot [32]byte, targetEpoch types.Epoch) {
	_, span := trace.StartSpan(ctx, "protoArrayForkChoice.ProcessAttestation")
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

// NodeCount returns the current number of nodes in the Store
func (f *ForkChoice) NodeCount() int {
	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()
	return len(f.store.nodes)
}

// ProposerBoost returns the proposerBoost of the store
func (f *ForkChoice) ProposerBoost() [fieldparams.RootLength]byte {
	return f.store.proposerBoost()
}

// InsertNode processes a new block by inserting it to the fork choice store.
func (f *ForkChoice) InsertNode(ctx context.Context, state state.BeaconState, root [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "protoArrayForkChoice.InsertNode")
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
		bj := f.store.bestJustifiedCheckpoint
		if bj == nil || jc.Epoch > bj.Epoch {
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
			// release the checkpoints lock here because
			// AncestorRoot takes a lock on nodes and that can lead
			// to double locks
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

	_, ok := f.store.nodesIndices[root]
	return ok
}

// HasParent returns true if the node parent exists in fork choice store,
// false else wise.
func (f *ForkChoice) HasParent(root [32]byte) bool {
	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()

	i, ok := f.store.nodesIndices[root]
	if !ok || i >= uint64(len(f.store.nodes)) {
		return false
	}

	return f.store.nodes[i].parent != NonExistentNode
}

// IsCanonical returns true if the given root is part of the canonical chain.
func (f *ForkChoice) IsCanonical(root [32]byte) bool {
	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()

	return f.store.canonicalNodes[root]
}

// AncestorRoot returns the ancestor root of input block root at a given slot.
func (f *ForkChoice) AncestorRoot(ctx context.Context, root [32]byte, slot types.Slot) ([32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "protoArray.AncestorRoot")
	defer span.End()

	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()

	i, ok := f.store.nodesIndices[root]
	if !ok {
		return [32]byte{}, errors.New("node does not exist")
	}
	if i >= uint64(len(f.store.nodes)) {
		return [32]byte{}, errors.New("node index out of range")
	}

	for f.store.nodes[i].slot > slot {
		if ctx.Err() != nil {
			return [32]byte{}, ctx.Err()
		}

		i = f.store.nodes[i].parent

		if i >= uint64(len(f.store.nodes)) {
			return [32]byte{}, errors.New("node index out of range")
		}
	}

	return f.store.nodes[i].root, nil
}

// CommonAncestorRoot returns the common ancestor root between the two block roots r1 and r2.
func (f *ForkChoice) CommonAncestor(ctx context.Context, r1 [32]byte, r2 [32]byte) ([32]byte, types.Slot, error) {
	ctx, span := trace.StartSpan(ctx, "protoArray.CommonAncestorRoot")
	defer span.End()

	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()

	i1, ok := f.store.nodesIndices[r1]
	if !ok || i1 >= uint64(len(f.store.nodes)) {
		return [32]byte{}, 0, forkchoice.ErrUnknownCommonAncestor
	}

	// Do nothing if the two input roots are the same.
	if r1 == r2 {
		n1 := f.store.nodes[i1]
		return r1, n1.slot, nil
	}

	i2, ok := f.store.nodesIndices[r2]
	if !ok || i2 >= uint64(len(f.store.nodes)) {
		return [32]byte{}, 0, forkchoice.ErrUnknownCommonAncestor
	}

	for {
		if ctx.Err() != nil {
			return [32]byte{}, 0, ctx.Err()
		}
		if i1 > i2 {
			n1 := f.store.nodes[i1]
			i1 = n1.parent
			// Reaches the end of the tree and unable to find common ancestor.
			if i1 >= uint64(len(f.store.nodes)) {
				return [32]byte{}, 0, forkchoice.ErrUnknownCommonAncestor
			}
		} else {
			n2 := f.store.nodes[i2]
			i2 = n2.parent
			// Reaches the end of the tree and unable to find common ancestor.
			if i2 >= uint64(len(f.store.nodes)) {
				return [32]byte{}, 0, forkchoice.ErrUnknownCommonAncestor
			}
		}
		if i1 == i2 {
			n1 := f.store.nodes[i1]
			return n1.root, n1.slot, nil
		}
	}
}

// PruneThreshold of fork choice store.
func (s *Store) PruneThreshold() uint64 {
	return s.pruneThreshold
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

// proposerBoost of fork choice store.
func (s *Store) proposerBoost() [fieldparams.RootLength]byte {
	s.proposerBoostLock.RLock()
	defer s.proposerBoostLock.RUnlock()
	return s.proposerBoostRoot
}

// head starts from justified root and then follows the best descendant links
// to find the best block for head. It assumes the caller has a lock on nodes.
func (s *Store) head(ctx context.Context) ([32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "protoArrayForkChoice.head")
	defer span.End()

	s.checkpointsLock.RLock()

	// Justified index has to be valid in node indices map, and can not be out of bound.
	if s.justifiedCheckpoint == nil {
		s.checkpointsLock.RUnlock()
		return [32]byte{}, errInvalidNilCheckpoint
	}

	justifiedIndex, ok := s.nodesIndices[s.justifiedCheckpoint.Root]
	if !ok {
		// If the justifiedCheckpoint is from genesis, then the root is
		// zeroHash. In this case it should be the root of forkchoice
		// tree.
		if s.justifiedCheckpoint.Epoch == params.BeaconConfig().GenesisEpoch {
			justifiedIndex = uint64(0)
		} else {
			s.checkpointsLock.RUnlock()
			return [32]byte{}, errUnknownJustifiedRoot
		}
	}
	s.checkpointsLock.RUnlock()
	if justifiedIndex >= uint64(len(s.nodes)) {
		return [32]byte{}, errInvalidJustifiedIndex
	}
	justifiedNode := s.nodes[justifiedIndex]
	bestDescendantIndex := justifiedNode.bestDescendant
	// If the justified node doesn't have a best descendant,
	// the best node is itself.
	if bestDescendantIndex == NonExistentNode {
		bestDescendantIndex = justifiedIndex
	}
	if bestDescendantIndex >= uint64(len(s.nodes)) {
		return [32]byte{}, errInvalidBestDescendantIndex
	}
	bestNode := s.nodes[bestDescendantIndex]

	if !s.viableForHead(bestNode) {
		s.allTipsAreInvalid = true
		s.checkpointsLock.RLock()
		jEpoch := s.justifiedCheckpoint.Epoch
		fEpoch := s.finalizedCheckpoint.Epoch
		s.checkpointsLock.RUnlock()
		return [32]byte{}, fmt.Errorf("head at slot %d with weight %d is not eligible, finalizedEpoch %d != %d, justifiedEpoch %d != %d",
			bestNode.slot, bestNode.weight/10e9, bestNode.finalizedEpoch, fEpoch, bestNode.justifiedEpoch, jEpoch)
	}
	s.allTipsAreInvalid = false

	// Update metrics and tracked head Root
	if bestNode.root != s.lastHeadRoot {
		headChangesCount.Inc()
		headSlotNumber.Set(float64(bestNode.slot))
		s.lastHeadRoot = bestNode.root
	}

	// Update canonical mapping given the head root.
	if err := s.updateCanonicalNodes(ctx, bestNode.root); err != nil {
		return [32]byte{}, err
	}

	return bestNode.root, nil
}

// updateCanonicalNodes updates the canonical nodes mapping given the input
// block root. This function assumes the caller holds a lock in Store.nodesLock
func (s *Store) updateCanonicalNodes(ctx context.Context, root [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "protoArrayForkChoice.updateCanonicalNodes")
	defer span.End()

	// Set the input node to canonical.
	i := s.nodesIndices[root]
	var newCanonicalRoots [][32]byte
	var n *Node
	for i != NonExistentNode {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Get the parent node, if the node is already in canonical mapping,
		// we can be sure rest of the ancestors are canonical. Exit early.
		n = s.nodes[i]
		if s.canonicalNodes[n.root] {
			break
		}

		// Set parent node to canonical. Repeat until parent node index is undefined.
		newCanonicalRoots = append(newCanonicalRoots, n.root)
		i = n.parent
	}

	// i is either NonExistentNode or has the index of the last canonical
	// node before the last head update.
	if i == NonExistentNode {
		s.canonicalNodes = make(map[[fieldparams.RootLength]byte]bool)
	} else {
		for j := i + 1; j < uint64(len(s.nodes)); j++ {
			delete(s.canonicalNodes, s.nodes[j].root)
		}
	}

	for _, canonicalRoot := range newCanonicalRoots {
		s.canonicalNodes[canonicalRoot] = true
	}

	return nil
}

// insert registers a new block node to the fork choice store's node list.
// It then updates the new node's parent with best child and descendant node.
func (s *Store) insert(ctx context.Context,
	slot types.Slot,
	root, parent, payloadHash [32]byte,
	justifiedEpoch, finalizedEpoch types.Epoch) (*Node, error) {
	ctx, span := trace.StartSpan(ctx, "protoArrayForkChoice.insert")
	defer span.End()

	s.nodesLock.Lock()
	defer s.nodesLock.Unlock()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Return if the block has been inserted into Store before.
	if idx, ok := s.nodesIndices[root]; ok {
		return s.nodes[idx], nil
	}

	index := uint64(len(s.nodes))
	parentIndex, ok := s.nodesIndices[parent]
	// Mark genesis block's parent as non-existent.
	if !ok {
		parentIndex = NonExistentNode
	}

	n := &Node{
		slot:                     slot,
		root:                     root,
		parent:                   parentIndex,
		justifiedEpoch:           justifiedEpoch,
		unrealizedJustifiedEpoch: justifiedEpoch,
		finalizedEpoch:           finalizedEpoch,
		unrealizedFinalizedEpoch: finalizedEpoch,
		bestChild:                NonExistentNode,
		bestDescendant:           NonExistentNode,
		weight:                   0,
		payloadHash:              payloadHash,
	}

	s.nodesIndices[root] = index
	s.payloadIndices[payloadHash] = index
	s.nodes = append(s.nodes, n)

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

	// Update parent with the best child and descendant only if it's available.
	if n.parent != NonExistentNode {
		if err := s.updateBestChildAndDescendant(parentIndex, index); err != nil {
			return n, err
		}
	}

	// Update metrics.
	processedBlockCount.Inc()
	nodeCount.Set(float64(len(s.nodes)))

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

// applyWeightChanges iterates backwards through the nodes in store. It checks all nodes parent
// and its best child. For each node, it updates the weight with input delta and
// back propagate the nodes' delta to its parents' delta. After scoring changes,
// the best child is then updated along with the best descendant. This function
// assumes the caller holds a lock in Store.nodesLock
func (s *Store) applyWeightChanges(
	ctx context.Context, newBalances []uint64, delta []int,
) error {
	_, span := trace.StartSpan(ctx, "protoArrayForkChoice.applyWeightChanges")
	defer span.End()

	// The length of the nodes can not be different than length of the delta.
	if len(s.nodes) != len(delta) {
		return errInvalidDeltaLength
	}

	// Proposer score defaults to 0.
	proposerScore := uint64(0)

	// Iterate backwards through all index to node in store.
	var err error
	for i := len(s.nodes) - 1; i >= 0; i-- {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		n := s.nodes[i]

		// There is no need to adjust the balances or manage parent of the zero hash, it
		// is an alias to the genesis block.
		if n.root == params.BeaconConfig().ZeroHash {
			continue
		}

		nodeDelta := delta[i]

		// If we have a node where the proposer boost was previously applied,
		// we then decrease the delta by the required score amount.
		s.proposerBoostLock.Lock()
		if s.previousProposerBoostRoot != params.BeaconConfig().ZeroHash && s.previousProposerBoostRoot == n.root {
			nodeDelta -= int(s.previousProposerBoostScore)
		}

		if s.proposerBoostRoot != params.BeaconConfig().ZeroHash && s.proposerBoostRoot == n.root {
			proposerScore, err = computeProposerBoostScore(newBalances)
			if err != nil {
				s.proposerBoostLock.Unlock()
				return err
			}
			iProposerScore, err := pmath.Int(proposerScore)
			if err != nil {
				s.proposerBoostLock.Unlock()
				return err
			}
			nodeDelta = nodeDelta + iProposerScore
		}
		s.proposerBoostLock.Unlock()

		// A node's weight can not be negative but the delta can be negative.
		if nodeDelta < 0 {
			d := uint64(-nodeDelta)
			if n.weight < d {
				s.proposerBoostLock.RLock()
				log.WithFields(logrus.Fields{
					"nodeDelta":                  d,
					"nodeRoot":                   fmt.Sprintf("%#x", bytesutil.Trunc(n.root[:])),
					"nodeWeight":                 n.weight,
					"proposerBoostRoot":          fmt.Sprintf("%#x", bytesutil.Trunc(s.proposerBoostRoot[:])),
					"previousProposerBoostRoot":  fmt.Sprintf("%#x", bytesutil.Trunc(s.previousProposerBoostRoot[:])),
					"previousProposerBoostScore": s.previousProposerBoostScore,
				}).Warning("node with invalid weight, setting it to zero")
				s.proposerBoostLock.RUnlock()
				n.weight = 0
			} else {
				n.weight -= d
			}
		} else {
			n.weight += uint64(nodeDelta)
		}

		// Update parent's best child and descendant if the node has a known parent.
		if n.parent != NonExistentNode {
			// Protection against node parent index out of bound. This should not happen.
			if int(n.parent) >= len(delta) {
				return errInvalidParentDelta
			}
			// Back propagate the nodes' delta to its parent.
			delta[n.parent] += nodeDelta
		}
	}

	// Set the previous boosted root and score.
	s.proposerBoostLock.Lock()
	s.previousProposerBoostRoot = s.proposerBoostRoot
	s.previousProposerBoostScore = proposerScore
	s.proposerBoostLock.Unlock()

	for i := len(s.nodes) - 1; i >= 0; i-- {
		n := s.nodes[i]
		if n.parent != NonExistentNode {
			if int(n.parent) >= len(delta) {
				return errInvalidParentDelta
			}
			if err := s.updateBestChildAndDescendant(n.parent, uint64(i)); err != nil {
				return err
			}
		}
	}

	return nil
}

// updateBestChildAndDescendant updates parent node's best child and descendant.
// It looks at input parent node and input child node and potentially modifies parent's best
// child and best descendant indices.
// There are four outcomes:
// 1.)  The child is already the best child, but it's now invalid due to a FFG change and should be removed.
// 2.)  The child is already the best child and the parent is updated with the new best descendant.
// 3.)  The child is not the best child but becomes the best child.
// 4.)  The child is not the best child and does not become the best child.
func (s *Store) updateBestChildAndDescendant(parentIndex, childIndex uint64) error {

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
	childLeadsToViableHead, err := s.leadsToViableHead(child)
	if err != nil {
		return err
	}

	// Define 3 variables for the 3 outcomes mentioned above. This is to
	// set `parent.bestChild` and `parent.bestDescendant` to. These
	// aliases are to assist readability.
	changeToNone := []uint64{NonExistentNode, NonExistentNode}
	bestDescendant := child.bestDescendant
	if bestDescendant == NonExistentNode {
		bestDescendant = childIndex
	}
	changeToChild := []uint64{childIndex, bestDescendant}
	noChange := []uint64{parent.bestChild, parent.bestDescendant}
	var newParentChild []uint64

	if parent.bestChild != NonExistentNode {
		if parent.bestChild == childIndex && !childLeadsToViableHead {
			// If the child is already the best child of the parent, but it's not viable for head,
			// we should remove it. (Outcome 1)
			newParentChild = changeToNone
		} else if parent.bestChild == childIndex {
			// If the child is already the best child of the parent, set it again to ensure the best
			// descendant of the parent is updated. (Outcome 2)
			newParentChild = changeToChild
		} else {
			// Protection against parent's best child going out of bound.
			if parent.bestChild > uint64(len(s.nodes)) {
				return errInvalidBestDescendantIndex
			}
			bestChild := s.nodes[parent.bestChild]
			// Is current parent's best child viable to be head? Based on justification and finalization rules.
			bestChildLeadsToViableHead, err := s.leadsToViableHead(bestChild)
			if err != nil {
				return err
			}

			if childLeadsToViableHead && !bestChildLeadsToViableHead {
				// The child leads to a viable head, but the current parent's best child doesn't.
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

// prune prunes the store with the new finalized root. The tree is only
// pruned if the number of the nodes in store has met prune threshold.
func (s *Store) prune(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "protoArrayForkChoice.prune")
	defer span.End()

	s.nodesLock.Lock()
	defer s.nodesLock.Unlock()
	s.checkpointsLock.RLock()
	finalizedRoot := s.finalizedCheckpoint.Root
	s.checkpointsLock.RUnlock()

	// Protection against invalid checkpoint
	finalizedIndex, ok := s.nodesIndices[finalizedRoot]
	if !ok {
		return errUnknownFinalizedRoot
	}

	// The number of the nodes has not met the prune threshold.
	// Pruning at small numbers incurs more cost than benefit.
	if finalizedIndex < s.pruneThreshold {
		return nil
	}

	canonicalNodesMap := make(map[uint64]uint64, uint64(len(s.nodes))-finalizedIndex)
	canonicalNodes := make([]*Node, 1, uint64(len(s.nodes))-finalizedIndex)
	finalizedNode := s.nodes[finalizedIndex]
	finalizedNode.parent = NonExistentNode
	canonicalNodes[0] = finalizedNode
	canonicalNodesMap[finalizedIndex] = uint64(0)

	for idx := uint64(0); idx < uint64(len(s.nodes)); idx++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		node := copyNode(s.nodes[idx])
		parentIdx, ok := canonicalNodesMap[node.parent]
		if ok {
			currentIndex := uint64(len(canonicalNodes))
			s.nodesIndices[node.root] = currentIndex
			s.payloadIndices[node.payloadHash] = currentIndex
			canonicalNodesMap[idx] = currentIndex
			node.parent = parentIdx
			canonicalNodes = append(canonicalNodes, node)
		} else {
			// Remove node that is not part of finalized branch.
			delete(s.nodesIndices, node.root)
			delete(s.canonicalNodes, node.root)
			delete(s.payloadIndices, node.payloadHash)
		}
	}
	s.nodesIndices[finalizedRoot] = uint64(0)
	s.canonicalNodes[finalizedRoot] = true
	s.payloadIndices[finalizedNode.payloadHash] = uint64(0)

	// Recompute the best child and descendant for each canonical nodes.
	for _, node := range canonicalNodes {
		if node.bestChild != NonExistentNode {
			node.bestChild = canonicalNodesMap[node.bestChild]
		}
		if node.bestDescendant != NonExistentNode {
			node.bestDescendant = canonicalNodesMap[node.bestDescendant]
		}
	}

	s.nodes = canonicalNodes
	prunedCount.Inc()
	return nil
}

// leadsToViableHead returns true if the node or the best descendant of the node is viable for head.
// Any node with diff finalized or justified epoch than the ones in fork choice store
// should not be viable to head.
func (s *Store) leadsToViableHead(node *Node) (bool, error) {
	if node.status == invalid {
		return false, nil
	}

	var bestDescendantViable bool
	bestDescendantIndex := node.bestDescendant

	// If the best descendant is not part of the leaves.
	if bestDescendantIndex != NonExistentNode {
		// Protection against out of bound, the best descendant index can not be
		// exceeds length of nodes list.
		if bestDescendantIndex >= uint64(len(s.nodes)) {
			return false, errInvalidBestDescendantIndex
		}

		bestDescendantNode := s.nodes[bestDescendantIndex]
		bestDescendantViable = s.viableForHead(bestDescendantNode)
	}

	// The node is viable as long as the best descendant is viable.
	return bestDescendantViable || s.viableForHead(node), nil
}

// viableForHead returns true if the node is viable to head.
// Any node with diff finalized or justified epoch than the ones in fork choice store
// should not be viable to head.
func (s *Store) viableForHead(node *Node) bool {
	s.checkpointsLock.RLock()
	defer s.checkpointsLock.RUnlock()
	// `node` is viable if its justified epoch and finalized epoch are the same as the one in `Store`.
	// It's also viable if we are in genesis epoch.
	justified := s.justifiedCheckpoint.Epoch == node.justifiedEpoch || s.justifiedCheckpoint.Epoch == 0
	finalized := s.finalizedCheckpoint.Epoch == node.finalizedEpoch || s.finalizedCheckpoint.Epoch == 0

	return justified && finalized
}

// Tips returns all possible chain heads (leaves of fork choice tree).
// Heads roots and heads slots are returned.
func (f *ForkChoice) Tips() ([][32]byte, []types.Slot) {

	// Deliberate choice to not preallocate space for below.
	// Heads cant be more than 2-3 in the worst case where pre-allocation will be 64 to begin with.
	headsRoots := make([][32]byte, 0)
	headsSlots := make([]types.Slot, 0)

	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()
	for _, node := range f.store.nodes {
		// Possible heads have no children.
		if node.BestDescendant() == NonExistentNode && node.BestChild() == NonExistentNode {
			headsRoots = append(headsRoots, node.Root())
			headsSlots = append(headsSlots, node.Slot())
		}
	}
	return headsRoots, headsSlots
}

// InsertSlashedIndex adds the given slashed validator index to the
// store-tracked list. Votes from these validators are not accounted for
// in forkchoice.
func (f *ForkChoice) InsertSlashedIndex(ctx context.Context, index types.ValidatorIndex) {
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

	nodeIndex, ok := f.store.nodesIndices[f.votes[index].currentRoot]
	if !ok {
		return
	}

	var node *Node
	for nodeIndex != NonExistentNode {
		if ctx.Err() != nil {
			return
		}

		node = f.store.nodes[nodeIndex]
		if node == nil {
			return
		}

		if node.weight < f.balances[index] {
			node.weight = 0
		} else {
			node.weight -= f.balances[index]
		}
		nodeIndex = node.parent
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

// InsertOptimisticChain inserts all nodes corresponding to blocks in the slice
// `blocks`. It includes all blocks **except** the first one.
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
	return f.store.lastHeadRoot
}

// FinalizedPayloadBlockHash returns the hash of the payload at the finalized checkpoint
func (f *ForkChoice) FinalizedPayloadBlockHash() [32]byte {
	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()
	root := f.FinalizedCheckpoint().Root
	idx := f.store.nodesIndices[root]
	if idx >= uint64(len(f.store.nodes)) {
		// This should not happen
		return [32]byte{}
	}
	node := f.store.nodes[idx]
	return node.payloadHash
}

// JustifiedPayloadBlockHash returns the hash of the payload at the justified checkpoint
func (f *ForkChoice) JustifiedPayloadBlockHash() [32]byte {
	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()
	root := f.JustifiedCheckpoint().Root
	idx := f.store.nodesIndices[root]
	if idx >= uint64(len(f.store.nodes)) {
		// This should not happen
		return [32]byte{}
	}
	node := f.store.nodes[idx]
	return node.payloadHash
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

func (*ForkChoice) ForkChoiceDump(_ context.Context) (*v1.ForkChoiceResponse, error) {
	return nil, errors.New("ForkChoiceDump is not supported by protoarray")
}
