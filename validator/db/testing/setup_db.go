package testing

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v4/validator/db/iface"
	"github.com/prysmaticlabs/prysm/v4/validator/db/kv"
)

// SetupDB instantiates and returns a DB instance for the validator client.
func SetupDB(t testing.TB, config *kv.Config) iface.ValidatorDB {
	db, err := kv.NewKVStore(context.Background(), t.TempDir(), config)
	if err != nil {
		t.Fatalf("Failed to instantiate DB: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Failed to close database: %v", err)
		}
		if err := db.ClearDB(); err != nil {
			t.Fatalf("Failed to clear database: %v", err)
		}
	})
	return db
}
