package kv

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path"
	"testing"
)

// setupDB instantiates and returns a Store instance.
func setupDB(t testing.TB) *Store {
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		t.Fatalf("Could not generate random file path: %v", err)
	}
	path := path.Join(tempDir(), fmt.Sprintf("/%d", randPath))
	if err := os.RemoveAll(path); err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}
	db, err := NewKVStore(path)
	if err != nil {
		t.Fatalf("Failed to instantiate DB: %v", err)
	}
	return db
}

// teardownDB cleans up a test Store instance.
func teardownDB(t testing.TB, db *Store) {
	if err := db.Close(); err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}
	if err := os.RemoveAll(db.DatabasePath); err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}
}

func tempDir() string {
	d := os.Getenv("TEST_TMPDIR")

	// If the test is not run via bazel, the environment var won't be set.
	if d == "" {
		return os.TempDir()
	}
	return d
}
