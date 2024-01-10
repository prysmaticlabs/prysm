//go:build !fuzz

package cache

import (
	"testing"

	forkchoicetypes "github.com/prysmaticlabs/prysm/v4/beacon-chain/forkchoice/types"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestProposerCache_Set(t *testing.T) {
	cache := NewProposerIndicesCache()
	bRoot := [32]byte{'A'}
	indices, ok := cache.ProposerIndices(0, bRoot)
	require.Equal(t, false, ok)
	emptyIndices := [fieldparams.SlotsPerEpoch]primitives.ValidatorIndex{}
	require.Equal(t, indices, emptyIndices, "Expected committee count not to exist in empty cache")
	emptyIndices[0] = 1
	cache.Set(0, bRoot, emptyIndices)

	received, ok := cache.ProposerIndices(0, bRoot)
	require.Equal(t, true, ok)
	require.Equal(t, received, emptyIndices)

	newRoot := [32]byte{'B'}
	copy(emptyIndices[3:], []primitives.ValidatorIndex{1, 2, 3, 4, 5, 6})
	cache.Set(0, newRoot, emptyIndices)

	received, ok = cache.ProposerIndices(0, newRoot)
	require.Equal(t, true, ok)
	require.Equal(t, emptyIndices, received)
}

func TestProposerCache_CheckpointAndPrune(t *testing.T) {
	cache := NewProposerIndicesCache()
	indices := [fieldparams.SlotsPerEpoch]primitives.ValidatorIndex{}
	root := [32]byte{'a'}
	cpRoot := [32]byte{'b'}
	copy(indices[3:], []primitives.ValidatorIndex{1, 2, 3, 4, 5, 6})
	for i := 1; i < 10; i++ {
		cache.Set(primitives.Epoch(i), root, indices)
		cache.SetCheckpoint(forkchoicetypes.Checkpoint{Epoch: primitives.Epoch(i - 1), Root: cpRoot}, root)
	}
	received, ok := cache.ProposerIndices(1, root)
	require.Equal(t, true, ok)
	require.Equal(t, indices, received)

	received, ok = cache.ProposerIndices(4, root)
	require.Equal(t, true, ok)
	require.Equal(t, indices, received)

	received, ok = cache.ProposerIndices(9, root)
	require.Equal(t, true, ok)
	require.Equal(t, indices, received)

	received, ok = cache.IndicesFromCheckpoint(forkchoicetypes.Checkpoint{Epoch: 0, Root: cpRoot})
	require.Equal(t, true, ok)
	require.Equal(t, indices, received)

	received, ok = cache.IndicesFromCheckpoint(forkchoicetypes.Checkpoint{Epoch: 3, Root: cpRoot})
	require.Equal(t, true, ok)
	require.Equal(t, indices, received)

	received, ok = cache.IndicesFromCheckpoint(forkchoicetypes.Checkpoint{Epoch: 4, Root: cpRoot})
	require.Equal(t, true, ok)
	require.Equal(t, indices, received)

	received, ok = cache.IndicesFromCheckpoint(forkchoicetypes.Checkpoint{Epoch: 8, Root: cpRoot})
	require.Equal(t, true, ok)
	require.Equal(t, indices, received)

	cache.Prune(5)

	emptyIndices := [fieldparams.SlotsPerEpoch]primitives.ValidatorIndex{}
	received, ok = cache.ProposerIndices(1, root)
	require.Equal(t, false, ok)
	require.Equal(t, emptyIndices, received)

	received, ok = cache.ProposerIndices(4, root)
	require.Equal(t, false, ok)
	require.Equal(t, emptyIndices, received)

	received, ok = cache.ProposerIndices(9, root)
	require.Equal(t, true, ok)
	require.Equal(t, indices, received)

	received, ok = cache.IndicesFromCheckpoint(forkchoicetypes.Checkpoint{Epoch: 0, Root: cpRoot})
	require.Equal(t, false, ok)
	require.Equal(t, emptyIndices, received)

	received, ok = cache.IndicesFromCheckpoint(forkchoicetypes.Checkpoint{Epoch: 3, Root: cpRoot})
	require.Equal(t, false, ok)
	require.Equal(t, emptyIndices, received)

	received, ok = cache.IndicesFromCheckpoint(forkchoicetypes.Checkpoint{Epoch: 4, Root: cpRoot})
	require.Equal(t, true, ok)
	require.Equal(t, indices, received)

	received, ok = cache.IndicesFromCheckpoint(forkchoicetypes.Checkpoint{Epoch: 8, Root: cpRoot})
	require.Equal(t, true, ok)
	require.Equal(t, indices, received)

}
