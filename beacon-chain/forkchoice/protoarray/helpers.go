package protoarray

import (
	"context"

	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	pmath "github.com/prysmaticlabs/prysm/v3/math"
	"go.opencensus.io/trace"
)

// This computes validator balance delta from validator votes.
// It returns a list of deltas that represents the difference between old
// balances and new balances. This function assumes the caller holds a lock in
// Store.nodesLock and Store.votesLock
func computeDeltas(
	ctx context.Context,
	count int,
	blockIndices map[[32]byte]uint64,
	votes []Vote,
	oldBalances, newBalances []uint64,
	slashedIndices map[types.ValidatorIndex]bool,
) ([]int, []Vote, error) {
	ctx, span := trace.StartSpan(ctx, "doublyLinkedForkchoice.computeDeltas")
	defer span.End()

	deltas := make([]int, count)

	for validatorIndex, vote := range votes {
		// Skip if validator has been slashed
		if slashedIndices[types.ValidatorIndex(validatorIndex)] {
			continue
		}
		oldBalance := uint64(0)
		newBalance := uint64(0)

		// Skip if validator has never voted for current root and next root (i.e. if the
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
				if nextDeltaIndex >= uint64(len(deltas)) {
					return nil, nil, errInvalidNodeDelta
				}
				delta, err := pmath.Int(newBalance)
				if err != nil {
					return nil, nil, err
				}
				deltas[nextDeltaIndex] += delta
			}

			currentDeltaIndex, ok := blockIndices[vote.currentRoot]
			if ok {
				// Protection against out of bound (same as above)
				if currentDeltaIndex >= uint64(len(deltas)) {
					return nil, nil, errInvalidNodeDelta
				}
				delta, err := pmath.Int(oldBalance)
				if err != nil {
					return nil, nil, err
				}
				deltas[currentDeltaIndex] -= delta
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

	return &Node{
		slot:           node.slot,
		root:           node.root,
		parent:         node.parent,
		payloadHash:    node.payloadHash,
		justifiedEpoch: node.justifiedEpoch,
		finalizedEpoch: node.finalizedEpoch,
		weight:         node.weight,
		bestChild:      node.bestChild,
		bestDescendant: node.bestDescendant,
		status:         node.status,
	}
}
