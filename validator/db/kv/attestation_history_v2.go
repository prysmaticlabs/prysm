package kv

import (
	"context"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// AttestedPublicKeys retrieves all public keys in our attestation history bucket.
func (store *Store) AttestedPublicKeys(ctx context.Context) ([][48]byte, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.AttestedPublicKeys")
	defer span.End()
	var err error
	attestedPublicKeys := make([][48]byte, 0)
	err = store.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicAttestationsBucket)
		return bucket.ForEach(func(key []byte, _ []byte) error {
			pubKeyBytes := [48]byte{}
			copy(pubKeyBytes[:], key)
			attestedPublicKeys = append(attestedPublicKeys, pubKeyBytes)
			return nil
		})
	})
	return attestedPublicKeys, err
}

// LowestSignedSourceEpoch returns the lowest signed source epoch for a validator public key.
// If no data exists, returning 0 is a sensible default.
func (store *Store) LowestSignedSourceEpoch(ctx context.Context, publicKey [48]byte) (uint64, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.LowestSignedSourceEpoch")
	defer span.End()

	var err error
	var lowestSignedSourceEpoch uint64
	err = store.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(lowestSignedSourceBucket)
		lowestSignedSourceBytes := bucket.Get(publicKey[:])
		// 8 because bytesutil.BytesToUint64BigEndian will return 0 if input is less than 8 bytes.
		if len(lowestSignedSourceBytes) < 8 {
			return nil
		}
		lowestSignedSourceEpoch = bytesutil.BytesToUint64BigEndian(lowestSignedSourceBytes)
		return nil
	})
	return lowestSignedSourceEpoch, err
}

// LowestSignedTargetEpoch returns the lowest signed target epoch for a validator public key.
// If no data exists, returning 0 is a sensible default.
func (store *Store) LowestSignedTargetEpoch(ctx context.Context, publicKey [48]byte) (uint64, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.LowestSignedTargetEpoch")
	defer span.End()

	var err error
	var lowestSignedTargetEpoch uint64
	err = store.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(lowestSignedTargetBucket)
		lowestSignedTargetBytes := bucket.Get(publicKey[:])
		// 8 because bytesutil.BytesToUint64BigEndian will return 0 if input is less than 8 bytes.
		if len(lowestSignedTargetBytes) < 8 {
			return nil
		}
		lowestSignedTargetEpoch = bytesutil.BytesToUint64BigEndian(lowestSignedTargetBytes)
		return nil
	})
	return lowestSignedTargetEpoch, err
}

// SaveLowestSignedSourceEpoch saves the lowest signed source epoch for a validator public key.
func (store *Store) SaveLowestSignedSourceEpoch(ctx context.Context, publicKey [48]byte, epoch uint64) error {
	ctx, span := trace.StartSpan(ctx, "Validator.SaveLowestSignedSourceEpoch")
	defer span.End()

	return store.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(lowestSignedSourceBucket)

		// If the incoming epoch is lower than the lowest signed epoch, override.
		lowestSignedSourceBytes := bucket.Get(publicKey[:])
		var lowestSignedSourceEpoch uint64
		if len(lowestSignedSourceBytes) >= 8 {
			lowestSignedSourceEpoch = bytesutil.BytesToUint64BigEndian(lowestSignedSourceBytes)
		}
		if len(lowestSignedSourceBytes) == 0 || epoch < lowestSignedSourceEpoch {
			if err := bucket.Put(publicKey[:], bytesutil.Uint64ToBytesBigEndian(epoch)); err != nil {
				return err
			}
		}
		return nil
	})
}

// SaveLowestSignedTargetEpoch saves the lowest signed target epoch for a validator public key.
func (store *Store) SaveLowestSignedTargetEpoch(ctx context.Context, publicKey [48]byte, epoch uint64) error {
	ctx, span := trace.StartSpan(ctx, "Validator.SaveLowestSignedTargetEpoch")
	defer span.End()

	return store.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(lowestSignedTargetBucket)

		// If the incoming epoch is lower than the lowest signed epoch, override.
		lowestSignedTargetBytes := bucket.Get(publicKey[:])
		var lowestSignedTargetEpoch uint64
		if len(lowestSignedTargetBytes) >= 8 {
			lowestSignedTargetEpoch = bytesutil.BytesToUint64BigEndian(lowestSignedTargetBytes)
		}
		if len(lowestSignedTargetBytes) == 0 || epoch < lowestSignedTargetEpoch {
			if err := bucket.Put(publicKey[:], bytesutil.Uint64ToBytesBigEndian(epoch)); err != nil {
				return err
			}
		}
		return nil
	})
}
