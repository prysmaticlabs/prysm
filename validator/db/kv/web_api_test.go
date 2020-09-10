package kv

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestStore_HashedPasswordForAPI_SaveAndRetrieve(t *testing.T) {
	db := setupDB(t, [][48]byte{})
	hashedPassword := []byte("2093402934902839489238492")
	ctx := context.Background()
	// Assert we have no hashed password stored.
	res, err := db.HashedPasswordForAPI(ctx)
	require.NoError(t, err)
	assert.DeepEqual(t, 0, len(res))

	// Save the hashed password and attempt to refetch it.
	require.NoError(t, db.SaveHashedPasswordForAPI(ctx, hashedPassword))
	res, err = db.HashedPasswordForAPI(ctx)
	require.NoError(t, err)
	// Assert the retrieves value equals what we saved.
	assert.DeepEqual(t, hashedPassword, res)
}
