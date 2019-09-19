package testing

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

// SetupDB instantiates and returns database backed by key value store.
func SetupDB(t testing.TB) db.Database {
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		t.Fatalf("could not generate random file path: %v", err)
	}
	p := path.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath))
	if err := os.RemoveAll(p); err != nil {
		t.Fatalf("failed to remove directory: %v", err)
	}
	s, err := kv.NewKVStore(p)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// TeardownDB closes a database and destroys the files at the database path.
func TeardownDB(t testing.TB, db db.Database) {
	if err := db.Close(); err != nil {
		t.Fatalf("failed to close database: %v", err)
	}
	if err := os.RemoveAll(db.DatabasePath()); err != nil {
		t.Fatalf("could not remove tmp db dir: %v", err)
	}
}
