// Package testing allows for spinning up a real bolt-db
// instance for unit tests throughout the Prysm repo.
package testing

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
)

// SetupDB instantiates and returns database backed by key value store.
func SetupDB(t testing.TB) db.Database {
	s, err := kv.NewKVStore(context.Background(), t.TempDir(), &kv.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Fatalf("failed to close database: %v", err)
		}
	})
	return s
}
