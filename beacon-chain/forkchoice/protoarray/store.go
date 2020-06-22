package protoarray

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// This defines the minimal number of block nodes that can be in the tree
// before getting pruned upon new finalization.
const defaultPruneThreshold = 256

// This tracks the last reported head root. Used for metrics.
var lastHeadRoot [32]byte

// New initializes a new fork choice store.
func New(justifiedEpoch uint64, finalizedEpoch uint64, finalizedRoot [32]byte) *ForkChoice {
	s := &Store{
		JustifiedEpoch: justifiedEpoch,
		FinalizedEpoch: finalizedEpoch,
		finalizedRoot:  finalizedRoot,
		Nodes:          make([]*Node, 0),
		NodeIndices:    make(map[[32]byte]uint64),
		PruneThreshold: defaultPruneThreshold,
	}

	b := make([]uint64, 0)
	v := make([]Vote, 0)

	return &ForkChoice{store: s, balances: b, votes: v}
}

// Head returns the head root from fork choice store.
// It firsts computes validator's balance changes then recalculates block tree from leaves to root.
func (f *ForkChoice) Head(ctx context.Context, justifiedEpoch uint64, justifiedRoot [32]byte, justifiedStateBalances []uint64, finalizedEpoch uint64) ([32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "protoArrayForkChoice.Head")
	defer span.End()
	calledHeadCount.Inc()

	newBalances := justifiedStateBalances

	// Using the read lock is ok here, rest of the operations below is read only.
	// The only time it writes to node indices is inserting and pruning blocks from the store.
	f.store.nodeIndicesLock.RLock()
	defer f.store.nodeIndicesLock.RUnlock()
	deltas, newVotes, err := computeDeltas(ctx, f.store.NodeIndices, f.votes, f.balances, newBalances)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "Could not compute deltas")
	}
	f.votes = newVotes

	if err := f.store.applyWeightChanges(ctx, justifiedEpoch, finalizedEpoch, deltas); err != nil {
		return [32]byte{}, errors.Wrap(err, "Could not apply score changes")
	}
	f.balances = newBalances

	return f.store.head(ctx, justifiedRoot)
}

// ProcessAttestation processes attestation for vote accounting, it iterates around validator indices
// and update their votes accordingly.
func (f *ForkChoice) ProcessAttestation(ctx context.Context, validatorIndices []uint64, blockRoot [32]byte, targetEpoch uint64) {
	ctx, span := trace.StartSpan(ctx, "protoArrayForkChoice.ProcessAttestation")
	defer span.End()

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

// ProcessBlock processes a new block by inserting it to the fork choice store.
func (f *ForkChoice) ProcessBlock(ctx context.Context, slot uint64, blockRoot [32]byte, parentRoot [32]byte, graffiti [32]byte, justifiedEpoch uint64, finalizedEpoch uint64) error {
	ctx, span := trace.StartSpan(ctx, "protoArrayForkChoice.ProcessBlock")
	defer span.End()

	return f.store.insert(ctx, slot, blockRoot, parentRoot, graffiti, justifiedEpoch, finalizedEpoch)
}

// Prune prunes the fork choice store with the new finalized root. The store is only pruned if the input
// root is different than the current store finalized root, and the number of the store has met prune threshold.
func (f *ForkChoice) Prune(ctx context.Context, finalizedRoot [32]byte) error {
	return f.store.prune(ctx, finalizedRoot)
}

// Nodes returns the copied list of block nodes in the fork choice store.
func (f *ForkChoice) Nodes() []*Node {
	f.store.nodeIndicesLock.RLock()
	defer f.store.nodeIndicesLock.RUnlock()

	cpy := make([]*Node, len(f.store.Nodes))
	copy(cpy, f.store.Nodes)
	return cpy
}

// Store returns the fork choice store object which contains all the information regarding proto array fork choice.
func (f *ForkChoice) Store() *Store {
	f.store.nodeIndicesLock.Lock()
	defer f.store.nodeIndicesLock.Unlock()
	return f.store
}

// Node returns the copied node in the fork choice store.
func (f *ForkChoice) Node(root [32]byte) *Node {
	f.store.nodeIndicesLock.RLock()
	defer f.store.nodeIndicesLock.RUnlock()

	index, ok := f.store.NodeIndices[root]
	if !ok {
		return nil
	}

	return copyNode(f.store.Nodes[index])
}

// HasNode returns true if the node exists in fork choice store,
// false else wise.
func (f *ForkChoice) HasNode(root [32]byte) bool {
	f.store.nodeIndicesLock.RLock()
	defer f.store.nodeIndicesLock.RUnlock()

	_, ok := f.store.NodeIndices[root]
	return ok
}
