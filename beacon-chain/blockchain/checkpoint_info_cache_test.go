package blockchain

import (
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestHotStateCache_RoundTrip(t *testing.T) {
	c := newCheckPointInfoCache()
	cp := &ethpb.Checkpoint{
		Epoch: 1,
		Root:  bytesutil.PadTo([]byte{'a'}, 32),
	}
	info, err := c.get(cp)
	require.NoError(t, err)
	require.Equal(t, (*pb.CheckPtInfo)(nil), info)

	i := &pb.CheckPtInfo{
		Seed:          bytesutil.PadTo([]byte{'c'}, 32),
		GenesisRoot:   bytesutil.PadTo([]byte{'d'}, 32),
		ActiveIndices: []uint64{0, 1, 2, 3},
	}

	require.NoError(t, c.put(cp, i))
	info, err = c.get(cp)
	require.NoError(t, err)
	require.DeepEqual(t, info, i)
}

func TestHotStateCache_CanPrune(t *testing.T) {
	c := newCheckPointInfoCache()
	for i := 0; i < maxInfoSize+1; i++ {
		cp := &ethpb.Checkpoint{Epoch: uint64(i), Root: make([]byte, 32)}
		require.NoError(t, c.put(cp, &pb.CheckPtInfo{}))
	}
	require.Equal(t, len(c.cache.Keys()), maxInfoSize)
}
