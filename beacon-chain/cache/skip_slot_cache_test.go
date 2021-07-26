package cache_test

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	statepb "github.com/prysmaticlabs/prysm/proto/prysm/v2/state"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestSkipSlotCache_RoundTrip(t *testing.T) {
	ctx := context.Background()
	c := cache.NewSkipSlotCache()

	r := [32]byte{'a'}
	s, err := c.Get(ctx, r)
	require.NoError(t, err)
	assert.Equal(t, state.BeaconState(nil), s, "Empty cache returned an object")

	require.NoError(t, c.MarkInProgress(r))

	s, err = v1.InitializeFromProto(&statepb.BeaconState{
		Slot: 10,
	})
	require.NoError(t, err)

	require.NoError(t, c.Put(ctx, r, s))
	require.NoError(t, c.MarkNotInProgress(r))

	res, err := c.Get(ctx, r)
	require.NoError(t, err)
	assert.DeepEqual(t, res.CloneInnerState(), s.CloneInnerState(), "Expected equal protos to return from cache")
}
