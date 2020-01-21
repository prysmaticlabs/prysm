package db

import (
	"context"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"go.opencensus.io/trace"
)

func unmarshalAttestationHistory(enc []byte) (*slashpb.AttestationHistory, error) {
	history := &slashpb.AttestationHistory{}
	err := proto.Unmarshal(enc, history)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoding")
	}
	return history, nil
}

// AttestationHistory accepts a validator public key and returns the corresponding attestation history.
// Returns nil if there is no attestation history for the validator.
func (db *Store) AttestationHistory(ctx context.Context, publicKey []byte) (*slashpb.AttestationHistory, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.AttestationHistory")
	defer span.End()

	var err error
	var attestationHistory *slashpb.AttestationHistory
	err = db.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicAttestationsBucket)
		enc := bucket.Get(publicKey)
		if enc == nil {
			return nil
		}
		attestationHistory, err = unmarshalAttestationHistory(enc)
		return err
	})
	return attestationHistory, err
}

// SaveAttestationHistory returns the attestation history for the requested validator public key.
func (db *Store) SaveAttestationHistory(ctx context.Context, pubKey []byte, attestationHistory *slashpb.AttestationHistory) error {
	ctx, span := trace.StartSpan(ctx, "Validator.SaveAttestationHistory")
	defer span.End()

	enc, err := proto.Marshal(attestationHistory)
	if err != nil {
		return errors.Wrap(err, "failed to encode attestation history")
	}

	err = db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicAttestationsBucket)
		return bucket.Put(pubKey, enc)
	})
	return err
}

// DeleteAttestationHistory deletes the attestation history for the corresponding validator public key.
func (db *Store) DeleteAttestationHistory(ctx context.Context, pubkey []byte) error {
	ctx, span := trace.StartSpan(ctx, "Validator.DeleteAttestationHistory")
	defer span.End()

	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicAttestationsBucket)
		if err := bucket.Delete(pubkey); err != nil {
			return errors.Wrap(err, "failed to delete the attestation history")
		}
		return nil
	})
}
