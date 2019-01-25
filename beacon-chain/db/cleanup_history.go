package db

import (
	"errors"

	"github.com/boltdb/bolt"
)

// CleanedFinalizedSlot returns the most recent finalized slot when we did a DB clean up.
func (db *BeaconDB) CleanedFinalizedSlot() (uint64, error) {
	var lastFinalizedSlot uint64

	err := db.view(func(tx *bolt.Tx) error {
		cleanupHistory := tx.Bucket(cleanupHistoryBucket)

		slotEnc := cleanupHistory.Get(cleanedFinalizedSlotKey)
		// If last cleaned slot number is not found, we will return 0 instead
		if slotEnc == nil {
			return nil
		}

		lastFinalizedSlot = decodeToSlotNumber(slotEnc)
		return nil
	})

	return lastFinalizedSlot, err
}

// SaveCleanedFinalizedSlot writes the slot when we did DB cleanup so we can start from here in future cleanup tasks.
func (db *BeaconDB) SaveCleanedFinalizedSlot(slot uint64) error {
	slotEnc := encodeSlotNumber(slot)

	err := db.update(func(tx *bolt.Tx) error {
		cleanupHistory := tx.Bucket(cleanupHistoryBucket)

		if err := cleanupHistory.Put(cleanedFinalizedSlotKey, slotEnc); err != nil {
			return errors.New("failed to store cleaned finalized slot in DB")
		}

		return nil
	})
	return err
}
