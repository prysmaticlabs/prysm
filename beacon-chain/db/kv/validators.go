package kv

import (
	"context"
	"encoding/binary"
	"time"

	"github.com/boltdb/bolt"
	"go.opencensus.io/trace"
)

// ValidatorIndex by public key.
func (k *Store) ValidatorIndex(ctx context.Context, publicKey [48]byte) (uint64, bool, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.ValidatorIndex")
	defer span.End()
	// Return latest validatorIndex from cache if it exists.
	if v := k.validatorIndexCache.Get(string(publicKey[:])); v != nil && v.Value() != nil {
		return v.Value().(uint64), true, nil
	}

	var validatorIdx uint64
	var ok bool
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(validatorsBucket)
		enc := bkt.Get(publicKey[:])
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
func (k *Store) HasValidatorIndex(ctx context.Context, publicKey [48]byte) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HasValidatorIndex")
	defer span.End()
	if v := k.validatorIndexCache.Get(string(publicKey[:])); v != nil && v.Value() != nil {
		return true
	}
	exists := false
	// #nosec G104. Always returns nil.
	k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(validatorsBucket)
		exists = bkt.Get(publicKey[:]) != nil
		return nil
	})
	return exists
}

// DeleteValidatorIndex clears a validator index from the db by the validator's public key.
func (k *Store) DeleteValidatorIndex(ctx context.Context, publicKey [48]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DeleteValidatorIndex")
	defer span.End()
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorsBucket)
		k.validatorIndexCache.Delete(string(publicKey[:]))
		return bucket.Delete(publicKey[:])
	})
}

// SaveValidatorIndex by public key in the db.
func (k *Store) SaveValidatorIndex(ctx context.Context, publicKey [48]byte, validatorIdx uint64) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveValidatorIndex")
	defer span.End()
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorsBucket)
		buf := uint64ToBytes(validatorIdx)
		k.validatorIndexCache.Set(string(publicKey[:]), validatorIdx, time.Hour)
		return bucket.Put(publicKey[:], buf)
	})
}

func uint64ToBytes(i uint64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, i)
	return buf
}
