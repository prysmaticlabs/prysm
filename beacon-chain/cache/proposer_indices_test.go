package cache

import (
	"strconv"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestProposerKeyFn_OK(t *testing.T) {
	item := &ProposerIndices{
		BlockRoot:       [32]byte{'A'},
		ProposerIndices: []uint64{1, 2, 3, 4, 5},
	}

	k, err := proposerIndicesKeyFn(item)
	require.NoError(t, err)
	assert.Equal(t, key(item.BlockRoot), k)
}

func TestProposerKeyFn_InvalidObj(t *testing.T) {
	_, err := proposerIndicesKeyFn("bad")
	assert.Equal(t, ErrNotProposerIndices, err)
}

func TestProposerCache_AddProposerIndicesList(t *testing.T) {
	cache := NewProposerIndicesCache()
	bRoot := [32]byte{'A'}
	indices, err := cache.ProposerIndices(bRoot)
	require.NoError(t, err)
	if indices != nil {
		t.Error("Expected committee count not to exist in empty cache")
	}
	require.NoError(t, cache.AddProposerIndices(&ProposerIndices{
		ProposerIndices: indices,
		BlockRoot:       bRoot,
	}))

	received, err := cache.ProposerIndices(bRoot)
	require.NoError(t, err)
	assert.DeepEqual(t, received, indices)

	item := &ProposerIndices{BlockRoot: [32]byte{'B'}, ProposerIndices: []uint64{1, 2, 3, 4, 5, 6}}
	require.NoError(t, cache.AddProposerIndices(item))

	received, err = cache.ProposerIndices(item.BlockRoot)
	require.NoError(t, err)
	assert.DeepEqual(t, item.ProposerIndices, received)
}

func TestProposerCache_CanRotate(t *testing.T) {
	cache := NewProposerIndicesCache()
	for i := 0; i < int(maxProposerIndicesCacheSize)+1; i++ {
		s := []byte(strconv.Itoa(i))
		item := &ProposerIndices{BlockRoot: bytesutil.ToBytes32(s)}
		require.NoError(t, cache.AddProposerIndices(item))
	}

	k := cache.ProposerIndicesCache.ListKeys()
	assert.Equal(t, maxProposerIndicesCacheSize, uint64(len(k)))
}
