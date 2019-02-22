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

// SetupDB instantiates and returns a BeaconDB instance.
func SetupDB(t testing.TB) *db.BeaconDB {
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		t.Fatalf("Could not generate random file path: %v", err)
	}
	path := path.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath))
	if err := os.RemoveAll(path); err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}
	db, err := db.NewDB(path)
	if err != nil {
		t.Fatalf("Could not setup DB: %v", err)
	}
	return db
}

// TeardownDB cleans up a BeaconDB instance.
func TeardownDB(t testing.TB, db *db.BeaconDB) {
	if err := db.Close(); err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}
	if err := os.RemoveAll(db.DatabasePath); err != nil {
		t.Fatalf("Could not remove tmp db dir: %v", err)
	}
}
