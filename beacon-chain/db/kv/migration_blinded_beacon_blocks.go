package kv

import (
	"bytes"
	"context"

	"github.com/prysmaticlabs/prysm/v3/config/features"
	bolt "go.etcd.io/bbolt"
)

var migrationBlindedBeaconBlocksKey = []byte("blinded-beacon-blocks-enabled")

func migrateBlindedBeaconBlocksEnabled(ctx context.Context, db *bolt.DB) error {
	if !features.Get().EnableOnlyBlindedBeaconBlocks {
		return nil // Only write to the migrations bucket if the feature is enabled.
	}
	if updateErr := db.Update(func(tx *bolt.Tx) error {
		mb := tx.Bucket(migrationsBucket)
		if b := mb.Get(migrationBlindedBeaconBlocksKey); bytes.Equal(b, migrationCompleted) {
			return nil // Migration already completed.
		}
		return mb.Put(migrationBlindedBeaconBlocksKey, migrationCompleted)
	}); updateErr != nil {
		return updateErr
	}
	return nil
}
