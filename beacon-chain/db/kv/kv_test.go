package kv

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

// setupDB instantiates and returns a Store instance.
func setupDB(t testing.TB) *Store {
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		t.Fatalf("Could not generate random file path: %v", err)
	}
	p := path.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath))
	if err := os.RemoveAll(p); err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}
	db, err := NewKVStore(p, cache.NewStateSummaryCache())
	if err != nil {
		t.Fatalf("Failed to instantiate DB: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Failed to close database: %v", err)
		}
		if err := os.RemoveAll(db.DatabasePath()); err != nil {
			t.Fatalf("Failed to remove directory: %v", err)
		}
	})
	return db
}

func TestStore_DatabasePath(t *testing.T) {
	db := setupDB(t)
	dbPath := db.DatabasePath()
	if !strings.Contains(dbPath, databaseFileName) {
		t.Fatal("Expected filepath to lead to database file")
	}
}
