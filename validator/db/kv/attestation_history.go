package kv

import (
	"context"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	attHist "github.com/prysmaticlabs/prysm/validator/slashing-protection/local/attesting-history"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// AttestationHistoryForPubKeys accepts an array of validator public keys and
// returns a mapping of corresponding attestation history.
func (store *Store) AttestationHistoryForPubKeys(
	ctx context.Context, publicKeys [][48]byte,
) (map[[48]byte]attHist.History, map[[48]byte]attHist.MinAttestation, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.AttestationHistoryForPubKeys")
	defer span.End()

	if len(publicKeys) == 0 {
		return make(map[[48]byte]attHist.History), nil, nil
	}

	var err error
	attestationHistoryForVals := make(map[[48]byte]attHist.History)
	minAttForVal := make(map[[48]byte]attHist.MinAttestation)
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
			var minAtt attHist.MinAttestation
			var setMin bool
			enc = bucket.Get(attHist.GetMinSourceKey(key))
			if len(enc) != 0 {
				minAtt.Source = bytesutil.BytesToUint64BigEndian(enc)
				setMin = true
			}
			enc = bucket.Get(attHist.GetMinTargetKey(key))
			if len(enc) != 0 {
				minAtt.Target = bytesutil.BytesToUint64BigEndian(enc)
				setMin = true
			}
			if setMin {
				minAttForVal[key] = minAtt
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

	return attestationHistoryForVals, minAttForVal, err
}

// AttestationHistoryForPubKey fetches the attestation history for a public key.
func (store *Store) AttestationHistoryForPubKey(ctx context.Context, publicKey [48]byte) (attHist.History, attHist.MinAttestation, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.AttestationHistoryForPubKey")
	defer span.End()

	var err error
	var attestingHistory attHist.History
	minAtt := attHist.MinAttestation{}
	err = store.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(newHistoricAttestationsBucket)
		enc := bucket.Get(publicKey[:])
		if len(enc) != 0 {
			// Copy to prevent internal array reference being overwritten by boltdb.
			attestingHistory = make(attHist.History, len(enc))
			copy(attestingHistory, enc)
		}
		enc = bucket.Get(attHist.GetMinSourceKey(publicKey))
		if len(enc) != 0 {
			minAtt.Source = bytesutil.BytesToUint64BigEndian(enc)
		}
		enc = bucket.Get(attHist.GetMinTargetKey(publicKey))
		if len(enc) != 0 {
			minAtt.Target = bytesutil.BytesToUint64BigEndian(enc)
		}
		return nil
	})
	return attestingHistory, minAtt, err
}

// SaveAttestationHistoryForPubKeys saves the attestation histories for the requested validator public keys.
func (store *Store) SaveAttestationHistoryForPubKeys(
	ctx context.Context,
	historyByPubKeys map[[48]byte]attHist.History,
	minByPubKeys map[[48]byte]attHist.MinAttestation,
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
		for pubKey, min := range minByPubKeys {
			if err := bucket.Put(attHist.GetMinSourceKey(pubKey), bytesutil.Uint64ToBytesBigEndian(min.Source)); err != nil {
				return err
			}
			if err := bucket.Put(attHist.GetMinTargetKey(pubKey), bytesutil.Uint64ToBytesBigEndian(min.Target)); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

// SaveAttestationHistoryForPubKey saves the attestation history for the requested validator public key.
func (store *Store) SaveAttestationHistoryForPubKey(
	ctx context.Context,
	pubKey [48]byte,
	history attHist.History,
) error {
	ctx, span := trace.StartSpan(ctx, "Validator.SaveAttestationHistoryForPubKey")
	defer span.End()
	err := store.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(newHistoricAttestationsBucket)
		if err := bucket.Put(pubKey[:], history); err != nil {
			return err
		}

		return nil
	})
	return err
}

// SaveMinAttestation saves min attestation values if they are lower from the ones that are currently set in db.
func (store *Store) SaveMinAttestation(ctx context.Context, pubKey [48]byte, minAtt attHist.MinAttestation) error {
	ctx, span := trace.StartSpan(ctx, "Validator.SaveMinAttestation")
	defer span.End()

	return store.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(newHistoricAttestationsBucket)
		minSourceStorageKey := attHist.GetMinSourceKey(pubKey)
		enc := bucket.Get(minSourceStorageKey)
		if len(enc) == 0 {
			minSource := bytesutil.BytesToUint64BigEndian(enc)
			if minAtt.Source < minSource {
				if err := bucket.Put(minSourceStorageKey, bytesutil.Uint64ToBytesBigEndian(minAtt.Source)); err != nil {
					return err
				}
			}
		} else {
			if err := bucket.Put(minSourceStorageKey, bytesutil.Uint64ToBytesBigEndian(minAtt.Source)); err != nil {
				return err
			}
		}
		minTargetStorageKey := attHist.GetMinTargetKey(pubKey)
		enc = bucket.Get(minTargetStorageKey)
		if len(enc) == 0 {
			minTarget := bytesutil.BytesToUint64BigEndian(enc)
			if minAtt.Target < minTarget {
				if err := bucket.Put(minTargetStorageKey, bytesutil.Uint64ToBytesBigEndian(minAtt.Target)); err != nil {
					return err
				}
			}
		} else {
			if err := bucket.Put(minTargetStorageKey, bytesutil.Uint64ToBytesBigEndian(minAtt.Source)); err != nil {
				return err
			}
		}
		return nil
	})

}

func (store *Store) MinAttestation(ctx context.Context, pubKey [48]byte) (*attHist.MinAttestation, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.MinAttestation")
	defer span.End()
	minAtt := attHist.MinAttestation{}
	err := store.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(newHistoricAttestationsBucket)
		minSourceStorageKey := attHist.GetMinSourceKey(pubKey)
		enc := bucket.Get(minSourceStorageKey)
		if len(enc) == 0 {
			minAtt.Source = bytesutil.BytesToUint64BigEndian(enc)
		}
		minTargetStorageKey := attHist.GetMinTargetKey(pubKey)
		enc = bucket.Get(minTargetStorageKey)
		if len(enc) == 0 {
			minAtt.Target = bytesutil.BytesToUint64BigEndian(enc)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &minAtt, nil
}
