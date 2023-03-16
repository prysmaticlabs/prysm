package cache

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestValidatorPayloadIDsCache_GetAndSaveValidatorPayloadIDs(t *testing.T) {
	cache := NewProposerPayloadIDsCache()
	var r [32]byte
	i, p, ok := cache.GetProposerPayloadIDs(0, r)
	require.Equal(t, false, ok)
	require.Equal(t, primitives.ValidatorIndex(0), i)
	require.Equal(t, [pIdLength]byte{}, p)

	slot := primitives.Slot(1234)
	vid := primitives.ValidatorIndex(34234324)
	pid := [8]byte{1, 2, 3, 3, 7, 8, 7, 8}
	r = [32]byte{1, 2, 3}
	cache.SetProposerAndPayloadIDs(slot, vid, pid, r)
	i, p, ok = cache.GetProposerPayloadIDs(slot, r)
	require.Equal(t, true, ok)
	require.Equal(t, vid, i)
	require.Equal(t, pid, p)

	slot = primitives.Slot(9456456)
	vid = primitives.ValidatorIndex(6786745)
	r = [32]byte{4, 5, 6}
	cache.SetProposerAndPayloadIDs(slot, vid, [pIdLength]byte{}, r)
	i, p, ok = cache.GetProposerPayloadIDs(slot, r)
	require.Equal(t, true, ok)
	require.Equal(t, vid, i)
	require.Equal(t, [pIdLength]byte{}, p)

	// reset cache without pid
	slot = primitives.Slot(9456456)
	vid = primitives.ValidatorIndex(11111)
	r = [32]byte{7, 8, 9}
	pid = [8]byte{3, 2, 3, 33, 72, 8, 7, 8}
	cache.SetProposerAndPayloadIDs(slot, vid, pid, r)
	i, p, ok = cache.GetProposerPayloadIDs(slot, r)
	require.Equal(t, true, ok)
	require.Equal(t, vid, i)
	require.Equal(t, pid, p)

	// Forked chain
	r = [32]byte{1, 2, 3}
	i, p, ok = cache.GetProposerPayloadIDs(slot, r)
	require.Equal(t, false, ok)
	require.Equal(t, primitives.ValidatorIndex(0), i)
	require.Equal(t, [pIdLength]byte{}, p)

	// existing pid - no change in cache
	slot = primitives.Slot(9456456)
	vid = primitives.ValidatorIndex(11111)
	r = [32]byte{7, 8, 9}
	newPid := [8]byte{1, 2, 3, 33, 72, 8, 7, 1}
	cache.SetProposerAndPayloadIDs(slot, vid, newPid, r)
	i, p, ok = cache.GetProposerPayloadIDs(slot, r)
	require.Equal(t, true, ok)
	require.Equal(t, vid, i)
	require.Equal(t, pid, p)

	// remove cache entry
	cache.PrunePayloadIDs(slot + 1)
	i, p, ok = cache.GetProposerPayloadIDs(slot, r)
	require.Equal(t, false, ok)
	require.Equal(t, primitives.ValidatorIndex(0), i)
	require.Equal(t, [pIdLength]byte{}, p)
}
