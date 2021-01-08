package kv

import (
	"bytes"
	"context"

	"github.com/golang/snappy"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/progressutil"
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

	bkt := tx.Bucket(historicAttestationsBucket)

	// Compress all attestation history data.
	ctx := context.Background()
	bar := progressutil.InitializeProgressBar(bkt.Stats().KeyN, "Migrating attesting history to more efficient format")
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

		bucket := tx.Bucket(pubKeysBucket)
		pkBucket, err := bucket.CreateBucketIfNotExists(k)
		if err != nil {
			return err
		}
		sourceEpochsBucket, err := pkBucket.CreateBucketIfNotExists(attestationSourceEpochsBucket)
		if err != nil {
			return err
		}
		signingRootsBucket, err := pkBucket.CreateBucketIfNotExists(attestationSigningRootsBucket)
		if err != nil {
			return err
		}

		// Extract every single source, target, signing root
		// from the attesting history then insert them into the
		// respective buckets under the new db schema.
		latestEpochWritten, err := attestingHistory.GetLatestEpochWritten(ctx)
		if err != nil {
			return err
		}
		// For every epoch since genesis up to the highest epoch written, we then
		// extract historical data and insert it into the new schema.
		for targetEpoch := uint64(0); targetEpoch <= latestEpochWritten; targetEpoch++ {
			historicalAtt, err := attestingHistory.GetTargetData(ctx, targetEpoch)
			if err != nil {
				return err
			}
			if historicalAtt.IsEmpty() {
				continue
			}
			targetEpochBytes := bytesutil.Uint64ToBytesBigEndian(targetEpoch)
			sourceEpochBytes := bytesutil.Uint64ToBytesBigEndian(historicalAtt.Source)
			if err := sourceEpochsBucket.Put(sourceEpochBytes, targetEpochBytes); err != nil {
				return err
			}
			if err := signingRootsBucket.Put(targetEpochBytes, historicalAtt.SigningRoot); err != nil {
				return err
			}
		}
		return bar.Add(1)
	}); err != nil {
		return err
	}

	return mb.Put(migrationOptimalAttesterProtectionKey, migrationCompleted)
}

func (db *Store) setupHistoryForTest(pubKeys [][48]byte, enc []byte) error {
	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicAttestationsBucket)
		for _, pubKey := range pubKeys {
			if err := bucket.Put(pubKey[:], enc); err != nil {
				return err
			}
		}
		return nil
	})
}

func (store *Store) migrateTxCommit() error {
	tx, err := store.db.Begin(true)
	if err != nil {
		return nil
	}
	mb := tx.Bucket(migrationsBucket)
	if b := mb.Get(migrationOptimalAttesterProtectionKey); bytes.Equal(b, migrationCompleted) {
		return nil // Migration already completed.
	}

	bkt := tx.Bucket(historicAttestationsBucket)

	// Compress all attestation history data.
	ctx := context.Background()
	bar := progressutil.InitializeProgressBar(bkt.Stats().KeyN, "Migrating attesting history to more efficient format")
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

		bucket := tx.Bucket(pubKeysBucket)
		pkBucket, err := bucket.CreateBucketIfNotExists(k)
		if err != nil {
			return err
		}
		sourceEpochsBucket, err := pkBucket.CreateBucketIfNotExists(attestationSourceEpochsBucket)
		if err != nil {
			return err
		}
		signingRootsBucket, err := pkBucket.CreateBucketIfNotExists(attestationSigningRootsBucket)
		if err != nil {
			return err
		}

		// Extract every single source, target, signing root
		// from the attesting history then insert them into the
		// respective buckets under the new db schema.
		latestEpochWritten, err := attestingHistory.GetLatestEpochWritten(ctx)
		if err != nil {
			return err
		}
		// For every epoch since genesis up to the highest epoch written, we then
		// extract historical data and insert it into the new schema.
		for targetEpoch := uint64(0); targetEpoch <= latestEpochWritten; targetEpoch++ {
			historicalAtt, err := attestingHistory.GetTargetData(ctx, targetEpoch)
			if err != nil {
				return err
			}
			if historicalAtt.IsEmpty() {
				continue
			}
			targetEpochBytes := bytesutil.Uint64ToBytesBigEndian(targetEpoch)
			sourceEpochBytes := bytesutil.Uint64ToBytesBigEndian(historicalAtt.Source)
			if err := sourceEpochsBucket.Put(sourceEpochBytes, targetEpochBytes); err != nil {
				return err
			}
			if err := signingRootsBucket.Put(targetEpochBytes, historicalAtt.SigningRoot); err != nil {
				return err
			}
		}
		return bar.Add(1)
	}); err != nil {
		return err
	}

	if err := mb.Put(migrationOptimalAttesterProtectionKey, migrationCompleted); err != nil {
		return err
	}

	return tx.Commit()
}
