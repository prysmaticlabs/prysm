package kv

import (
	"bytes"
	"context"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/progressutil"
	bolt "go.etcd.io/bbolt"
)

var migrationOptimalAttesterProtectionKey = []byte("optimal_attester_protection_0")

// Migrate attester protection to a more optimal format in the DB. Given we
// stored attesting history as large, 2Mb arrays per validator, we need to perform
// this migration differently than the rest, ensuring we perform each expensive bolt
// update in its own transaction to prevent having everything on the heap.
func (store *Store) migrateOptimalAttesterProtection(ctx context.Context) error {
	publicKeyBytes := make([][]byte, 0)
	attestingHistoryBytes := make([][]byte, 0)
	numKeys := 0
	err := store.db.Update(func(tx *bolt.Tx) error {
		mb := tx.Bucket(migrationsBucket)
		if b := mb.Get(migrationOptimalAttesterProtectionKey); bytes.Equal(b, migrationCompleted) {
			return nil // Migration already completed.
		}

		bkt := tx.Bucket(historicAttestationsBucket)
		numKeys = bkt.Stats().KeyN
		if err := bkt.ForEach(func(k, v []byte) error {
			if v == nil {
				return nil
			}
			bucket := tx.Bucket(pubKeysBucket)
			pkBucket, err := bucket.CreateBucketIfNotExists(k)
			if err != nil {
				return err
			}
			_, err = pkBucket.CreateBucketIfNotExists(attestationSourceEpochsBucket)
			if err != nil {
				return err
			}
			_, err = pkBucket.CreateBucketIfNotExists(attestationSigningRootsBucket)
			if err != nil {
				return err
			}
			nk := make([]byte, len(k))
			copy(nk, k)
			nv := make([]byte, len(v))
			copy(nv, v)
			publicKeyBytes = append(publicKeyBytes, nk)
			attestingHistoryBytes = append(attestingHistoryBytes, nv)
			return nil
		}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	bar := progressutil.InitializeProgressBar(numKeys, "Migrating attesting history to more efficient format")
	for i, publicKey := range publicKeyBytes {
		var attestingHistory deprecatedEncodedAttestingHistory
		attestingHistory = attestingHistoryBytes[i]
		err = store.db.Update(func(tx *bolt.Tx) error {
			if attestingHistory == nil {
				return nil
			}
			bucket := tx.Bucket(pubKeysBucket)
			pkBucket := bucket.Bucket(publicKey)
			sourceEpochsBucket := pkBucket.Bucket(attestationSourceEpochsBucket)

			signingRootsBucket := pkBucket.Bucket(attestationSigningRootsBucket)

			// Extract every single source, target, signing root
			// from the attesting history then insert them into the
			// respective buckets under the new db schema.
			latestEpochWritten, err := attestingHistory.getLatestEpochWritten(ctx)
			if err != nil {
				return err
			}
			// For every epoch since genesis up to the highest epoch written, we then
			// extract historical data and insert it into the new schema.
			for targetEpoch := uint64(0); targetEpoch <= latestEpochWritten; targetEpoch++ {
				historicalAtt, err := attestingHistory.getTargetData(ctx, targetEpoch)
				if err != nil {
					return err
				}
				if historicalAtt.isEmpty() {
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
		})
		if err != nil {
			return err
		}
	}

	return store.db.Update(func(tx *bolt.Tx) error {
		mb := tx.Bucket(migrationsBucket)
		if err := mb.Put(migrationOptimalAttesterProtectionKey, migrationCompleted); err != nil {
			return err
		}
		return nil
	})
}
