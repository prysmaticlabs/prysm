package cache

import (
	"encoding/binary"
	"math"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	state "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	"github.com/prysmaticlabs/prysm/config/features"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestBalanceCache_AddGetBalance(t *testing.T) {
	resetCfg := features.InitWithReset(&features.Flags{
		EnableActiveBalanceCache: true,
	})
	defer resetCfg()

	blockRoots := make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot)
	for i := 0; i < len(blockRoots); i++ {
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(i))
		blockRoots[i] = b
	}
	raw := &ethpb.BeaconState{
		BlockRoots: blockRoots,
	}
	st, err := state.InitializeFromProto(raw)
	require.NoError(t, err)

	cache := NewEffectiveBalanceCache()
	_, err = cache.Get(st)
	require.ErrorContains(t, ErrNotFound.Error(), err)

	b := uint64(100)
	require.NoError(t, cache.AddTotalEffectiveBalance(st, b))
	cachedB, err := cache.Get(st)
	require.NoError(t, err)
	require.Equal(t, b, cachedB)

	require.NoError(t, st.SetSlot(1000))
	_, err = cache.Get(st)
	require.ErrorContains(t, ErrNotFound.Error(), err)

	b = uint64(200)
	require.NoError(t, cache.AddTotalEffectiveBalance(st, b))
	cachedB, err = cache.Get(st)
	require.NoError(t, err)
	require.Equal(t, b, cachedB)

	require.NoError(t, st.SetSlot(1000+params.BeaconConfig().SlotsPerHistoricalRoot))
	_, err = cache.Get(st)
	require.ErrorContains(t, ErrNotFound.Error(), err)

	b = uint64(300)
	require.NoError(t, cache.AddTotalEffectiveBalance(st, b))
	cachedB, err = cache.Get(st)
	require.NoError(t, err)
	require.Equal(t, b, cachedB)
}

func TestBalanceCache_BalanceKey(t *testing.T) {
	blockRoots := make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot)
	for i := 0; i < len(blockRoots); i++ {
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(i))
		blockRoots[i] = b
	}
	raw := &ethpb.BeaconState{
		BlockRoots: blockRoots,
	}
	st, err := state.InitializeFromProto(raw)
	require.NoError(t, err)
	require.NoError(t, st.SetSlot(types.Slot(math.MaxUint64)))

	_, err = balanceCacheKey(st)
	require.NoError(t, err)
}
