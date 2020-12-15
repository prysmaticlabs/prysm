package kv

import (
	"bytes"

	"github.com/golang/snappy"
	bolt "go.etcd.io/bbolt"
)

var migrationSnappyAttestationHistory0Key = []byte("snappy_attestation_history_0")


// Migrate the attestation history data to use snappy compression on disk. This paradigm will
// significantly reduce disk I/O at the cost of a slightly increased CPU usage. Early benchmarks and
// tests indicate that this compression saves 25% on disk I/O and storage.
func migrateSnappyAttestationHistory(tx *bolt.Tx) error {
	mb := tx.Bucket(migrationsBucket)
	if b := mb.Get(migrationSnappyAttestationHistory0Key); bytes.Equal(b, migrationCompleted) {
		return nil // Migration already completed.
	}

	bkt := tx.Bucket(historicAttestationsBucket)

	// Compress all attestation history data.
	if err := bkt.ForEach(func(k, v []byte) error {
		enc := snappy.Encode(nil /*dst*/, v)
		return bkt.Put(k, enc)
	}); err != nil {
		return err
	}

	return mb.Put(migrationSnappyAttestationHistory0Key, migrationCompleted)
}

