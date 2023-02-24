package kv

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/config/features"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	bolt "go.etcd.io/bbolt"
)

// setupDB instantiates and returns a Store instance.
func setupDB(t testing.TB) *Store {
	db, err := NewKVStore(context.Background(), t.TempDir())
	require.NoError(t, err, "Failed to instantiate DB")
	t.Cleanup(func() {
		require.NoError(t, db.Close(), "Failed to close database")
	})
	return db
}

func Test_checkNeedsResync(t *testing.T) {
	store := setupDB(t)
	resetFn := features.InitWithReset(&features.Flags{
		EnableOnlyBlindedBeaconBlocks: false,
	})
	defer resetFn()
	require.NoError(t, store.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(migrationsBucket)
		return bkt.Put(migrationBlindedBeaconBlocksKey, migrationCompleted)
	}))
	err := store.checkNeedsResync()
	require.ErrorContains(t, "your node must resync", err)
}
