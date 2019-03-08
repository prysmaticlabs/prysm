package db

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/boltdb/bolt"
)

// UpdateSlot updates the slot number of the chain.
func (db *BeaconDB) UpdateSlot(slot uint64) error {
	return db.update(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)
		buf :=make([]byte, 64)
		binary.LittleEndian.PutUint64(buf, slot)

		if err := chainInfo.Put(slotLookupKey, buf); err != nil {
			return fmt.Errorf("failed to record the slot nummber as the head of the main chain: %v", err)
		}

		return nil
	})
}

// Slot returns the slot number of the main chain.
func (db *BeaconDB) Slot() (uint64, error) {
	var slot uint64
	err := db.view(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)

		buf := chainInfo.Get(slotLookupKey)
		if buf == nil {
			return errors.New("unable to determine slot number")
		}

		slot = binary.LittleEndian.Uint64(buf)
		return nil
	})
	return slot, err
}
