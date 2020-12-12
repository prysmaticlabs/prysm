package kv

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

// setupDB instantiates and returns a Store instance.
func setupDB(t testing.TB) *Store {
	db, err := NewKVStore(t.TempDir(), newStateSummaryCache())
	require.NoError(t, err, "Failed to instantiate DB")
	t.Cleanup(func() {
		require.NoError(t, db.Close(), "Failed to close database")
	})
	return db
}
