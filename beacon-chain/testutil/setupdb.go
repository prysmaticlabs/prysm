package testutil

import (
	"os"
	"path"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
)

func getPath(t *testing.T) string {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get the current working directory: %v", err)
	}

	return path.Join(wd, "/testdb")
}

// SetupDB instantiates and returns a BeaconDB instance.
func SetupDB(t *testing.T) *db.BeaconDB {
	path := getPath(t)
	if err := os.RemoveAll(path); err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}

	db, err := db.NewDB(path)
	if err != nil {
		t.Fatalf("Failed to instantiate DB: %v", err)
	}

	return db
}

// TeardownDB cleans up a test BeaconDB instance.
func TeardownDB(t *testing.T, db *db.BeaconDB) {
	if err := db.Close(); err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}
	if err := os.RemoveAll(getPath(t)); err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}
}
