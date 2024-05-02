package state_native

import (
	"testing"

	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func Test_FinalizedValidatorIndexCache(t *testing.T) {
	c := newFinalizedValidatorIndexCache()
	b := &BeaconState{validatorIndexCache: c}

	// What happens if you call getValidatorIndex with a public key that is not in the cache and state?
	// The function will return 0 and false.
	i, exists := b.getValidatorIndex([fieldparams.BLSPubkeyLength]byte{0})
	require.Equal(t, primitives.ValidatorIndex(0), i)
	require.Equal(t, false, exists)

	// Validators are added to the state. They are [0, 1, 2]
	b.validators = []*ethpb.Validator{
		{PublicKey: []byte{1}},
		{PublicKey: []byte{2}},
		{PublicKey: []byte{3}},
	}
	// We should be able to retrieve these validators by public key even when they are not in the cache
	i, exists = b.getValidatorIndex([fieldparams.BLSPubkeyLength]byte{1})
	require.Equal(t, primitives.ValidatorIndex(0), i)
	require.Equal(t, true, exists)
	i, exists = b.getValidatorIndex([fieldparams.BLSPubkeyLength]byte{2})
	require.Equal(t, primitives.ValidatorIndex(1), i)
	require.Equal(t, true, exists)
	i, exists = b.getValidatorIndex([fieldparams.BLSPubkeyLength]byte{3})
	require.Equal(t, primitives.ValidatorIndex(2), i)
	require.Equal(t, true, exists)

	// State is finalized. We save [0, 1, 2 ] to the cache.
	b.saveValidatorIndices()
	require.Equal(t, 3, len(b.validatorIndexCache.indexMap))
	i, exists = b.getValidatorIndex([fieldparams.BLSPubkeyLength]byte{1})
	require.Equal(t, primitives.ValidatorIndex(0), i)
	require.Equal(t, true, exists)
	i, exists = b.getValidatorIndex([fieldparams.BLSPubkeyLength]byte{2})
	require.Equal(t, primitives.ValidatorIndex(1), i)
	require.Equal(t, true, exists)
	i, exists = b.getValidatorIndex([fieldparams.BLSPubkeyLength]byte{3})
	require.Equal(t, primitives.ValidatorIndex(2), i)
	require.Equal(t, true, exists)

	// New validators are added to the state. They are [4, 5]
	b.validators = []*ethpb.Validator{
		{PublicKey: []byte{1}},
		{PublicKey: []byte{2}},
		{PublicKey: []byte{3}},
		{PublicKey: []byte{4}},
		{PublicKey: []byte{5}},
	}
	// We should be able to retrieve these validators by public key even when they are not in the cache
	i, exists = b.getValidatorIndex([fieldparams.BLSPubkeyLength]byte{4})
	require.Equal(t, primitives.ValidatorIndex(3), i)
	require.Equal(t, true, exists)
	i, exists = b.getValidatorIndex([fieldparams.BLSPubkeyLength]byte{5})
	require.Equal(t, primitives.ValidatorIndex(4), i)
	require.Equal(t, true, exists)

	// State is finalized. We save [4, 5] to the cache.
	b.saveValidatorIndices()
	require.Equal(t, 5, len(b.validatorIndexCache.indexMap))

	// New validators are added to the state. They are [6]
	b.validators = []*ethpb.Validator{
		{PublicKey: []byte{1}},
		{PublicKey: []byte{2}},
		{PublicKey: []byte{3}},
		{PublicKey: []byte{4}},
		{PublicKey: []byte{5}},
		{PublicKey: []byte{6}},
	}
	// We should be able to retrieve these validators by public key even when they are not in the cache
	i, exists = b.getValidatorIndex([fieldparams.BLSPubkeyLength]byte{6})
	require.Equal(t, primitives.ValidatorIndex(5), i)
	require.Equal(t, true, exists)

	// State is finalized. We save [6] to the cache.
	b.saveValidatorIndices()
	require.Equal(t, 6, len(b.validatorIndexCache.indexMap))

	// Save a few more times.
	b.saveValidatorIndices()
	b.saveValidatorIndices()
	require.Equal(t, 6, len(b.validatorIndexCache.indexMap))

	// Can still retrieve the validators from the cache
	i, exists = b.getValidatorIndex([fieldparams.BLSPubkeyLength]byte{1})
	require.Equal(t, primitives.ValidatorIndex(0), i)
	require.Equal(t, true, exists)
	i, exists = b.getValidatorIndex([fieldparams.BLSPubkeyLength]byte{2})
	require.Equal(t, primitives.ValidatorIndex(1), i)
	require.Equal(t, true, exists)
	i, exists = b.getValidatorIndex([fieldparams.BLSPubkeyLength]byte{3})
	require.Equal(t, primitives.ValidatorIndex(2), i)
	require.Equal(t, true, exists)
}
