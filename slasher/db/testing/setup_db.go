package testing

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	slasherDB "github.com/prysmaticlabs/prysm/slasher/db"
	"github.com/prysmaticlabs/prysm/slasher/db/kv"
)

// SetupSlasherDB instantiates and returns a SlasherDB instance.
func SetupSlasherDB(t testing.TB, spanCacheEnabled bool) *kv.Store {
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		t.Fatalf("Could not generate random file path: %v", err)
	}
	p := path.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath))
	if err := os.RemoveAll(p); err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}
	cfg := &kv.Config{CacheItems: 0, MaxCacheSize: 0, SpanCacheEnabled: spanCacheEnabled}
	db, err := slasherDB.NewDB(p, cfg)
	if err != nil {
		t.Fatalf("Failed to instantiate DB: %v", err)
	}
	return db
}

// SetupSlasherDBDiffCacheSize instantiates and returns a SlasherDB instance with non default cache size.
func SetupSlasherDBDiffCacheSize(t testing.TB, cacheItems int64, maxCacheSize int64) *kv.Store {
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		t.Fatalf("Could not generate random file path: %v", err)
	}
	p := path.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath))
	if err := os.RemoveAll(p); err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}
	cfg := &kv.Config{CacheItems: cacheItems, MaxCacheSize: maxCacheSize, SpanCacheEnabled: true}
	newDB, err := slasherDB.NewDB(p, cfg)
	if err != nil {
		t.Fatalf("Failed to instantiate DB: %v", err)
	}
	return newDB
}

// TeardownSlasherDB cleans up a test SlasherDB instance.
func TeardownSlasherDB(t testing.TB, db *kv.Store) {
	if err := db.Close(); err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}
	if err := os.RemoveAll(db.DatabasePath()); err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}
}
