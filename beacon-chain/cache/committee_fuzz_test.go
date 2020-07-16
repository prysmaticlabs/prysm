package cache

import (
	"testing"

	fuzz "github.com/google/gofuzz"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestCommitteeKeyFuzz_OK(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	c := &Committees{}

	for i := 0; i < 100000; i++ {
		fuzzer.Fuzz(c)
		k, err := committeeKeyFn(c)
		require.NoError(t, err)
		assert.Equal(t, key(c.Seed), k)
	}
}

func TestCommitteeCache_FuzzCommitteesByEpoch(t *testing.T) {
	cache := NewCommitteesCache()
	fuzzer := fuzz.NewWithSeed(0)
	c := &Committees{}

	for i := 0; i < 100000; i++ {
		fuzzer.Fuzz(c)
		require.NoError(t, cache.AddCommitteeShuffledList(c))
		_, err := cache.Committee(0, c.Seed, 0)
		require.NoError(t, err)
	}

	assert.Equal(t, maxCommitteesCacheSize, uint64(len(cache.CommitteeCache.ListKeys())), "Incorrect key size")
}

func TestCommitteeCache_FuzzActiveIndices(t *testing.T) {
	cache := NewCommitteesCache()
	fuzzer := fuzz.NewWithSeed(0)
	c := &Committees{}

	for i := 0; i < 100000; i++ {
		fuzzer.Fuzz(c)
		require.NoError(t, cache.AddCommitteeShuffledList(c))

		indices, err := cache.ActiveIndices(c.Seed)
		require.NoError(t, err)
		assert.DeepEqual(t, c.SortedIndices, indices)
	}

	assert.Equal(t, maxCommitteesCacheSize, uint64(len(cache.CommitteeCache.ListKeys())), "Incorrect key size")
}
