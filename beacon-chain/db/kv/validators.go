package kv

import (
	"bytes"
	"context"
	"encoding/binary"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// ValidatorLatestVote retrieval by validator index.
func (k *Store) ValidatorLatestVote(ctx context.Context, validatorIdx uint64) (*pb.ValidatorLatestVote, error) {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, validatorIdx)
	var latestVote *pb.ValidatorLatestVote
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(validatorsBucket)
		enc := bkt.Get(buf)
		if enc == nil {
			return nil
		}
		return proto.Unmarshal(enc, latestVote)
	})
	return latestVote, err
}

// HasValidatorLatestVote verifies if a validator index has a latest vote stored in the db.
func (k *Store) HasValidatorLatestVote(ctx context.Context, validatorIdx uint64) bool {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, validatorIdx)
	exists := false
	// #nosec G104, similar to HasBlock, HasAttestation... etc
	k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(validatorsBucket)
		exists = bkt.Get(buf) != nil
		return nil
	})
	return exists
}

// SaveValidatorLatestVote by validator index.
func (k *Store) SaveValidatorLatestVote(ctx context.Context, validatorIdx uint64, vote *pb.ValidatorLatestVote) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, validatorIdx)
	enc, err := proto.Marshal(vote)
	if err != nil {
		return err
	}
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorsBucket)
		return bucket.Put(buf, enc)
	})
}

// ValidatorIndex by public key.
func (k *Store) ValidatorIndex(ctx context.Context, publicKey [48]byte) (uint64, error) {
	var validatorIdx uint64
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(validatorsBucket)
		enc := bkt.Get(publicKey[:])
		if enc == nil {
			return nil
		}
		var err error
		buf := bytes.NewBuffer(enc)
		validatorIdx, err = binary.ReadUvarint(buf)
		return err
	})
	return validatorIdx, err
}

// HasValidatorIndex verifies if a validator's index by public key exists in the db.
func (k *Store) HasValidatorIndex(ctx context.Context, publicKey [48]byte) bool {
	exists := false
	// #nosec G104, similar to HasBlock, HasAttestation... etc
	k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(validatorsBucket)
		exists = bkt.Get(publicKey[:]) != nil
		return nil
	})
	return exists
}

// DeleteValidatorIndex clears a validator index from the db by the validator's public key.
func (k *Store) DeleteValidatorIndex(ctx context.Context, publicKey [48]byte) error {
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorsBucket)
		return bucket.Delete(publicKey[:])
	})
}

// SaveValidatorIndex by public key in the db.
func (k *Store) SaveValidatorIndex(ctx context.Context, publicKey [48]byte, validatorIdx uint64) error {
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorsBucket)
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, validatorIdx)
		return bucket.Put(publicKey[:], buf)
	})
}
