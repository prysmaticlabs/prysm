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

// AttestationHistoryForPubKeysV2 accepts an array of validator public keys and returns a mapping of corresponding attestation history.
func (store *Store) AttestationHistoryForPubKeysV2(ctx context.Context, publicKeys [][48]byte) (map[[48]byte]EncHistoryData, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.AttestationHistoryForPubKeysV2")
	defer span.End()

	if len(publicKeys) == 0 {
		return make(map[[48]byte]EncHistoryData), nil
	}

	var err error
	attestationHistoryForVals := make(map[[48]byte]EncHistoryData)
	err = store.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(newHistoricAttestationsBucket)
		for _, key := range publicKeys {
			enc := bucket.Get(key[:])
			var attestationHistory EncHistoryData
			if len(enc) == 0 {
				attestationHistory = NewAttestationHistoryArray(0)
			} else {
				attestationHistory = make(EncHistoryData, len(enc))
				copy(attestationHistory, enc)
			}
			attestationHistoryForVals[key] = attestationHistory
		}
		return nil
	})
	for pk, ah := range attestationHistoryForVals {
		ehd := make(EncHistoryData, len(ah))
		copy(ehd, ah)
		attestationHistoryForVals[pk] = ehd
	}
	return attestationHistoryForVals, err
}

// SaveAttestationHistoryForPubKeysV2 saves the attestation histories for the requested validator public keys.
func (store *Store) SaveAttestationHistoryForPubKeysV2(ctx context.Context, historyByPubKeys map[[48]byte]EncHistoryData) error {
	ctx, span := trace.StartSpan(ctx, "Validator.SaveAttestationHistoryForPubKeysV2")
	defer span.End()

	err := store.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(newHistoricAttestationsBucket)
		for pubKey, encodedHistory := range historyByPubKeys {
			if err := bucket.Put(pubKey[:], encodedHistory); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

// SaveAttestationHistoryForPubKeyV2 saves the attestation history for the requested validator public key.
func (store *Store) SaveAttestationHistoryForPubKeyV2(ctx context.Context, pubKey [48]byte, history EncHistoryData) error {
	ctx, span := trace.StartSpan(ctx, "Validator.SaveAttestationHistoryForPubKeyV2")
	defer span.End()
	err := store.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(newHistoricAttestationsBucket)
		return bucket.Put(pubKey[:], history)
	})

	return err
}

// HighestSignedSourceEpoch returns the highest signed source epoch for a validator public key.
// If no data exists, returning 0 is a sensible default.
func (store *Store) HighestSignedSourceEpoch(ctx context.Context, publicKey [48]byte) (uint64, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.HighestSignedSourceEpoch")
	defer span.End()

	var err error
	var highestSignedSourceEpoch uint64
	err = store.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(highestSignedSourceBucket)
		highestSignedSourceBytes := bucket.Get(publicKey[:])
		// 8 because bytesutil.BytesToUint64BigEndian will return 0 if input is less than 8 bytes.
		if len(highestSignedSourceBytes) < 8 {
			return nil
		}
		highestSignedSourceEpoch = bytesutil.BytesToUint64BigEndian(highestSignedSourceBytes)
		return nil
	})
	return highestSignedSourceEpoch, err
}

// HighestSignedTargetEpoch returns the highest signed target epoch for a validator public key.
// If no data exists, returning 0 is a sensible default.
func (store *Store) HighestSignedTargetEpoch(ctx context.Context, publicKey [48]byte) (uint64, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.HighestSignedTargetEpoch")
	defer span.End()

	var err error
	var highestSignedTargetEpoch uint64
	err = store.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(highestSignedTargetBucket)
		highestSignedTargetBytes := bucket.Get(publicKey[:])
		// 8 because bytesutil.BytesToUint64BigEndian will return 0 if input is less than 8 bytes.
		if len(highestSignedTargetBytes) < 8 {
			return nil
		}
		highestSignedTargetEpoch = bytesutil.BytesToUint64BigEndian(highestSignedTargetBytes)
		return nil
	})
	return highestSignedTargetEpoch, err
}

// SaveHighestSignedSourceEpoch saves the highest signed source epoch for a validator public key.
func (store *Store) SaveHighestSignedSourceEpoch(ctx context.Context, publicKey [48]byte, epoch uint64) error {
	ctx, span := trace.StartSpan(ctx, "Validator.SaveHighestSignedSourceEpoch")
	defer span.End()

	return store.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(highestSignedSourceBucket)

		// If the incoming epoch is higher than the highest signed epoch, override.
		highestSignedSourceBytes := bucket.Get(publicKey[:])
		var highestSignedSourceEpoch uint64
		if len(highestSignedSourceBytes) >= 8 {
			highestSignedSourceEpoch = bytesutil.BytesToUint64BigEndian(highestSignedSourceBytes)
		}
		if len(highestSignedSourceBytes) == 0 || epoch > highestSignedSourceEpoch {
			if err := bucket.Put(publicKey[:], bytesutil.Uint64ToBytesBigEndian(epoch)); err != nil {
				return err
			}
		}
		return nil
	})
}

// SaveHighestSignedTargetEpoch saves the highest signed target epoch for a validator public key.
func (store *Store) SaveHighestSignedTargetEpoch(ctx context.Context, publicKey [48]byte, epoch uint64) error {
	ctx, span := trace.StartSpan(ctx, "Validator.SaveHighestSignedTargetEpoch")
	defer span.End()

	return store.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(highestSignedTargetBucket)

		// If the incoming epoch is higher than the highest signed epoch, override.
		highestSignedTargetBytes := bucket.Get(publicKey[:])
		var highestSignedTargetEpoch uint64
		if len(highestSignedTargetBytes) >= 8 {
			highestSignedTargetEpoch = bytesutil.BytesToUint64BigEndian(highestSignedTargetBytes)
		}
		if len(highestSignedTargetBytes) == 0 || epoch > highestSignedTargetEpoch {
			if err := bucket.Put(publicKey[:], bytesutil.Uint64ToBytesBigEndian(epoch)); err != nil {
				return err
			}
		}
		return nil
	})
}
