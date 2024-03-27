//go:build !fuzz

package cache

import (
	"context"
	"errors"
	"testing"

	fuzz "github.com/google/gofuzz"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestCommitteeKey_Nil(t *testing.T) {
	var c *Committees

	_, err := committeeCachesKeyFn(c)
	require.ErrorIs(t, err, ErrNilValueProvided)
}

func TestCommitteeKeyFuzz_OK(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	c := &Committees{}

	for i := 0; i < 100000; i++ {
		fuzzer.Fuzz(c)
		k, err := committeeCachesKeyFn(c)
		require.NoError(t, err)
		assert.Equal(t, committeeCachesKey(c.Seed), k)
	}
}

func TestCommitteeCache_FuzzCommitteesByEpoch(t *testing.T) {
	cache, err := NewCommitteesCache()
	require.NoError(t, err)

	fuzzer := fuzz.NewWithSeed(0)
	c := &Committees{}

	for i := 0; i < 100000; i++ {
		fuzzer.Fuzz(c)
		err := cache.AddCommitteeShuffledList(context.Background(), c)
		if err != nil {
			require.Equal(t, false, !errors.Is(err, ErrNilValueProvided))
		}
		_, err = cache.Committee(context.Background(), 0, c.Seed, 0)
		require.NoError(t, err)
	}

	assert.Equal(t, maxCommitteesCacheSize, len(keys[string, Committees](cache)), "Incorrect key size")
}

func TestCommitteeCache_FuzzActiveIndices(t *testing.T) {
	cache, err := NewCommitteesCache()
	require.NoError(t, err)
	fuzzer := fuzz.NewWithSeed(0)
	c := &Committees{}

	for i := 0; i < 100000; i++ {
		fuzzer.Fuzz(c)
		if err = cache.AddCommitteeShuffledList(context.Background(), c); err != nil {
			continue
		}

		indices, err := cache.ActiveIndices(context.Background(), c.Seed)
		require.NoError(t, err)
		assert.DeepEqual(t, c.SortedIndices, indices)
	}

	assert.Equal(t, maxCommitteesCacheSize, len(keys[string, Committees](cache)), "Incorrect key size")
}
