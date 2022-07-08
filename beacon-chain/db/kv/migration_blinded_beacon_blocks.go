package kv

import (
	"bytes"
	"context"

	"github.com/prysmaticlabs/prysm/config/features"
	bolt "go.etcd.io/bbolt"
)

var migrationBlindedBeaconBlocksKey = []byte("blinded-beacon-blocks-enabled")

func migrateBlindedBeaconBlocksEnabled(ctx context.Context, db *bolt.DB) error {
	if updateErr := db.Update(func(tx *bolt.Tx) error {
		mb := tx.Bucket(migrationsBucket)
		if b := mb.Get(migrationBlindedBeaconBlocksKey); bytes.Equal(b, migrationCompleted) {
			return nil // Migration already completed.
		}
		if !features.Get().EnableOnlyBlindedBeaconBlocks {
			return nil // Only write to the migrations bucket if the feature is enabled.
		}
		return mb.Put(migrationBlindedBeaconBlocksKey, migrationCompleted)
	}); updateErr != nil {
		return updateErr
	}
	return nil
}
