package internal

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

// SetupDBDeprecated instantiates and returns a BeaconDB instance.
// This is deprecated and used to set up the pre refactored db for testing.
// DEPRECATED: Use beacon-chain/db/testing.SetupDB
func SetupDBDeprecated(t testing.TB) *db.BeaconDB {
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		t.Fatalf("Could not generate random file path: %v", err)
	}
	path := path.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath))
	if err := os.RemoveAll(path); err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}
	db, err := db.NewDBDeprecated(path)
	if err != nil {
		t.Fatalf("Could not setup DB: %v", err)
	}
	return db
}

// TeardownDBDeprecated cleans up a BeaconDB instance.
// This is deprecated and used to tear up the pre refactored db for testing.
// DEPRECATED: Use beacon-chain/db/testing.TeardownDB
func TeardownDBDeprecated(t testing.TB, db *db.BeaconDB) {
	if err := db.Close(); err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}
	if err := os.RemoveAll(db.DatabasePath()); err != nil {
		t.Fatalf("Could not remove tmp db dir: %v", err)
	}
}
