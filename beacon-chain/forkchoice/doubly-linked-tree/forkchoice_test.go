package doublylinkedtree

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/hash"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestForkChoice_UpdateBalancesPositiveChange(t *testing.T) {
	f := setup(0, 0)
	ctx := context.Background()
	require.NoError(t, f.ProcessBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, 0, 0))
	require.NoError(t, f.ProcessBlock(ctx, 2, indexToHash(2), indexToHash(1), 0, 0))
	require.NoError(t, f.ProcessBlock(ctx, 3, indexToHash(3), indexToHash(2), 0, 0))

	f.votes = []Vote{
		{indexToHash(1), indexToHash(1), 0},
		{indexToHash(2), indexToHash(2), 0},
		{indexToHash(3), indexToHash(3), 0},
	}

	// Each node gets one unique vote. The weight should look like 103 <- 102 <- 101 because
	// they get propagated back.
	require.NoError(t, f.updateBalances([]uint64{10, 20, 30}))
	s := f.store
	assert.Equal(t, uint64(10), s.nodeByRoot[indexToHash(1)].balance)
	assert.Equal(t, uint64(20), s.nodeByRoot[indexToHash(2)].balance)
	assert.Equal(t, uint64(30), s.nodeByRoot[indexToHash(3)].balance)
}

func TestForkChoice_UpdateBalancesNegativeChange(t *testing.T) {
	f := setup(0, 0)
	ctx := context.Background()
	require.NoError(t, f.ProcessBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, 0, 0))
	require.NoError(t, f.ProcessBlock(ctx, 2, indexToHash(2), indexToHash(1), 0, 0))
	require.NoError(t, f.ProcessBlock(ctx, 3, indexToHash(3), indexToHash(2), 0, 0))
	s := f.store
	s.nodeByRoot[indexToHash(1)].balance = 100
	s.nodeByRoot[indexToHash(2)].balance = 100
	s.nodeByRoot[indexToHash(3)].balance = 100

	f.balances = []uint64{100, 100, 100}
	f.votes = []Vote{
		{indexToHash(1), indexToHash(1), 0},
		{indexToHash(2), indexToHash(2), 0},
		{indexToHash(3), indexToHash(3), 0},
	}

	require.NoError(t, f.updateBalances([]uint64{10, 20, 30}))
	assert.Equal(t, uint64(10), s.nodeByRoot[indexToHash(1)].balance)
	assert.Equal(t, uint64(20), s.nodeByRoot[indexToHash(2)].balance)
	assert.Equal(t, uint64(30), s.nodeByRoot[indexToHash(3)].balance)
}

