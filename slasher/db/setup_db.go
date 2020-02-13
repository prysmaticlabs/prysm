package db

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path"
	"testing"

	"github.com/prysmaticlabs/prysm/slasher/flags"
	"github.com/urfave/cli"
)

// SetupSlasherDB instantiates and returns a SlasherDB instance.
func SetupSlasherDB(t testing.TB, ctx *cli.Context) *Store {
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		t.Fatalf("Could not generate random file path: %v", err)
	}
	p := path.Join(TempDir(), fmt.Sprintf("/%d", randPath))
	if err := os.RemoveAll(p); err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}
	cfg := &Config{cacheItems: 0, maxCacheSize: 0, SpanCacheEnabled: ctx.GlobalBool(flags.UseSpanCacheFlag.Name)}
	db, err := NewDB(p, cfg)
	if err != nil {
		t.Fatalf("Failed to instantiate DB: %v", err)
	}
	return db
}

// SetupSlasherDBDiffCacheSize instantiates and returns a SlasherDB instance with non default cache size.
func SetupSlasherDBDiffCacheSize(t testing.TB, cacheItems int64, maxCacheSize int64) *Store {
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		t.Fatalf("Could not generate random file path: %v", err)
	}
	p := path.Join(TempDir(), fmt.Sprintf("/%d", randPath))
	if err := os.RemoveAll(p); err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}
	cfg := &Config{cacheItems: cacheItems, maxCacheSize: maxCacheSize, SpanCacheEnabled: true}
	db, err := NewDB(p, cfg)
	if err != nil {
		t.Fatalf("Failed to instantiate DB: %v", err)
	}
	return db
}

// TempDir returns a directory path for temporary test storage.
func TempDir() string {
	d := os.Getenv("TEST_TMPDIR")
	// If the test is not run via bazel, the environment var won't be set.
	if d == "" {
		return os.TempDir()
	}
	return d
}

// TeardownSlasherDB cleans up a test SlasherDB instance.
func TeardownSlasherDB(t testing.TB, db *Store) {
	if err := db.Close(); err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}
	if err := os.RemoveAll(db.DatabasePath()); err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}
}
