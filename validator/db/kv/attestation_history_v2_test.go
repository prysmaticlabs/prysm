package kv

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestLowestSignedSourceEpoch_SaveRetrieve(t *testing.T) {
	ctx := context.Background()
	validatorDB, err := NewKVStore(ctx, t.TempDir(), nil)
	require.NoError(t, err, "Failed to instantiate DB")
	t.Cleanup(func() {
		require.NoError(t, validatorDB.Close(), "Failed to close database")
		require.NoError(t, validatorDB.ClearDB(), "Failed to clear database")
	})
	p0 := [48]byte{0}
	p1 := [48]byte{1}
	// Can save.
	require.NoError(t, validatorDB.SaveLowestSignedSourceEpoch(ctx, p0, 100))
	require.NoError(t, validatorDB.SaveLowestSignedSourceEpoch(ctx, p1, 200))
	got, err := validatorDB.LowestSignedSourceEpoch(ctx, p0)
	require.NoError(t, err)
	require.Equal(t, uint64(100), got)
	got, err = validatorDB.LowestSignedSourceEpoch(ctx, p1)
	require.NoError(t, err)
	require.Equal(t, uint64(200), got)

	// Can replace.
	require.NoError(t, validatorDB.SaveLowestSignedSourceEpoch(ctx, p0, 99))
	require.NoError(t, validatorDB.SaveLowestSignedSourceEpoch(ctx, p1, 199))
	got, err = validatorDB.LowestSignedSourceEpoch(ctx, p0)
	require.NoError(t, err)
	require.Equal(t, uint64(99), got)
	got, err = validatorDB.LowestSignedSourceEpoch(ctx, p1)
	require.NoError(t, err)
	require.Equal(t, uint64(199), got)

	// Can not replace.
	require.NoError(t, validatorDB.SaveLowestSignedSourceEpoch(ctx, p0, 100))
	require.NoError(t, validatorDB.SaveLowestSignedSourceEpoch(ctx, p1, 200))
	got, err = validatorDB.LowestSignedSourceEpoch(ctx, p0)
	require.NoError(t, err)
	require.Equal(t, uint64(99), got)
	got, err = validatorDB.LowestSignedSourceEpoch(ctx, p1)
	require.NoError(t, err)
	require.Equal(t, uint64(199), got)
}

func TestLowestSignedTargetEpoch_SaveRetrieveReplace(t *testing.T) {
	ctx := context.Background()
	validatorDB, err := NewKVStore(ctx, t.TempDir(), nil)
	require.NoError(t, err, "Failed to instantiate DB")
	t.Cleanup(func() {
		require.NoError(t, validatorDB.Close(), "Failed to close database")
		require.NoError(t, validatorDB.ClearDB(), "Failed to clear database")
	})
	p0 := [48]byte{0}
	p1 := [48]byte{1}
	// Can save.
	require.NoError(t, validatorDB.SaveLowestSignedTargetEpoch(ctx, p0, 100))
	require.NoError(t, validatorDB.SaveLowestSignedTargetEpoch(ctx, p1, 200))
	got, err := validatorDB.LowestSignedTargetEpoch(ctx, p0)
	require.NoError(t, err)
	require.Equal(t, uint64(100), got)
	got, err = validatorDB.LowestSignedTargetEpoch(ctx, p1)
	require.NoError(t, err)
	require.Equal(t, uint64(200), got)

	// Can replace.
	require.NoError(t, validatorDB.SaveLowestSignedTargetEpoch(ctx, p0, 99))
	require.NoError(t, validatorDB.SaveLowestSignedTargetEpoch(ctx, p1, 199))
	got, err = validatorDB.LowestSignedTargetEpoch(ctx, p0)
	require.NoError(t, err)
	require.Equal(t, uint64(99), got)
	got, err = validatorDB.LowestSignedTargetEpoch(ctx, p1)
	require.NoError(t, err)
	require.Equal(t, uint64(199), got)

	// Can not replace.
	require.NoError(t, validatorDB.SaveLowestSignedTargetEpoch(ctx, p0, 100))
	require.NoError(t, validatorDB.SaveLowestSignedTargetEpoch(ctx, p1, 200))
	got, err = validatorDB.LowestSignedTargetEpoch(ctx, p0)
	require.NoError(t, err)
	require.Equal(t, uint64(99), got)
	got, err = validatorDB.LowestSignedTargetEpoch(ctx, p1)
	require.NoError(t, err)
	require.Equal(t, uint64(199), got)
}
