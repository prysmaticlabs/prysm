// Package testing defines useful helper functions for unit tests with
// the slasher database.
package testing

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/rand"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	slasherDB "github.com/prysmaticlabs/prysm/slasher/db"
	"github.com/prysmaticlabs/prysm/slasher/db/kv"
)

// SetupSlasherDB instantiates and returns a SlasherDB instance.
func SetupSlasherDB(t testing.TB, spanCacheEnabled bool) *kv.Store {
	randPath := rand.NewDeterministicGenerator().Int()
	p := path.Join(testutil.TempDir(), fmt.Sprintf("/%d", randPath))
	if err := os.RemoveAll(p); err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}
	cfg := &kv.Config{}
	db, err := slasherDB.NewDB(p, cfg)
	if err != nil {
		t.Fatalf("Failed to instantiate DB: %v", err)
	}
	db.EnableSpanCache(spanCacheEnabled)
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
