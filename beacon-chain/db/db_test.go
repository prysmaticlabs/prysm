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

func setupDB(t *testing.T) *BeaconDB {
	path := getPath(t)
	os.RemoveAll(path)

	db, err := NewDB(path)
	if err != nil {
		t.Fatalf("Failed to instantiate DB: %v", err)
	}

	return db
}

func teardownDB(t *testing.T, db *BeaconDB) {
	db.Close()
	os.RemoveAll(getPath(t))
}
