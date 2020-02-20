package kv

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// ValidatorIndex by public key.
func (k *Store) ValidatorIndex(ctx context.Context, publicKey []byte) (uint64, bool, error) {
	if len(publicKey) != params.BeaconConfig().BLSPubkeyLength {
		return 0, false, errors.New("incorrect key length")
	}
	// Return latest validatorIndex from cache if it exists.
	if v, ok := k.validatorIndexCache.Get(string(publicKey)); v != nil && ok {
		return v.(uint64), true, nil
	}
	var validatorIdx uint64
	var ok bool
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(validatorsBucket)
		enc := bkt.Get(publicKey)
		if enc == nil {
			return nil
		}
		validatorIdx = binary.LittleEndian.Uint64(enc)

		ok = true
		return nil
	})
	return validatorIdx, ok, err
}

// HasValidatorIndex verifies if a validator's index by public key exists in the db.
func (k *Store) HasValidatorIndex(ctx context.Context, publicKey []byte) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HasValidatorIndex")
	defer span.End()
	if v, ok := k.validatorIndexCache.Get(string(publicKey)); v != nil && ok {
		return true
	}
	exists := false
	// #nosec G104. Always returns nil.
	k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(validatorsBucket)
		exists = bkt.Get(publicKey) != nil
		return nil
	})
	return exists
}

// DeleteValidatorIndex clears a validator index from the db by the validator's public key.
func (k *Store) DeleteValidatorIndex(ctx context.Context, publicKey []byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DeleteValidatorIndex")
	defer span.End()
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorsBucket)
		k.validatorIndexCache.Del(string(publicKey))
		return bucket.Delete(publicKey)
	})
}

// SaveValidatorIndex by public key in the db.
func (k *Store) SaveValidatorIndex(ctx context.Context, publicKey []byte, validatorIdx uint64) error {
	if len(publicKey) != params.BeaconConfig().BLSPubkeyLength {
		return errors.New("incorrect key length")
	}
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveValidatorIndex")
	defer span.End()
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorsBucket)
		buf := uint64ToBytes(validatorIdx)
		k.validatorIndexCache.Set(string(publicKey), validatorIdx, int64(len(buf)))
		return bucket.Put(publicKey, buf)
	})
}

// SaveValidatorIndices by public keys to the DB.
func (k *Store) SaveValidatorIndices(ctx context.Context, publicKeys [][48]byte, validatorIndices []uint64) error {
	if len(publicKeys) != len(validatorIndices) {
		return fmt.Errorf(
			"expected same number of public keys and validator indices, received %d != %d",
			len(publicKeys),
			len(validatorIndices),
		)
	}
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveValidatorIndices")
	defer span.End()
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorsBucket)
		var err error
		for i := 0; i < len(publicKeys); i++ {
			buf := uint64ToBytes(validatorIndices[i])
			k.validatorIndexCache.Set(string(publicKeys[i][:]), validatorIndices[i], int64(len(buf)))
			err = bucket.Put(publicKeys[i][:], buf)
		}
		return err
	})
}

func uint64ToBytes(i uint64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, i)
	return buf
}
