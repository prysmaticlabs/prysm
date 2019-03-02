package db

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
)

// setupDB instantiates and returns a BeaconDB instance.
func setupDB(tb *testing.TB) *BeaconDB {
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		tb.Fatalf("Could not generate random file path: %v", err)
	}
	path := path.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath))
	if err := os.RemoveAll(path); err != nil {
		tb.Fatalf("Failed to remove directory: %v", err)
	}
	db, err := NewDB(path)
	if err != nil {
		tb.Fatalf("Failed to instantiate DB: %v", err)
	}
	return db
}

// teardownDB cleans up a test BeaconDB instance.
func teardownDB(tb *testing.TB, db *BeaconDB) {
	if err := db.Close(); err != nil {
		tb.Fatalf("Failed to close database: %v", err)
	}
	if err := os.RemoveAll(db.DatabasePath); err != nil {
		tb.Fatalf("Failed to remove directory: %v", err)
	}
}
