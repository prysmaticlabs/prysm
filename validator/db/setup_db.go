package db

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path"
	"testing"
)

// SetupDB instantiates and returns a DB instance for the validator client.
func SetupDB(t testing.TB) *Store {
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		t.Fatalf("Could not generate random file path: %v", err)
	}
	p := path.Join(TempDir(), fmt.Sprintf("/%d", randPath))
	if err := os.RemoveAll(p); err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}
	db, err := NewDB(p)

	if err != nil {
		t.Fatalf("Failed to instantiate DB: %v", err)
	}
	return db
}

// TempDir returns a directory path for temporary test storage.
func TempDir() string {
	d := os.Getenv("TEST_TMPDIR")

	// If the test is not run via bazel, the environment var won't be set.
	if d == "" {
		return os.TempDir()
	}
	return d
}

// TeardownDB cleans up a test DB instance.
func TeardownDB(t testing.TB, db *Store) {
	if err := db.Close(); err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}
	if err := os.RemoveAll(db.DatabasePath()); err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}
}
