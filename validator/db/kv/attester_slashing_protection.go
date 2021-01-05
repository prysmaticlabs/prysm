package kv

import (
	"context"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	bolt "go.etcd.io/bbolt"
)

// CheckSurroundVote --
func (store *Store) CheckSurroundVote(
	ctx context.Context, pubKey [48]byte, att *ethpb.Attestation,
) bool {
	err := store.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(pubKeysBucket)
		sourceEpochsBucket := bucket.Bucket(pubKey[:])
		if sourceEpochsBucket == nil {
			return nil
		}
		return sourceEpochsBucket.ForEach(func(sourceEpochBytes []byte, targetEpochBytes []byte) error {
			sourceEpoch := bytesutil.BytesToUint64BigEndian(sourceEpochBytes)
			targetEpoch := bytesutil.BytesToUint64BigEndian(targetEpochBytes)
			surrounding := att.Data.Source.Epoch < sourceEpoch && att.Data.Target.Epoch > targetEpoch
			surrounded := att.Data.Source.Epoch > sourceEpoch && att.Data.Target.Epoch < targetEpoch
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
	ctx context.Context, pubKey [48]byte, att *ethpb.Attestation,
) error {
	return store.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(pubKeysBucket)
		sourceEpochsBucket, err := bucket.CreateBucketIfNotExists(pubKey[:])
		if err != nil {
			return err
		}
		sourceEpoch := bytesutil.Uint64ToBytesBigEndian(att.Data.Source.Epoch)
		targetEpoch := bytesutil.Uint64ToBytesBigEndian(att.Data.Target.Epoch)
		return sourceEpochsBucket.Put(sourceEpoch, targetEpoch)
	})
}
