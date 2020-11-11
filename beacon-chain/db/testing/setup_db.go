// Package testing allows for spinning up a real bolt-db
// instance for unit tests throughout the Prysm repo.
package testing

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
)

// SetupDB instantiates and returns database backed by key value store.
func SetupDB(t testing.TB) (db.Database, *cache.StateSummaryCache) {
	sc := cache.NewStateSummaryCache()
	s, err := kv.NewKVStore(t.TempDir(), sc)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Fatalf("failed to close database: %v", err)
		}
	})
	return s, sc
}
