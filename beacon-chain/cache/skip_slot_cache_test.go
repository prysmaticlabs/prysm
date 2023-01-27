package cache_test

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/cache"
	state "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/state"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/state/types"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestSkipSlotCache_RoundTrip(t *testing.T) {
	ctx := context.Background()
	c := cache.NewSkipSlotCache()

	r := [32]byte{'a'}
	s, err := c.Get(ctx, r)
	require.NoError(t, err)
	assert.Equal(t, types.BeaconState(nil), s, "Empty cache returned an object")

	require.NoError(t, c.MarkInProgress(r))

	s, err = state.InitializeFromProtoPhase0(&ethpb.BeaconState{
		Slot: 10,
	})
	require.NoError(t, err)

	c.Put(ctx, r, s)
	c.MarkNotInProgress(r)

	res, err := c.Get(ctx, r)
	require.NoError(t, err)
	assert.DeepEqual(t, res.ToProto(), s.ToProto(), "Expected equal protos to return from cache")
}
