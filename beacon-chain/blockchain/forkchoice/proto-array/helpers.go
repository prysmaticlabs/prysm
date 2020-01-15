package proto_array

import (
	"errors"

	"github.com/prysmaticlabs/prysm/shared/params"
)

// This computes balance delta from votes. It returns a list of deltas that represents the difference
// between old balances and new balances.
func computeDeltas(indices map[[32]byte]uint64, votes []Vote, oldBalances []uint64, newBalances []uint64) ([]int, error) {
	deltas := make([]int, len(indices))

	for validatorIndex, vote := range votes {
		oldBalance := uint64(0)
		newBalance := uint64(0)

		// Skip if validator has never voted or voted for zero hash (ie genesis block)
		if vote.currentRoot == params.BeaconConfig().ZeroHash || vote.nextRoot == params.BeaconConfig().ZeroHash {
			continue
		}

		// If the validator did not exist in `old` or `new balance` list before, the balance is just 0.
		if validatorIndex < len(oldBalances) {
			oldBalance = oldBalances[validatorIndex]
		}
		if validatorIndex < len(newBalances) {
			newBalance = newBalances[validatorIndex]
		}

		// Perform delta only if vote has changed and balance has changed.
		if vote.currentRoot != vote.nextRoot || oldBalance != newBalance {
			// Ignore the vote if it's not known in `indices`
			nextDeltaIndex, ok := indices[vote.nextRoot]
			if !ok {
				return nil, errors.New("vote is not a key in indices")
			}
			deltas[nextDeltaIndex] += int(newBalance)

			currentDeltaIndex, ok := indices[vote.currentRoot]
			if !ok {
				return nil, errors.New("vote is not a key in indices")
			}
			deltas[currentDeltaIndex] -= int(oldBalance)
		}

		vote.currentRoot = vote.nextRoot
	}

	return deltas, nil
}
