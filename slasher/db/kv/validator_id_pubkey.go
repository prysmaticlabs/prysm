package kv

import (
	"context"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/encoding/bytes"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// ValidatorPubKey accepts validator id and returns the corresponding pubkey.
// Returns nil if the pubkey for this validator id does not exist.
func (s *Store) ValidatorPubKey(ctx context.Context, validatorIndex types.ValidatorIndex) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.ValidatorPubKey")
	defer span.End()
	var pk []byte
	err := s.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(validatorsPublicKeysBucket)
		pk = b.Get(bytes.Bytes4(uint64(validatorIndex)))
		return nil
	})
	return pk, err
}

// SavePubKey accepts a validator id and its public key  and writes it to disk.
func (s *Store) SavePubKey(ctx context.Context, validatorIndex types.ValidatorIndex, pubKey []byte) error {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.SavePubKey")
	defer span.End()
	err := s.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorsPublicKeysBucket)
		key := bytes.Bytes4(uint64(validatorIndex))
		if err := bucket.Put(key, pubKey); err != nil {
			return errors.Wrap(err, "failed to add validator public key to slasher s.")
		}
		return nil
	})
	return err
}

// DeletePubKey deletes a public key of a validator id.
func (s *Store) DeletePubKey(ctx context.Context, validatorIndex types.ValidatorIndex) error {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.DeletePubKey")
	defer span.End()
	return s.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorsPublicKeysBucket)
		key := bytes.Bytes4(uint64(validatorIndex))
		if err := bucket.Delete(key); err != nil {
			return errors.Wrap(err, "failed to delete public key from validators public key bucket")
		}
		return bucket.Delete(key)
	})
}
