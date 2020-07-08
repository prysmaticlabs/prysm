package kv

import (
	"bytes"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	bolt "go.etcd.io/bbolt"
)

var migrationArchivedIndex0Key = []byte("archive_index_0")

func migrateArchivedIndex(tx *bolt.Tx) error {
	mb := tx.Bucket(migrationsBucket)
	if b := mb.Get(migrationArchivedIndex0Key); bytes.Equal(b, migrationCompleted) {
		return nil // Migration already completed.
	}
	bkt := tx.Bucket(archivedRootBucket)

	// Migration must be done in reverse order to prevent key collisions during migration.
	c := bkt.Cursor()
	for k, v := c.Last(); k != nil; k, v = c.Prev() {
		idx := bytesutil.BytesToUint64(k)
		// Migrate index to slot.
		slot := idx / params.BeaconConfig().SlotsPerArchivedPoint
		if err := bkt.Put(bytesutil.Uint64ToBytes(slot), v); err != nil {
			return err
		}
		// Delete the old key.
		if err := bkt.Delete(k); err != nil {
			return err
		}
	}

	// Remove the saved bitlists, if they exist.
	if tx.Bucket(slotsHasObjectBucket) != nil {
		if err := tx.Bucket(slotsHasObjectBucket).Delete(savedStateSlotsKey); err != nil {
			return err
		}
		if err := tx.Bucket(slotsHasObjectBucket).Delete(savedBlockSlotsKey); err != nil {
			return err
		}
	}

	return mb.Put(migrationArchivedIndex0Key, migrationCompleted)
}
