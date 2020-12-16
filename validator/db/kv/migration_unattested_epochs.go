package kv

import (
	"bytes"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
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

// Migrates to a safer format where unattested epochs (source: 0, signing root: 0x0) are marked by
// FAR_FUTURE_EPOCH as a better default, allowing us to perform better slashing protection.
func markUnattestedEpochsCorrectly(hd EncHistoryData) (EncHistoryData, error) {
	if err := hd.assertSize(); err != nil {
		return nil, err
	}
	// Navigate the history data by HISTORY_SIZE chunks.
	for i := latestEpochWrittenSize; i < len(hd); i += historySize {
		sourceEpoch := bytesutil.FromBytes8(hd[i : i+sourceSize])
		signingRoot := make([]byte, 32)
		copy(signingRoot, hd[i+sourceSize:i+historySize])
		// If source is 0 and signing root is 0x0, we replace that source with FAR_FUTURE_EPOCH
		// which means that epoch in the slice is not yet attested for.
		if sourceEpoch == 0 && bytes.Equal(signingRoot, params.BeaconConfig().ZeroHash[:]) {
			copy(
				hd[i:i+sourceSize],
				bytesutil.Uint64ToBytesLittleEndian(params.BeaconConfig().FarFutureEpoch),
			)
		}
	}
	return hd, nil
}
