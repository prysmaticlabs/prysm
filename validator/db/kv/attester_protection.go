package kv

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	bolt "go.etcd.io/bbolt"
)

// CheckSlashableAttestation --
func (store *Store) CheckSlashableAttestation(
	ctx context.Context, pubKey [48]byte, sourceEpoch, targetEpoch uint64,
) bool {
	err := store.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(pubKeysBucket)
		sourceEpochsBucket := bucket.Bucket(pubKey[:])
		if sourceEpochsBucket == nil {
			return nil
		}
		return sourceEpochsBucket.ForEach(func(sourceEpochBytes []byte, targetEpochBytes []byte) error {
			existingSourceEpoch := bytesutil.BytesToUint64BigEndian(sourceEpochBytes)
			existingTargetEpoch := bytesutil.BytesToUint64BigEndian(targetEpochBytes)
			surrounding := sourceEpoch < existingSourceEpoch && targetEpoch > existingTargetEpoch
			surrounded := sourceEpoch > existingSourceEpoch && targetEpoch < existingTargetEpoch
			if surrounding || surrounded {
				// Returning an error allows us to exit early.
				return errors.New("slashable vote found")
			}
			return nil
		})
	})
	return err != nil
}

// ApplyAttestationForPubKey --
func (store *Store) ApplyAttestationForPubKey(
	ctx context.Context, pubKey [48]byte, sourceEpoch, targetEpoch uint64,
) error {
	return store.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(pubKeysBucket)
		sourceEpochsBucket, err := bucket.CreateBucketIfNotExists(pubKey[:])
		if err != nil {
			return err
		}
		sourceEpoch := bytesutil.Uint64ToBytesBigEndian(sourceEpoch)
		targetEpoch := bytesutil.Uint64ToBytesBigEndian(targetEpoch)

		signingRootsBucket, err := bucket.CreateBucketIfNotExists(attestationSigningRootsBucket)
		if err != nil {
			return err
		}
		_ = signingRootsBucket
		return sourceEpochsBucket.Put(sourceEpoch, targetEpoch)
	})
}
