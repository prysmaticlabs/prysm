package kv

import (
	"context"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
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
		bucket := tx.Bucket(newHistoricAttestationsBucket)
		return bucket.ForEach(func(key []byte, _ []byte) error {
			pubKeyBytes := [48]byte{}
			copy(pubKeyBytes[:], key)
			attestedPublicKeys = append(attestedPublicKeys, pubKeyBytes)
			return nil
		})
	})
	return attestedPublicKeys, err
}

// AttestationHistoryForPubKeyV2 returns the corresponding attesting
// history for a specified validator public key.
func (store *Store) AttestationHistoryForPubKeyV2(ctx context.Context, publicKey [48]byte) (EncHistoryData, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.AttestationHistoryForPubKeyV2")
	defer span.End()
	if !featureconfig.Get().DisableAttestingHistoryDBCache {
		store.lock.Lock()
		defer store.lock.Unlock()
		if history, ok := store.attestingHistoriesByPubKey[publicKey]; ok {
			return history, nil
		}
	}
	var err error
	var attestationHistory EncHistoryData
	err = store.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(newHistoricAttestationsBucket)
		enc := bucket.Get(publicKey[:])
		if len(enc) == 0 {
			attestationHistory = NewAttestationHistoryArray(0)
		} else {
			attestationHistory = make(EncHistoryData, len(enc))
			copy(attestationHistory, enc)
		}
		return nil
	})
	if !featureconfig.Get().DisableAttestingHistoryDBCache {
		store.attestingHistoriesByPubKey[publicKey] = attestationHistory
	}
	return attestationHistory, err
}

// SaveAttestationHistoryForPubKeyV2 saves the attestation history for the requested validator public key.
func (store *Store) SaveAttestationHistoryForPubKeyV2(
	ctx context.Context,
	pubKey [48]byte,
	history EncHistoryData,
	sourceEpoch,
	targetEpoch uint64,
) error {
	ctx, span := trace.StartSpan(ctx, "Validator.SaveAttestationHistoryForPubKeyV2")
	defer span.End()
	err := store.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(newHistoricAttestationsBucket)
		err := bucket.Put(pubKey[:], history)
		if err != nil {
			return err
		}
		//replace lowest source epoch if needed
		lowestSourceBucket := tx.Bucket(lowestSignedSourceBucket)
		// If the incoming epoch is lower than the lowest signed epoch, override.
		lowestSignedSourceBytes := lowestSourceBucket.Get(pubKey[:])
		var lowestSignedSourceEpoch uint64
		if len(lowestSignedSourceBytes) >= 8 {
			lowestSignedSourceEpoch = bytesutil.BytesToUint64BigEndian(lowestSignedSourceBytes)
		}
		if len(lowestSignedSourceBytes) == 0 || sourceEpoch < lowestSignedSourceEpoch {
			if err := lowestSourceBucket.Put(pubKey[:], bytesutil.Uint64ToBytesBigEndian(sourceEpoch)); err != nil {
				return err
			}
		}
		// replace lowest target epoch if needed
		lowestTargetBucket := tx.Bucket(lowestSignedTargetBucket)
		// If the incoming epoch is lower than the lowest signed epoch, override.
		lowestSignedTargetBytes := lowestTargetBucket.Get(pubKey[:])
		var lowestSignedTargetEpoch uint64
		if len(lowestSignedTargetBytes) >= 8 {
			lowestSignedTargetEpoch = bytesutil.BytesToUint64BigEndian(lowestSignedTargetBytes)
		}
		if len(lowestSignedTargetBytes) == 0 || targetEpoch < lowestSignedTargetEpoch {
			if err := lowestTargetBucket.Put(pubKey[:], bytesutil.Uint64ToBytesBigEndian(targetEpoch)); err != nil {
				return err
			}
		}
		return nil
	})
	if !featureconfig.Get().DisableAttestingHistoryDBCache {
		store.lock.Lock()
		store.attestingHistoriesByPubKey[pubKey] = history
		store.lock.Unlock()
	}
	return err
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
