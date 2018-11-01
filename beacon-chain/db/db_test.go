package db

import (
	"os"
	"path"
	"testing"
)

func getPath(t *testing.T) string {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get the current working directory: %v", err)
	}

	return path.Join(wd, "/testdb")
}

// setupDB instantiates and returns a BeaconDB instance.
func setupDB(t *testing.T) *BeaconDB {
	path := getPath(t)
	if err := os.RemoveAll(path); err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}

	db, err := NewDB(path)
	if err != nil {
		t.Fatalf("Failed to instantiate DB: %v", err)
	}

	return db
}

// teardownDB cleans up a test BeaconDB instance.
func teardownDB(t *testing.T, db *BeaconDB) {
	if err := db.Close(); err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}
	if err := os.RemoveAll(getPath(t)); err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}
}
