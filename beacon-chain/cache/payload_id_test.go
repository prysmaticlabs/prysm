package cache

import (
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestValidatorPayloadIDsCache_GetAndSaveValidatorPayloadIDs(t *testing.T) {
	cache := NewValidatorPayloadIDsCache()
	i, p, ok := cache.GetValidatorPayloadIDs(0)
	require.Equal(t, false, ok)
	require.Equal(t, types.ValidatorIndex(0), i)
	require.Equal(t, uint64(0), p)

	slot := types.Slot(1234)
	vid := types.ValidatorIndex(34234324)
	pid := uint64(234234)
	cache.SetValidatorAndPayloadIDs(slot, vid, pid)
	i, p, ok = cache.GetValidatorPayloadIDs(slot)
	require.Equal(t, true, ok)
	require.Equal(t, vid, i)
	require.Equal(t, pid, p)

	slot = types.Slot(9456456)
	vid = types.ValidatorIndex(6786745)
	pid = uint64(87687)
	cache.SetValidatorAndPayloadIDs(slot, vid, pid)
	i, p, ok = cache.GetValidatorPayloadIDs(slot)
	require.Equal(t, true, ok)
	require.Equal(t, vid, i)
	require.Equal(t, pid, p)
}
