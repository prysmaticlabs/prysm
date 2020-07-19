package stategen

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestEpochBoundaryStateCache_BadSlotKey(t *testing.T) {
	_, err := slotKeyFn("sushi")
	assert.ErrorContains(t, errNotSlotRootInfo.Error(), err, "Did not get wanted error")
}

func TestEpochBoundaryStateCache_BadRootKey(t *testing.T) {
	_, err := rootKeyFn("noodle")
	assert.ErrorContains(t, errNotRootStateInfo.Error(), err, "Did not get wanted error")
}

func TestEpochBoundaryStateCache_CanSave(t *testing.T) {
	e := newBoundaryStateCache()
	s := testutil.NewBeaconState()
	require.NoError(t, s.SetSlot(1))
	r := [32]byte{'a'}
	require.NoError(t, e.put(r, s))

	got, exists, err := e.getByRoot([32]byte{'b'})
	require.NoError(t, err)
	assert.Equal(t, false, exists, "Should not exist")
	assert.Equal(t, (*rootStateInfo)(nil), got, "Should not exist")

	got, exists, err = e.getByRoot([32]byte{'a'})
	require.NoError(t, err)
	assert.Equal(t, true, exists, "Should exist")
	assert.DeepEqual(t, s.InnerStateUnsafe(), got.state.InnerStateUnsafe(), "Should have the same state")

	got, exists, err = e.getBySlot(2)
	require.NoError(t, err)
	assert.Equal(t, false, exists, "Should not exist")
	assert.Equal(t, (*rootStateInfo)(nil), got, "Should not exist")

	got, exists, err = e.getBySlot(1)
	require.NoError(t, err)
	assert.Equal(t, true, exists, "Should exist")
	assert.DeepEqual(t, s.InnerStateUnsafe(), got.state.InnerStateUnsafe(), "Should have the same state")
}

func TestEpochBoundaryStateCache_CanTrim(t *testing.T) {
	e := newBoundaryStateCache()
	offSet := uint64(10)
	for i := uint64(0); i < maxCacheSize+offSet; i++ {
		s := testutil.NewBeaconState()
		require.NoError(t, s.SetSlot(i))
		r := [32]byte{byte(i)}
		require.NoError(t, e.put(r, s))
	}

	assert.Equal(t, int(maxCacheSize), len(e.rootStateCache.ListKeys()), "Did not trim to the correct amount")
	assert.Equal(t, int(maxCacheSize), len(e.slotRootCache.ListKeys()), "Did not trim to the correct amount")
	for _, l := range e.rootStateCache.List() {
		i, ok := l.(*rootStateInfo)
		require.Equal(t, true, ok, "Bad type assertion")
		if i.state.Slot() < offSet {
			t.Error("Did not trim the correct state")
		}
	}
}
