package protoarray

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestComputeDelta_ZeroHash(t *testing.T) {
	validatorCount := uint64(16)
	indices := make(map[[32]byte]uint64)
	votes := make([]Vote, 0)
	oldBalances := make([]uint64, 0)
	newBalances := make([]uint64, 0)

	for i := uint64(0); i < validatorCount; i++ {
		indices[indexToHash(i)] = i
		votes = append(votes, Vote{params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 0})
		oldBalances = append(oldBalances, 0)
		newBalances = append(newBalances, 0)
	}

	delta, _, err := computeDeltas(context.Background(), indices, votes, oldBalances, newBalances)
	require.NoError(t, err)
	assert.Equal(t, int(validatorCount), len(delta))

	for _, d := range delta {
		assert.Equal(t, 0, d)
	}
	for _, vote := range votes {
		assert.Equal(t, vote.currentRoot, vote.nextRoot, "The vote should not have changed")
	}
}

func TestComputeDelta_AllVoteTheSame(t *testing.T) {
	validatorCount := uint64(16)
	balance := uint64(32)
	indices := make(map[[32]byte]uint64)
	votes := make([]Vote, 0)
	oldBalances := make([]uint64, 0)
	newBalances := make([]uint64, 0)

	for i := uint64(0); i < validatorCount; i++ {
		indices[indexToHash(i)] = i
		votes = append(votes, Vote{params.BeaconConfig().ZeroHash, indexToHash(0), 0})
		oldBalances = append(oldBalances, balance)
		newBalances = append(newBalances, balance)
	}

	delta, _, err := computeDeltas(context.Background(), indices, votes, oldBalances, newBalances)
	require.NoError(t, err)
	assert.Equal(t, int(validatorCount), len(delta))

	for i, d := range delta {
		if i == 0 {
			assert.Equal(t, balance*validatorCount, uint64(d))
		} else {
			assert.Equal(t, 0, d)
		}
	}

	for _, vote := range votes {
		assert.Equal(t, vote.currentRoot, vote.nextRoot, "The vote should not have changed")
	}
}

func TestComputeDelta_DifferentVotes(t *testing.T) {
	validatorCount := uint64(16)
	balance := uint64(32)
	indices := make(map[[32]byte]uint64)
	votes := make([]Vote, 0)
	oldBalances := make([]uint64, 0)
	newBalances := make([]uint64, 0)

	for i := uint64(0); i < validatorCount; i++ {
		indices[indexToHash(i)] = i
		votes = append(votes, Vote{params.BeaconConfig().ZeroHash, indexToHash(i), 0})
		oldBalances = append(oldBalances, balance)
		newBalances = append(newBalances, balance)
	}

	delta, _, err := computeDeltas(context.Background(), indices, votes, oldBalances, newBalances)
	require.NoError(t, err)
	assert.Equal(t, int(validatorCount), len(delta))

	for _, d := range delta {
		assert.Equal(t, balance, uint64(d))
	}

	for _, vote := range votes {
		assert.Equal(t, vote.currentRoot, vote.nextRoot, "The vote should not have changed")
	}
}

func TestComputeDelta_MovingVotes(t *testing.T) {
	validatorCount := uint64(16)
	balance := uint64(32)
	indices := make(map[[32]byte]uint64)
	votes := make([]Vote, 0)
	oldBalances := make([]uint64, 0)
	newBalances := make([]uint64, 0)

	lastIndex := uint64(len(indices) - 1)
	for i := uint64(0); i < validatorCount; i++ {
		indices[indexToHash(i)] = i
		votes = append(votes, Vote{indexToHash(0), indexToHash(lastIndex), 0})
		oldBalances = append(oldBalances, balance)
		newBalances = append(newBalances, balance)
	}

	delta, _, err := computeDeltas(context.Background(), indices, votes, oldBalances, newBalances)
	require.NoError(t, err)
	assert.Equal(t, int(validatorCount), len(delta))

	for i, d := range delta {
		if i == 0 {
			assert.Equal(t, -int(balance*validatorCount), d, "First root should have negative delta")
		} else if i == int(lastIndex) {
			assert.Equal(t, int(balance*validatorCount), d, "Last root should have positive delta")
		} else {
			assert.Equal(t, 0, d)
		}
	}

	for _, vote := range votes {
		assert.Equal(t, vote.currentRoot, vote.nextRoot, "The vote should not have changed")
	}
}

