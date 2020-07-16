package cache

import (
	"testing"

	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestHotStateCache_RoundTrip(t *testing.T) {
	c := NewHotStateCache()
	root := [32]byte{'A'}
	state := c.Get(root)
	assert.Equal(t, (*stateTrie.BeaconState)(nil), state)
	assert.Equal(t, false, c.Has(root), "Empty cache has an object")

	state, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Slot: 10,
	})
	require.NoError(t, err)

	c.Put(root, state)
	assert.Equal(t, true, c.Has(root), "Empty cache does not have an object")

	res := c.Get(root)
	assert.NotNil(t, state)
	assert.DeepEqual(t, res.CloneInnerState(), state.CloneInnerState(), "Expected equal protos to return from cache")

	c.Delete(root)
	assert.Equal(t, false, c.Has(root), "Cache not supposed to have the object")
}
