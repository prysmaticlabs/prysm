package cache

import (
	"testing"

	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestValidatorPayloadIDsCache_GetAndSaveValidatorPayloadIDs(t *testing.T) {
	cache := NewProposerPayloadIDsCache()
	i, p, ok := cache.GetProposerPayloadIDs(0)
	require.Equal(t, false, ok)
	require.Equal(t, types.ValidatorIndex(0), i)
	require.Equal(t, [pIdLength]byte{}, p)

	slot := types.Slot(1234)
	vid := types.ValidatorIndex(34234324)
	pid := [8]byte{1, 2, 3, 3, 7, 8, 7, 8}
	cache.SetProposerAndPayloadIDs(slot, vid, pid)
	i, p, ok = cache.GetProposerPayloadIDs(slot)
	require.Equal(t, true, ok)
	require.Equal(t, vid, i)
	require.Equal(t, pid, p)

	slot = types.Slot(9456456)
	vid = types.ValidatorIndex(6786745)
	cache.SetProposerAndPayloadIDs(slot, vid, [pIdLength]byte{})
	i, p, ok = cache.GetProposerPayloadIDs(slot)
	require.Equal(t, true, ok)
	require.Equal(t, vid, i)
	require.Equal(t, [pIdLength]byte{}, p)

	// reset cache without pid
	slot = types.Slot(9456456)
	vid = types.ValidatorIndex(11111)
	pid = [8]byte{3, 2, 3, 33, 72, 8, 7, 8}
	cache.SetProposerAndPayloadIDs(slot, vid, pid)
	i, p, ok = cache.GetProposerPayloadIDs(slot)
	require.Equal(t, true, ok)
	require.Equal(t, vid, i)
	require.Equal(t, pid, p)

	// reset cache with existing pid
	slot = types.Slot(9456456)
	vid = types.ValidatorIndex(11111)
	newPid := [8]byte{1, 2, 3, 33, 72, 8, 7, 1}
	cache.SetProposerAndPayloadIDs(slot, vid, newPid)
	i, p, ok = cache.GetProposerPayloadIDs(slot)
	require.Equal(t, true, ok)
	require.Equal(t, vid, i)
	require.Equal(t, newPid, p)

	// remove cache entry
	cache.PrunePayloadIDs(slot + 1)
	i, p, ok = cache.GetProposerPayloadIDs(slot)
	require.Equal(t, false, ok)
	require.Equal(t, types.ValidatorIndex(0), i)
	require.Equal(t, [pIdLength]byte{}, p)
}