func TestComputeDelta_MoveOutOfTree(t *testing.T) {
	balance := uint64(32)
	indices := make(map[[32]byte]uint64)
	votes := make([]Vote, 0)
	oldBalances := []uint64{balance, balance}
	newBalances := []uint64{balance, balance}

	indices[indexToHash(1)] = 0

	votes = append(votes, Vote{indexToHash(1), params.BeaconConfig().ZeroHash, 0})
	votes = append(votes, Vote{indexToHash(1), [32]byte{'A'}, 0})

	delta, _, err := computeDeltas(context.Background(), indices, votes, oldBalances, newBalances)
	require.NoError(t, err)
	assert.Equal(t, 1, len(delta))
	assert.Equal(t, 0-2*int(balance), delta[0])

	for _, vote := range votes {
		assert.Equal(t, vote.currentRoot, vote.nextRoot, "The vote should not have changed")
	}
}

func TestComputeDelta_ChangingBalances(t *testing.T) {
	oldBalance := uint64(32)
	newBalance := oldBalance * 2
	validatorCount := uint64(16)
	indices := make(map[[32]byte]uint64)
	votes := make([]Vote, 0)
	oldBalances := make([]uint64, 0)
	newBalances := make([]uint64, 0)

	indices[indexToHash(1)] = 0

	for i := uint64(0); i < validatorCount; i++ {
		indices[indexToHash(i)] = i
		votes = append(votes, Vote{indexToHash(0), indexToHash(1), 0})
		oldBalances = append(oldBalances, oldBalance)
		newBalances = append(newBalances, newBalance)
	}

	delta, _, err := computeDeltas(context.Background(), indices, votes, oldBalances, newBalances)
	require.NoError(t, err)
	assert.Equal(t, 16, len(delta))

	for i, d := range delta {
		if i == 0 {
			assert.Equal(t, -int(oldBalance*validatorCount), d, "First root should have negative delta")
		} else if i == 1 {
			assert.Equal(t, int(newBalance*validatorCount), d, "Last root should have positive delta")
		} else {
			assert.Equal(t, 0, d)
		}
	}

	for _, vote := range votes {
		assert.Equal(t, vote.currentRoot, vote.nextRoot, "The vote should not have changed")
	}
}

func TestComputeDelta_ValidatorAppear(t *testing.T) {
	balance := uint64(32)
	indices := make(map[[32]byte]uint64)
	votes := make([]Vote, 0)
	oldBalances := []uint64{balance}
	newBalances := []uint64{balance, balance}

	indices[indexToHash(1)] = 0
	indices[indexToHash(2)] = 1

	votes = append(votes, Vote{indexToHash(1), indexToHash(2), 0})
	votes = append(votes, Vote{indexToHash(1), indexToHash(2), 0})

	delta, _, err := computeDeltas(context.Background(), indices, votes, oldBalances, newBalances)
	require.NoError(t, err)
	assert.Equal(t, 2, len(delta))
	assert.Equal(t, 0-int(balance), delta[0])
	assert.Equal(t, 2*int(balance), delta[1])

	for _, vote := range votes {
		assert.Equal(t, vote.currentRoot, vote.nextRoot, "The vote should not have changed")
	}
}

func TestComputeDelta_ValidatorDisappears(t *testing.T) {
	balance := uint64(32)
	indices := make(map[[32]byte]uint64)
	votes := make([]Vote, 0)
	oldBalances := []uint64{balance, balance}
	newBalances := []uint64{balance}

	indices[indexToHash(1)] = 0
	indices[indexToHash(2)] = 1

	votes = append(votes, Vote{indexToHash(1), indexToHash(2), 0})
	votes = append(votes, Vote{indexToHash(1), indexToHash(2), 0})

	delta, _, err := computeDeltas(context.Background(), indices, votes, oldBalances, newBalances)
	require.NoError(t, err)
	assert.Equal(t, 2, len(delta))
	assert.Equal(t, 0-2*int(balance), delta[0])
	assert.Equal(t, int(balance), delta[1])

	for _, vote := range votes {
		assert.Equal(t, vote.currentRoot, vote.nextRoot, "The vote should not have changed")
	}
}

func indexToHash(i uint64) [32]byte {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], i)
	return hashutil.Hash(b[:])
}
