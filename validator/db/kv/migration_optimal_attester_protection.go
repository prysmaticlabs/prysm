package kv

import (
	"bytes"

	"github.com/golang/snappy"
	bolt "go.etcd.io/bbolt"
)

var migrationOptimalAttesterProtectionKey = []byte("optimal_attester_protection_0")

// Migrate the attestation history data for each validator key into an optimal db schema which
// will completely eradicate its heavy impact on the validator client runtime.
func migrateOptimalAttesterProtection(tx *bolt.Tx) error {
	mb := tx.Bucket(migrationsBucket)
	if b := mb.Get(migrationOptimalAttesterProtectionKey); bytes.Equal(b, migrationCompleted) {
		return nil // Migration already completed.
	}

	// TODO: Implement logic.
	bkt := tx.Bucket(historicAttestationsBucket)

	// Compress all attestation history data.
	if err := bkt.ForEach(func(k, v []byte) error {
		if v == nil {
			return nil
		}
		var attestingHistory EncHistoryData
		var err error
		attestingHistory, err = snappy.Decode(nil /*dst*/, v)
		if err != nil {
			return err
		}

		// Extract every single source, target, signing root
		// from the attesting history then insert them into the
		// respective buckets under the new db schema.
		_ = attestingHistory

		return bkt.Put(k, v)
	}); err != nil {
		return err
	}

	return mb.Put(migrationOptimalAttesterProtectionKey, migrationCompleted)
}
