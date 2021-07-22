package cache

import (
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	ethereum_slashing "github.com/prysmaticlabs/prysm/proto/prysm/v2"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestStoringAndFetching(t *testing.T) {
	cache, err := NewHighestAttestationCache(10, nil)
	require.NoError(t, err)

	// Cache a test attestation.
	cache.Set(1, &ethereum_slashing.HighestAttestation{
		ValidatorId:        1,
		HighestSourceEpoch: 2,
		HighestTargetEpoch: 3,
	})

	// Require it to exist.
	require.Equal(t, true, cache.Has(1))

	// fetch
	res, b := cache.Get(1)
	require.Equal(t, true, b)
	require.Equal(t, uint64(1), res[1].ValidatorId)
	require.Equal(t, types.Epoch(2), res[1].HighestSourceEpoch)
	require.Equal(t, types.Epoch(3), res[1].HighestTargetEpoch)

	// Delete it.
	require.Equal(t, true, cache.Delete(1))
	// Confirm deletion.
	res2, b2 := cache.Get(1)
	require.Equal(t, false, b2)
	require.Equal(t, true, res2 == nil)
}

func TestPurge(t *testing.T) {
	wasEvicted := false
	onEvicted := func(key interface{}, value interface{}) {
		wasEvicted = true
	}
	cache, err := NewHighestAttestationCache(10, onEvicted)
	require.NoError(t, err)

	// Cache several test attestation.
	cache.Set(1, &ethereum_slashing.HighestAttestation{
		ValidatorId:        1,
		HighestSourceEpoch: 2,
		HighestTargetEpoch: 3,
	})
	cache.Set(2, &ethereum_slashing.HighestAttestation{
		ValidatorId:        4,
		HighestSourceEpoch: 5,
		HighestTargetEpoch: 6,
	})
	cache.Set(3, &ethereum_slashing.HighestAttestation{
		ValidatorId:        7,
		HighestSourceEpoch: 8,
		HighestTargetEpoch: 9,
	})

	cache.Purge()

	// Require all attestations to be deleted
	require.Equal(t, false, cache.Has(1))
	require.Equal(t, false, cache.Has(2))
	require.Equal(t, false, cache.Has(3))

	// Require the eviction function to be called.
	require.Equal(t, true, wasEvicted)
}

func TestClear(t *testing.T) {
	wasEvicted := false
	onEvicted := func(key interface{}, value interface{}) {
		wasEvicted = true
	}
	cache, err := NewHighestAttestationCache(10, onEvicted)
	require.NoError(t, err)

	// Cache several test attestation.
	cache.Set(1, &ethereum_slashing.HighestAttestation{
		ValidatorId:        1,
		HighestSourceEpoch: 2,
		HighestTargetEpoch: 3,
	})
	cache.Set(2, &ethereum_slashing.HighestAttestation{
		ValidatorId:        4,
		HighestSourceEpoch: 5,
		HighestTargetEpoch: 6,
	})
	cache.Set(3, &ethereum_slashing.HighestAttestation{
		ValidatorId:        7,
		HighestSourceEpoch: 8,
		HighestTargetEpoch: 9,
	})

	cache.Clear()

	// Require all attestations to be deleted
	require.Equal(t, false, cache.Has(1))
	require.Equal(t, false, cache.Has(2))
	require.Equal(t, false, cache.Has(3))

	// Require the eviction function to be called.
	require.Equal(t, true, wasEvicted)
}
