package protoarray

import (
	"context"

	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// This computes validator balance delta from validator votes.
// It returns a list of deltas that represents the difference between old balances and new balances.
func computeDeltas(
	ctx context.Context,
	blockIndices map[[32]byte]uint64,
	votes []Vote,
	oldBalances []uint64,
	newBalances []uint64,
) ([]int, []Vote, error) {
	ctx, span := trace.StartSpan(ctx, "protoArrayForkChoice.computeDeltas")
	defer span.End()

	deltas := make([]int, len(blockIndices))

	for validatorIndex, vote := range votes {
		oldBalance := uint64(0)
		newBalance := uint64(0)

		// Skip if validator has never voted for current root and next root (ie. if the
		// votes are zero hash aka genesis block), there's nothing to compute.
		if vote.currentRoot == params.BeaconConfig().ZeroHash && vote.nextRoot == params.BeaconConfig().ZeroHash {
			continue
		}

		// If the validator index did not exist in `oldBalance` or `newBalance` list above, the balance is just 0.
		if validatorIndex < len(oldBalances) {
			oldBalance = oldBalances[validatorIndex]
		}
		if validatorIndex < len(newBalances) {
			newBalance = newBalances[validatorIndex]
		}

		// Perform delta only if the validator's balance or vote has changed.
		if vote.currentRoot != vote.nextRoot || oldBalance != newBalance {
			// Ignore the vote if it's not known in `blockIndices`,
			// that means we have not seen the block before.
			nextDeltaIndex, ok := blockIndices[vote.nextRoot]
			if ok {
				// Protection against out of bound, the `nextDeltaIndex` which defines
				// the block location in the dag can not exceed the total `delta` length.
				if int(nextDeltaIndex) >= len(deltas) {
					return nil, nil, errInvalidNodeDelta
				}
				deltas[nextDeltaIndex] += int(newBalance)
			}

			currentDeltaIndex, ok := blockIndices[vote.currentRoot]
			if ok {
				// Protection against out of bound (same as above)
				if int(currentDeltaIndex) >= len(deltas) {
					return nil, nil, errInvalidNodeDelta
				}
				deltas[currentDeltaIndex] -= int(oldBalance)
			}
		}

		// Rotate the validator vote.
		vote.currentRoot = vote.nextRoot
		votes[validatorIndex] = vote
	}

	return deltas, votes, nil
}

// This return a copy of the proto array node object.
func copyNode(node *Node) *Node {
	if node == nil {
		return &Node{}
	}

	copiedRoot := [32]byte{}
	copy(copiedRoot[:], node.Root[:])

	return &Node{
		Slot:           node.Slot,
		Root:           copiedRoot,
		Parent:         node.Parent,
		JustifiedEpoch: node.JustifiedEpoch,
		FinalizedEpoch: node.FinalizedEpoch,
		Weight:         node.Weight,
		BestChild:      node.BestChild,
		BestDescendant: node.BestDescendant,
	}
}