func TestForkChoice_IsCanonical(t *testing.T) {
	f := setup(1, 1)
	ctx := context.Background()
	require.NoError(t, f.ProcessBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.ProcessBlock(ctx, 2, indexToHash(2), params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.ProcessBlock(ctx, 3, indexToHash(3), indexToHash(1), 1, 1))
	require.NoError(t, f.ProcessBlock(ctx, 4, indexToHash(4), indexToHash(2), 1, 1))
	require.NoError(t, f.ProcessBlock(ctx, 5, indexToHash(5), indexToHash(4), 1, 1))
	require.NoError(t, f.ProcessBlock(ctx, 6, indexToHash(6), indexToHash(5), 1, 1))

	require.Equal(t, true, f.IsCanonical(params.BeaconConfig().ZeroHash))
	require.Equal(t, false, f.IsCanonical(indexToHash(1)))
	require.Equal(t, true, f.IsCanonical(indexToHash(2)))
	require.Equal(t, false, f.IsCanonical(indexToHash(3)))
	require.Equal(t, true, f.IsCanonical(indexToHash(4)))
	require.Equal(t, true, f.IsCanonical(indexToHash(5)))
	require.Equal(t, true, f.IsCanonical(indexToHash(6)))
}

func TestForkChoice_IsCanonicalReorg(t *testing.T) {
	f := setup(1, 1)
	ctx := context.Background()
	require.NoError(t, f.ProcessBlock(ctx, 1, [32]byte{'1'}, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.ProcessBlock(ctx, 2, [32]byte{'2'}, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.ProcessBlock(ctx, 3, [32]byte{'3'}, [32]byte{'1'}, 1, 1))
	require.NoError(t, f.ProcessBlock(ctx, 4, [32]byte{'4'}, [32]byte{'2'}, 1, 1))
	require.NoError(t, f.ProcessBlock(ctx, 5, [32]byte{'5'}, [32]byte{'4'}, 1, 1))
	require.NoError(t, f.ProcessBlock(ctx, 6, [32]byte{'6'}, [32]byte{'5'}, 1, 1))

	f.store.nodesLock.Lock()
	f.store.nodeByRoot[[32]byte{'3'}].balance = 10
	require.NoError(t, f.store.treeRootNode.applyWeightChanges(ctx))
	require.Equal(t, uint64(10), f.store.nodeByRoot[[32]byte{'1'}].weight)
	require.Equal(t, uint64(0), f.store.nodeByRoot[[32]byte{'2'}].weight)

	require.NoError(t, f.store.treeRootNode.updateBestDescendant(ctx, 1, 1))
	require.DeepEqual(t, [32]byte{'3'}, f.store.treeRootNode.bestDescendant.root)
	f.store.nodesLock.Unlock()

	h, err := f.store.head(ctx, [32]byte{'1'})
	require.NoError(t, err)
	require.DeepEqual(t, [32]byte{'3'}, h)
	require.DeepEqual(t, h, f.store.headNode.root)

	require.Equal(t, true, f.IsCanonical(params.BeaconConfig().ZeroHash))
	require.Equal(t, true, f.IsCanonical([32]byte{'1'}))
	require.Equal(t, false, f.IsCanonical([32]byte{'2'}))
	require.Equal(t, true, f.IsCanonical([32]byte{'3'}))
	require.Equal(t, false, f.IsCanonical([32]byte{'4'}))
	require.Equal(t, false, f.IsCanonical([32]byte{'5'}))
	require.Equal(t, false, f.IsCanonical([32]byte{'6'}))
}

func TestForkChoice_AncestorRoot(t *testing.T) {
	f := setup(1, 1)
	ctx := context.Background()
	require.NoError(t, f.ProcessBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.ProcessBlock(ctx, 2, indexToHash(2), indexToHash(1), 1, 1))
	require.NoError(t, f.ProcessBlock(ctx, 5, indexToHash(3), indexToHash(2), 1, 1))
	f.store.treeRootNode = f.store.nodeByRoot[indexToHash(1)]
	f.store.treeRootNode.parent = nil

	r, err := f.AncestorRoot(ctx, indexToHash(3), 6)
	assert.NoError(t, err)
	assert.Equal(t, bytesutil.ToBytes32(r), indexToHash(3))

	_, err = f.AncestorRoot(ctx, indexToHash(3), 0)
	assert.ErrorContains(t, errNilNode.Error(), err)

	root, err := f.AncestorRoot(ctx, indexToHash(3), 5)
	require.NoError(t, err)
	hash3 := indexToHash(3)
	require.DeepEqual(t, hash3[:], root)
	root, err = f.AncestorRoot(ctx, indexToHash(3), 1)
	require.NoError(t, err)
	hash1 := indexToHash(1)
	require.DeepEqual(t, hash1[:], root)
}

func TestForkChoice_AncestorEqualSlot(t *testing.T) {
	f := setup(1, 1)
	ctx := context.Background()
	require.NoError(t, f.ProcessBlock(ctx, 100, [32]byte{'1'}, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.ProcessBlock(ctx, 101, [32]byte{'3'}, [32]byte{'1'}, 1, 1))

	r, err := f.AncestorRoot(ctx, [32]byte{'3'}, 100)
	require.NoError(t, err)
	root := bytesutil.ToBytes32(r)
	require.Equal(t, root, [32]byte{'1'})
}

func TestForkChoice_AncestorLowerSlot(t *testing.T) {
	f := setup(1, 1)
	ctx := context.Background()
	require.NoError(t, f.ProcessBlock(ctx, 100, [32]byte{'1'}, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, f.ProcessBlock(ctx, 200, [32]byte{'3'}, [32]byte{'1'}, 1, 1))

	r, err := f.AncestorRoot(ctx, [32]byte{'3'}, 150)
	require.NoError(t, err)
	root := bytesutil.ToBytes32(r)
	require.Equal(t, root, [32]byte{'1'})
}

func indexToHash(i uint64) [32]byte {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], i)
	return hash.Hash(b[:])
}
