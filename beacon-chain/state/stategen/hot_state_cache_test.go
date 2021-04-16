package stategen

import (
	"testing"

	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateV0"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestHotStateCache_RoundTrip(t *testing.T) {
	c := newHotStateCache()
	root := [32]byte{'A'}
	state := c.get(root)
	assert.Equal(t, iface.BeaconState(nil), state)
	assert.Equal(t, false, c.has(root), "Empty cache has an object")

	state, err := stateV0.InitializeFromProto(&pb.BeaconState{
		Slot: 10,
	})
	require.NoError(t, err)

	c.put(root, state)
	assert.Equal(t, true, c.has(root), "Empty cache does not have an object")

	res := c.get(root)
	assert.NotNil(t, state)
	assert.DeepEqual(t, res.CloneInnerState(), state.CloneInnerState(), "Expected equal protos to return from cache")

	c.delete(root)
	assert.Equal(t, false, c.has(root), "Cache not supposed to have the object")
}
