package kv

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/slasher/db/iface"
	"github.com/prysmaticlabs/prysm/slasher/flags"
	"github.com/urfave/cli"
)

func setupDB(t testing.TB, ctx *cli.Context) *Store {
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		t.Fatalf("Could not generate random file path: %v", err)
	}
	p := path.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath))
	if err := os.RemoveAll(p); err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}
	cfg := &Config{SpanCacheEnabled: ctx.GlobalBool(flags.UseSpanCacheFlag.Name)}
	db, err := NewKVStore(p, cfg)
	if err != nil {
		t.Fatalf("Failed to instantiate DB: %v", err)
	}
	return db
}

func setupDBDiffCacheSize(t testing.TB, cacheSize int) *Store {
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		t.Fatalf("Could not generate random file path: %v", err)
	}
	p := path.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath))
	if err := os.RemoveAll(p); err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}
	cfg := &Config{SpanCacheEnabled: true, SpanCacheSize: cacheSize}
	newDB, err := NewKVStore(p, cfg)
	if err != nil {
		t.Fatalf("Failed to instantiate DB: %v", err)
	}
	return newDB
}

func teardownDB(t testing.TB, db iface.Database) {
	if err := db.Close(); err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}
	if err := os.RemoveAll(db.DatabasePath()); err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}
}
