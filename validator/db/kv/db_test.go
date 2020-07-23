package kv

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/rand"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

// setupDB instantiates and returns a DB instance for the validator client.
func setupDB(t testing.TB, pubkeys [][48]byte) *Store {
	randPath := rand.NewDeterministicGenerator().Int()
	p := filepath.Join(tempdir(), fmt.Sprintf("/%d", randPath))
	require.NoError(t, os.RemoveAll(p), "Failed to remove directory")
	db, err := NewKVStore(p, pubkeys)
	require.NoError(t, err, "Failed to instantiate DB")

	t.Cleanup(func() {
		require.NoError(t, db.Close(), "Failed to close database")
		require.NoError(t, db.ClearDB(), "Failed to clear database")
	})
	return db
}

// tempdir returns a directory path for temporary test storage.
func tempdir() string {
	d := os.Getenv("TEST_TMPDIR")

	// If the test is not run via bazel, the environment var won't be set.
	if d == "" {
		return os.TempDir()
	}
	return d
}
