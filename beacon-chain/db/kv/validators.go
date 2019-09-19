package kv

import (
	"context"
	"encoding/binary"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"go.opencensus.io/trace"
)

// ValidatorLatestVote retrieval by validator index.
func (k *Store) ValidatorLatestVote(ctx context.Context, validatorIdx uint64) (*pb.ValidatorLatestVote, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.ValidatorLatestVote")
	defer span.End()

	// Return latest vote from cache if it exists.
	if v := k.votesCache.Get(string(validatorIdx)); v != nil {
		return v.Value().(*pb.ValidatorLatestVote), nil
	}

	buf := uint64ToBytes(validatorIdx)
	var latestVote *pb.ValidatorLatestVote
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(validatorsBucket)
		enc := bkt.Get(buf)
		if enc == nil {
			return nil
		}
		latestVote = &pb.ValidatorLatestVote{}
		return proto.Unmarshal(enc, latestVote)
	})
	return latestVote, err
}

// HasValidatorLatestVote verifies if a validator index has a latest vote stored in the db.
func (k *Store) HasValidatorLatestVote(ctx context.Context, validatorIdx uint64) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HasValidatorLatestVote")
	defer span.End()

	if v := k.votesCache.Get(string(validatorIdx)); v != nil {
		return true
	}

	buf := uint64ToBytes(validatorIdx)
	exists := false
	// #nosec G104. Always returns nil.
	k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(validatorsBucket)
		exists = bkt.Get(buf) != nil
		return nil
	})
	return exists
}

// SaveValidatorLatestVote by validator index.
func (k *Store) SaveValidatorLatestVote(ctx context.Context, validatorIdx uint64, vote *pb.ValidatorLatestVote) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveValidatorLatestVote")
	defer span.End()
	buf := uint64ToBytes(validatorIdx)
	enc, err := proto.Marshal(vote)
	if err != nil {
		return err
	}
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorsBucket)
		k.votesCache.Set(string(validatorIdx), vote, time.Hour)
		return bucket.Put(buf, enc)
	})
}

// DeleteValidatorLatestVote from the db.
func (k *Store) DeleteValidatorLatestVote(ctx context.Context, validatorIdx uint64) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DeleteValidatorLatestVote")
	defer span.End()
	return k.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(validatorsBucket)
		enc := bkt.Get(uint64ToBytes(validatorIdx))
		if enc == nil {
			return nil
		}
		k.votesCache.Delete(string(validatorIdx))
		return bkt.Delete(uint64ToBytes(validatorIdx))
	})
}

// ValidatorIndex by public key.
func (k *Store) ValidatorIndex(ctx context.Context, publicKey [48]byte) (uint64, bool, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.ValidatorIndex")
	defer span.End()
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
		return bucket.Put(publicKey[:], buf)
	})
}

func uint64ToBytes(i uint64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, i)
	return buf
}
