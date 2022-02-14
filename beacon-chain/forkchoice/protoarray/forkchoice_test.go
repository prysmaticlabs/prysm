package protoarray

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/hash"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestForkChoice_UpdateBalancesPositiveChange(t *testing.T) {
	f := setup(0, 0)
	ctx := context.Background()
	require.NoError(t, f.ProcessBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, 0, 0, false))
	require.NoError(t, f.ProcessBlock(ctx, 2, indexToHash(2), indexToHash(1), 0, 0, false))
	require.NoError(t, f.ProcessBlock(ctx, 3, indexToHash(3), indexToHash(2), 0, 0, false))

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
	require.NoError(t, f.ProcessBlock(ctx, 1, indexToHash(1), params.BeaconConfig().ZeroHash, 0, 0, false))
	require.NoError(t, f.ProcessBlock(ctx, 2, indexToHash(2), indexToHash(1), 0, 0, false))
	require.NoError(t, f.ProcessBlock(ctx, 3, indexToHash(3), indexToHash(2), 0, 0, false))
	s := f.store
	s.nodeByRoot[indexToHash(1)].balance = 100
	s.nodeByRoot[indexToHash(2)].balance = 100
	s.nodeByRoot[indexToHash(3)].balance = 100

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

func indexToHash(i uint64) [32]byte {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], i)
	return hash.Hash(b[:])
}
