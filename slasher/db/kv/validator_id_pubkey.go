package kv

import (
	"context"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"go.opencensus.io/trace"
)

// ValidatorPubKey accepts validator id and returns the corresponding pubkey.
// Returns nil if the pubkey for this validator id does not exist.
func (db *Store) ValidatorPubKey(ctx context.Context, validatorID uint64) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.ValidatorPubKey")
	defer span.End()
	var pk []byte
	err := db.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(validatorsPublicKeysBucket)
		pk = b.Get(bytesutil.Bytes4(validatorID))
		return nil
	})
	return pk, err
}

// SavePubKey accepts a validator id and its public key  and writes it to disk.
func (db *Store) SavePubKey(ctx context.Context, validatorID uint64, pubKey []byte) error {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.SavePubKey")
	defer span.End()
	err := db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorsPublicKeysBucket)
		key := bytesutil.Bytes4(validatorID)
		if err := bucket.Put(key, pubKey); err != nil {
			return errors.Wrap(err, "failed to add validator public key to slasher db.")
		}
		return nil
	})
	return err
}

// DeletePubKey deletes a public key of a validator id.
func (db *Store) DeletePubKey(ctx context.Context, validatorID uint64) error {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.DeletePubKey")
	defer span.End()
	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorsPublicKeysBucket)
		key := bytesutil.Bytes4(validatorID)
		if err := bucket.Delete(key); err != nil {
			return errors.Wrap(err, "failed to delete public key from validators public key bucket")
		}
		return bucket.Delete(key)
	})
}
