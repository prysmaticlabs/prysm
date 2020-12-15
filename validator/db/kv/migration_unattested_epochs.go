package kv

import (
	"bytes"

	bolt "go.etcd.io/bbolt"
)

var migrationUnattestedEpochs0Key = []byte("unattested_epochs_0")

func migrateUnattestedEpochsForAttestingHistory(tx *bolt.Tx) error {
	mb := tx.Bucket(migrationsBucket)
	if b := mb.Get(migrationUnattestedEpochs0Key); bytes.Equal(b, migrationCompleted) {
		return nil // Migration already completed.
	}

	bucket := tx.Bucket(newHistoricAttestationsBucket)
	err := bucket.ForEach(func(publicKeyBytes []byte, encodedHistory []byte) error {
		if len(publicKeyBytes) != 48 {
			return nil
		}
		pubKey := [48]byte{}
		copy(pubKey[:], publicKeyBytes)
		var attestationHistory EncHistoryData
		if len(encodedHistory) == 0 {
			return nil
		} else {
			history := make(EncHistoryData, len(encodedHistory))
			copy(history, encodedHistory)
			attestationHistory = history
		}
		newHist, err := markUnattestedEpochsCorrectly(attestationHistory)
		if err != nil {
			return err
		}
		if err := bucket.Put(pubKey[:], newHist); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	// Mark migration complete.
	return mb.Put(migrationUnattestedEpochs0Key, migrationCompleted)
}
