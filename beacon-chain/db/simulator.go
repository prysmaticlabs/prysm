package db

import (
	"github.com/boltdb/bolt"
)

// GetSimulatorSlot returns the last saved simulator slot
// from the disk.
func (db *BeaconDB) GetSimulatorSlot() (uint64, error) {
	var slot uint64
	err := db.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(simulatorBucket)

		enc := b.Get(simSlotLookupKey)
		if enc == nil {
			return nil
		}

		slot = decodeToSlotNumber(enc)
		return nil
	})

	return slot, err
}

// SaveSimulatorSlot saves the current slot of the simulator to the disk.
func (db *BeaconDB) SaveSimulatorSlot(slot uint64) error {
	return db.update(func(tx *bolt.Tx) error {
		b := tx.Bucket(simulatorBucket)
		enc := encodeSlotNumber(slot)

		return b.Put(simSlotLookupKey, enc)
	})
}
