package testutils

import (
	"os"
	"path"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
)

// SetupDB instantiates a new database for testing purposes
func SetupDB(t *testing.T) *db.DB {
	datadir := path.Join("/tmp", "test")
	if err := os.RemoveAll(datadir); err != nil {
		t.Fatalf("failed to clean dir: %v", err)
	}

	if err := os.MkdirAll(datadir, 0700); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	datafile := path.Join(datadir, "test.db")
	boltDB, err := bolt.Open(datafile, 0600, nil)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	boltDB.NoSync = true

	return db.NewDB(boltDB)
}
