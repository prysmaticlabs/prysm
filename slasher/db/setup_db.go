package db

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path"
	"testing"

	"github.com/pkg/errors"
)

// SetupSlasherDB instantiates and returns a SlasherDB instance.
func SetupSlasherDB() (*Store, error) {
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return nil, err
	}
	p := path.Join(TempDir(), fmt.Sprintf("/%d", randPath))
	if err := os.RemoveAll(p); err != nil {
		return nil, errors.Wrap(err, "Failed to remove directory.")
	}
	db, err := NewDB(p)

	if err != nil {
		return nil, errors.Wrap(err, "Failed to instantiate DB.")
	}
	return db, nil
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

// TeardownSlasherDB cleans up a test BeaconDB instance.
func TeardownSlasherDB(t testing.TB, db *Store) {
	if err := db.Close(); err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}
	if err := os.RemoveAll(db.DatabasePath()); err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}
}
