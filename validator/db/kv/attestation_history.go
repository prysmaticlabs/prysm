package kv

import (
	"context"

	attHist "github.com/prysmaticlabs/prysm/validator/slashing-protection/local/attesting-history"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// AttestationHistoryForPubKeys accepts an array of validator public keys and
// returns a mapping of corresponding attestation history.
func (store *Store) AttestationHistoryForPubKeys(
	ctx context.Context, publicKeys [][48]byte,
) (map[[48]byte]attHist.History, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.AttestationHistoryForPubKeys")
	defer span.End()

	if len(publicKeys) == 0 {
		return make(map[[48]byte]attHist.History), nil
	}

	var err error
	attestationHistoryForVals := make(map[[48]byte]attHist.History)
	err = store.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(newHistoricAttestationsBucket)
		for _, key := range publicKeys {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			enc := bucket.Get(key[:])
			var attestationHistory attHist.History
			if len(enc) == 0 {
				attestationHistory = attHist.New(0)
			} else {
				attestationHistory = enc
			}
			attestationHistoryForVals[key] = attestationHistory
		}
		return nil
	})
	for pk, ah := range attestationHistoryForVals {
		ehd := make(attHist.History, len(ah))
		copy(ehd, ah)
		attestationHistoryForVals[pk] = ehd
	}
	return attestationHistoryForVals, err
}

// AttestationHistoryForPubKey fetches the attestation history for a public key.
func (store *Store) AttestationHistoryForPubKey(ctx context.Context, publicKey [48]byte) (attHist.History, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.AttestationHistoryForPubKey")
	defer span.End()

	var err error
	var attestingHistory attHist.History
	err = store.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(newHistoricAttestationsBucket)
		enc := bucket.Get(publicKey[:])
		if len(enc) != 0 {
			// Copy to prevent internal array reference being overwritten by boltdb.
			attestingHistory = make(attHist.History, len(enc))
			copy(attestingHistory, enc)
		}
		return nil
	})
	return attestingHistory, err
}

// SaveAttestationHistoryForPubKeys saves the attestation histories for the requested validator public keys.
func (store *Store) SaveAttestationHistoryForPubKeys(
	ctx context.Context, historyByPubKeys map[[48]byte]attHist.History,
) error {
	ctx, span := trace.StartSpan(ctx, "Validator.SaveAttestationHistoryForPubKeys")
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

// SaveAttestationHistoryForPubKey saves the attestation history for the requested validator public key.
func (store *Store) SaveAttestationHistoryForPubKey(
	ctx context.Context, pubKey [48]byte, history attHist.History,
) error {
	ctx, span := trace.StartSpan(ctx, "Validator.SaveAttestationHistoryForPubKey")
	defer span.End()
	err := store.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(newHistoricAttestationsBucket)
		return bucket.Put(pubKey[:], history)
	})
	return err
}
