// Package testing allows for spinning up a real bolt-db
// instance for unit tests throughout the Prysm repo.
package testing

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	"github.com/prysmaticlabs/prysm/shared/rand"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

// SetupDB instantiates and returns database backed by key value store.
func SetupDB(t testing.TB) (db.Database, *cache.StateSummaryCache) {
	randPath := rand.NewDeterministicGenerator().Int()
	p := path.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath))
	if err := os.RemoveAll(p); err != nil {
		t.Fatalf("failed to remove directory: %v", err)
	}
	sc := cache.NewStateSummaryCache()
	s, err := kv.NewKVStore(p, sc)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Fatalf("failed to close database: %v", err)
		}
		if err := os.Remove(s.DatabasePath()); err != nil {
			t.Fatalf("could not remove tmp db dir: %v", err)
		}
	})
	return s, sc
}
