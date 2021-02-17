package kv

import (
	"bytes"
	"context"

	"github.com/prysmaticlabs/prysm/shared/progressutil"
	bolt "go.etcd.io/bbolt"
)

var migrationSourceTargetEpochsBucketKey = []byte("source_target_epochs_bucket_0")

func (s *Store) migrateSourceTargetEpochsBucketUp(ctx context.Context) error {
	// First, we extract the public keys we need to migrate data for.
	publicKeyBytes := make([][]byte, 0)
	err := s.db.View(func(tx *bolt.Tx) error {
		mb := tx.Bucket(migrationsBucket)
		if b := mb.Get(migrationSourceTargetEpochsBucketKey); bytes.Equal(b, migrationCompleted) {
			return nil // Migration already completed.
		}
		bkt := tx.Bucket(pubKeysBucket)
		return bkt.ForEach(func(k, _ []byte) error {
			if k == nil {
				return nil
			}
			nk := make([]byte, len(k))
			copy(nk, k)
			publicKeyBytes = append(publicKeyBytes, nk)
			return nil
		})
	})
	if err != nil {
		return err
	}

	// Next up, we initiate a bolt transaction for each public key.
	bar := progressutil.InitializeProgressBar(
		len(publicKeyBytes), "Adding optimizations for validator slashing protection",
	)
	for _, pubKey := range publicKeyBytes {
		err = s.db.Update(func(tx *bolt.Tx) error {
			bkt := tx.Bucket(pubKeysBucket)
			pkb := bkt.Bucket(pubKey)
			sourceBucket := pkb.Bucket(attestationSourceEpochsBucket)
			if sourceBucket == nil {
				return nil
			}
			targetBucket, err := pkb.CreateBucketIfNotExists(attestationTargetEpochsBucket)
			if err != nil {
				return err
			}
			return sourceBucket.ForEach(func(sourceEpochBytes, targetEpochsBytes []byte) error {
				for i := 0; i < len(targetEpochsBytes); i += 8 {
					if err := insertTargetSource(
						targetBucket,
						targetEpochsBytes[i:i+8],
						sourceEpochBytes,
					); err != nil {
						return err
					}
				}
				return nil
			})
		})
		if err != nil {
			return err
		}
		if err := bar.Add(1); err != nil {
			return err
		}
	}

	// Finally we mark the migration as completed.
	return s.db.Update(func(tx *bolt.Tx) error {
		mb := tx.Bucket(migrationsBucket)
		return mb.Put(migrationSourceTargetEpochsBucketKey, migrationCompleted)
	})
}

func (s *Store) migrateSourceTargetEpochsBucketDown(ctx context.Context) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		migrationsBkt := tx.Bucket(migrationsBucket)
		return migrationsBkt.Delete(migrationOptimalAttesterProtectionKey)
	})
}

func insertTargetSource(bkt *bolt.Bucket, targetEpochBytes, sourceEpochBytes []byte) error {
	var existingAttestedSourceBytes []byte
	if existing := bkt.Get(targetEpochBytes); existing != nil {
		existingAttestedSourceBytes = append(existing, sourceEpochBytes...)
	} else {
		existingAttestedSourceBytes = sourceEpochBytes
	}
	return bkt.Put(targetEpochBytes, existingAttestedSourceBytes)
}
