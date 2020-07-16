package cache

import (
	"math"
	"sort"
	"strconv"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestCommitteeKeyFn_OK(t *testing.T) {
	item := &Committees{
		CommitteeCount:  1,
		Seed:            [32]byte{'A'},
		ShuffledIndices: []uint64{1, 2, 3, 4, 5},
	}

	k, err := committeeKeyFn(item)
	require.NoError(t, err)
	assert.Equal(t, key(item.Seed), k)
}

func TestCommitteeKeyFn_InvalidObj(t *testing.T) {
	_, err := committeeKeyFn("bad")
	assert.Equal(t, ErrNotCommittee, err)
}

func TestCommitteeCache_CommitteesByEpoch(t *testing.T) {
	cache := NewCommitteesCache()

	item := &Committees{
		ShuffledIndices: []uint64{1, 2, 3, 4, 5, 6},
		Seed:            [32]byte{'A'},
		CommitteeCount:  3,
	}

	slot := params.BeaconConfig().SlotsPerEpoch
	committeeIndex := uint64(1)
	indices, err := cache.Committee(slot, item.Seed, committeeIndex)
	require.NoError(t, err)
	if indices != nil {
		t.Error("Expected committee not to exist in empty cache")
	}
	require.NoError(t, cache.AddCommitteeShuffledList(item))

	wantedIndex := uint64(0)
	indices, err = cache.Committee(slot, item.Seed, wantedIndex)
	require.NoError(t, err)

	start, end := startEndIndices(item, wantedIndex)
	assert.DeepEqual(t, item.ShuffledIndices[start:end], indices)
}

func TestCommitteeCache_ActiveIndices(t *testing.T) {
	cache := NewCommitteesCache()

	item := &Committees{Seed: [32]byte{'A'}, SortedIndices: []uint64{1, 2, 3, 4, 5, 6}}
	indices, err := cache.ActiveIndices(item.Seed)
	require.NoError(t, err)
	if indices != nil {
		t.Error("Expected committee not to exist in empty cache")
	}

	require.NoError(t, cache.AddCommitteeShuffledList(item))

	indices, err = cache.ActiveIndices(item.Seed)
	require.NoError(t, err)
	assert.DeepEqual(t, item.SortedIndices, indices)
}

func TestCommitteeCache_ActiveCount(t *testing.T) {
	cache := NewCommitteesCache()

	item := &Committees{Seed: [32]byte{'A'}, SortedIndices: []uint64{1, 2, 3, 4, 5, 6}}
	count, err := cache.ActiveIndicesCount(item.Seed)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "Expected active count not to exist in empty cache")

	require.NoError(t, cache.AddCommitteeShuffledList(item))

	count, err = cache.ActiveIndicesCount(item.Seed)
	require.NoError(t, err)
	assert.Equal(t, len(item.SortedIndices), count)
}

func TestCommitteeCache_AddProposerIndicesList(t *testing.T) {
	cache := NewCommitteesCache()

	seed := [32]byte{'A'}
	indices := []uint64{1, 2, 3, 4, 5}
	indices, err := cache.ProposerIndices(seed)
	require.NoError(t, err)
	if indices != nil {
		t.Error("Expected committee count not to exist in empty cache")
	}
	require.NoError(t, cache.AddProposerIndicesList(seed, indices))

	received, err := cache.ProposerIndices(seed)
	require.NoError(t, err)
	assert.DeepEqual(t, received, indices)

	item := &Committees{Seed: [32]byte{'B'}, SortedIndices: []uint64{1, 2, 3, 4, 5, 6}}
	require.NoError(t, cache.AddCommitteeShuffledList(item))

	indices, err = cache.ProposerIndices(item.Seed)
	require.NoError(t, err)
	if indices != nil {
		t.Error("Expected committee count not to exist in empty cache")
	}
	require.NoError(t, cache.AddProposerIndicesList(item.Seed, indices))

	received, err = cache.ProposerIndices(item.Seed)
	require.NoError(t, err)
	assert.DeepEqual(t, received, indices)
}

func TestCommitteeCache_CanRotate(t *testing.T) {
	cache := NewCommitteesCache()

	// Should rotate out all the epochs except 190 through 199.
	for i := 100; i < 200; i++ {
		s := []byte(strconv.Itoa(i))
		item := &Committees{Seed: bytesutil.ToBytes32(s)}
		require.NoError(t, cache.AddCommitteeShuffledList(item))
	}

	k := cache.CommitteeCache.ListKeys()
	assert.Equal(t, maxCommitteesCacheSize, uint64(len(k)))

	sort.Slice(k, func(i, j int) bool {
		return k[i] < k[j]
	})
	s := bytesutil.ToBytes32([]byte(strconv.Itoa(190)))
	assert.Equal(t, key(s), k[0], "incorrect key received for slot 190")

	s = bytesutil.ToBytes32([]byte(strconv.Itoa(199)))
	assert.Equal(t, key(s), k[len(k)-1], "incorrect key received for slot 199")
}

func TestCommitteeCacheOutOfRange(t *testing.T) {
	cache := NewCommitteesCache()
	seed := bytesutil.ToBytes32([]byte("foo"))
	err := cache.CommitteeCache.Add(&Committees{
		CommitteeCount:  1,
		Seed:            seed,
		ShuffledIndices: []uint64{0},
		SortedIndices:   []uint64{},
		ProposerIndices: []uint64{},
	})
	require.NoError(t, err)

	_, err = cache.Committee(0, seed, math.MaxUint64) // Overflow!
	require.NotNil(t, err, "Did not fail as expected")
}
