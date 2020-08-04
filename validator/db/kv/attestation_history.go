package kv

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/params"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

func unmarshalAttestationHistory(ctx context.Context, enc []byte) (*slashpb.AttestationHistory, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.unmarshalAttestationHistory")
	defer span.End()
	history := &slashpb.AttestationHistory{}
	if err := proto.Unmarshal(enc, history); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoding")
	}
	return history, nil
}

// AttestationHistoryForPubKeys accepts an array of validator public keys and returns a mapping of corresponding attestation history.
func (store *Store) AttestationHistoryForPubKeys(ctx context.Context, publicKeys [][48]byte) (map[[48]byte]*slashpb.AttestationHistory, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.AttestationHistoryForPubKeys")
	defer span.End()

	if len(publicKeys) == 0 {
		return make(map[[48]byte]*slashpb.AttestationHistory), nil
	}

	var err error
	attestationHistoryForVals := make(map[[48]byte]*slashpb.AttestationHistory)
	err = store.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicAttestationsBucket)
		for _, key := range publicKeys {
			enc := bucket.Get(key[:])
			var attestationHistory *slashpb.AttestationHistory
			if len(enc) == 0 {
				newMap := make(map[uint64]uint64)
				newMap[0] = params.BeaconConfig().FarFutureEpoch
				attestationHistory = &slashpb.AttestationHistory{
					TargetToSource: newMap,
				}
			} else {
				attestationHistory, err = unmarshalAttestationHistory(ctx, enc)
				if err != nil {
					return err
				}
			}
			attestationHistoryForVals[key] = attestationHistory
		}
		return nil
	})
	return attestationHistoryForVals, err
}

// SaveAttestationHistoryForPubKeys saves the attestation histories for the requested validator public keys.
func (store *Store) SaveAttestationHistoryForPubKeys(ctx context.Context, historyByPubKeys map[[48]byte]*slashpb.AttestationHistory) error {
	ctx, span := trace.StartSpan(ctx, "Validator.SaveAttestationHistory")
	defer span.End()

	encoded := make(map[[48]byte][]byte)
	for pubKey, history := range historyByPubKeys {
		enc, err := proto.Marshal(history)
		if err != nil {
			return errors.Wrap(err, "failed to encode attestation history")
		}
		encoded[pubKey] = enc
	}

	err := store.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicAttestationsBucket)
		for pubKey, encodedHistory := range encoded {
			if err := bucket.Put(pubKey[:], encodedHistory); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}
