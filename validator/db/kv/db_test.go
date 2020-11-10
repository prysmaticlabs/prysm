package kv

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

// setupDB instantiates and returns a DB instance for the validator client.
func setupDB(t testing.TB, pubkeys [][48]byte) *Store {
	p := t.TempDir()
	db, err := NewKVStore(p, pubkeys)
	require.NoError(t, err, "Failed to instantiate DB")
	err = db.OldUpdatePublicKeysBuckets(pubkeys)
	require.NoError(t, err, "Failed to create old buckets for public keys")
	t.Cleanup(func() {
		require.NoError(t, db.Close(), "Failed to close database")
		require.NoError(t, db.ClearDB(), "Failed to clear database")
	})
	return db
}
