package kv

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/rand"
)

// setupDB instantiates and returns a DB instance for the validator client.
func setupDB(t testing.TB, pubkeys [][48]byte) *Store {
	randPath := rand.NewDeterministicGenerator().Int()
	p := filepath.Join(tempdir(), fmt.Sprintf("/%d", randPath))
	if err := os.RemoveAll(p); err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}
	db, err := NewKVStore(p, pubkeys)
	if err != nil {
		t.Fatalf("Failed to instantiate DB: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Failed to close database: %v", err)
		}
		if err := db.ClearDB(); err != nil {
			t.Fatalf("Failed to clear database: %v", err)
		}
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

func TestStore_DatabasePath(t *testing.T) {
	db := setupDB(t, [][48]byte{})
	dbPath := db.DatabasePath()
	if !strings.Contains(dbPath, databaseFileName) {
		t.Fatal("Expected filepath to lead to database file")
	}
}
