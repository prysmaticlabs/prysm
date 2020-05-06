package kv

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	"gopkg.in/urfave/cli.v2"
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
	cfg := &Config{}
	db, err := NewKVStore(p, cfg)
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

func setupDBDiffCacheSize(t testing.TB, cacheSize int) *Store {
	randPath, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		t.Fatalf("Could not generate random file path: %v", err)
	}
	p := path.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath))
	if err := os.RemoveAll(p); err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}
	cfg := &Config{SpanCacheSize: cacheSize}
	db, err := NewKVStore(p, cfg)
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
