package protoarray

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// This defines the minimal number of block nodes has to be in in the tree
// before getting pruned upon new finalization.
const defaultPruneThreshold = 64

// New initializes a new fork choice store.
func New(justifiedEpoch uint64, finalizedEpoch uint64, finalizedRoot [32]byte) *ForkChoice {
	s := &Store{
		justifiedEpoch: justifiedEpoch,
		finalizedEpoch: finalizedEpoch,
		finalizedRoot:  finalizedRoot,
		nodes:          make([]*Node, 0),
		nodeIndices:    make(map[[32]byte]uint64),
		pruneThreshold: defaultPruneThreshold,
	}

	b := make([]uint64, 0)
	v := make([]Vote, 0)

	return &ForkChoice{store: s, balances: b, votes: v}
}

// Head returns the head root from fork choice store.
func (f *ForkChoice) Head(ctx context.Context, finalizedEpoch uint64, justifiedRoot [32]byte, justifiedStateBalances []uint64, justifiedEpoch uint64) ([32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "protoArrayForkChoice.Head")
	defer span.End()

	newBalances := justifiedStateBalances

	deltas, newVotes, err := computeDeltas(ctx, f.store.nodeIndices, f.votes, f.balances, newBalances)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "Could not compute deltas")
	}
	f.votes = newVotes

	if err := f.store.applyScoreChanges(ctx, justifiedEpoch, finalizedEpoch, deltas); err != nil {
		return [32]byte{}, errors.Wrap(err, "Could not apply score changes")
	}
	f.balances = newBalances

	return f.store.head(ctx, justifiedRoot)
}

// ProcessAttestation processes attestation for vote accounting to be used for fork choice.
func (f *ForkChoice) ProcessAttestation(ctx context.Context, validatorIndices []uint64, blockRoot [32]byte, targetEpoch uint64) {
	ctx, span := trace.StartSpan(ctx, "protoArrayForkChoice.ProcessAttestation")
	defer span.End()

	for _, index := range validatorIndices {
		// Validator indices will grow the votes cache on demand.
		for index >= uint64(len(f.votes)) {
			f.votes = append(f.votes, Vote{currentRoot: params.BeaconConfig().ZeroHash, nextRoot: params.BeaconConfig().ZeroHash})
		}

		newVote := f.votes[index].nextRoot == params.BeaconConfig().ZeroHash && f.votes[index].currentRoot == params.BeaconConfig().ZeroHash
		if newVote || targetEpoch > f.votes[index].nextEpoch {
			f.votes[index].nextEpoch = targetEpoch
			f.votes[index].nextRoot = blockRoot
		}
	}
}

// ProcessBlock processes block by inserting it to the fork choice store.
func (f *ForkChoice) ProcessBlock(ctx context.Context, slot uint64, blockRoot [32]byte, parentRoot [32]byte, justifiedEpoch uint64, finalizedEpoch uint64) error {
	ctx, span := trace.StartSpan(ctx, "protoArrayForkChoice.ProcessBlock")
	defer span.End()

	return f.store.insert(ctx, slot, blockRoot, parentRoot, justifiedEpoch, finalizedEpoch)
}

// Prune prunes the fork choice store with the new finalized root. The store is only pruned if the input
// root is different than the current store finalized root, and the number of the store has met prune threshold.
func (f *ForkChoice) Prune(ctx context.Context, finalizedRoot [32]byte) error {
	return f.store.prune(ctx, finalizedRoot)
}
