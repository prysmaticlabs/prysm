package cache

import (
	"context"
	"testing"

	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestSkipSlotCache_RoundTrip(t *testing.T) {
	ctx := context.Background()
	c := NewSkipSlotCache()

	state, err := c.Get(ctx, 5)
	require.NoError(t, err)
	assert.Equal(t, (*stateTrie.BeaconState)(nil), state, "Empty cache returned an object")

	require.NoError(t, c.MarkInProgress(5))

	state, err = stateTrie.InitializeFromProto(&pb.BeaconState{
		Slot: 10,
	})
	require.NoError(t, err)

	require.NoError(t, c.Put(ctx, 5, state))
	require.NoError(t, c.MarkNotInProgress(5))

	res, err := c.Get(ctx, 5)
	require.NoError(t, err)
	assert.DeepEqual(t, res.CloneInnerState(), state.CloneInnerState(), "Expected equal protos to return from cache")
}
