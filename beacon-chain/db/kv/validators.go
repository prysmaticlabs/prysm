package kv

import (
	"context"
	"encoding/binary"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// ValidatorLatestVote retrieval by validator index.
// TODO(#3164): Implement.
func (k *Store) ValidatorLatestVote(ctx context.Context, validatorIdx uint64) (*pb.ValidatorLatestVote, error) {
	return nil, nil
}

// HasValidatorLatestVote verifies if a validator index has a latest vote stored in the db.
// TODO(#3164): Implement.
func (k *Store) HasValidatorLatestVote(ctx context.Context, validatorIdx uint64) bool {
	return false
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
// TODO(#3164): Implement.
func (k *Store) ValidatorIndex(ctx context.Context, publicKey [48]byte) (uint64, error) {
	return 0, nil
}

// HasValidatorIndex verifies if a validator's index by public key exists in the db.
// TODO(#3164): Implement.
func (k *Store) HasValidatorIndex(ctx context.Context, publicKey [48]byte) bool {
	return false
}

// DeleteValidatorIndex clears a validator index from the db by the validator's public key.
// TODO(#3164): Implement.
func (k *Store) DeleteValidatorIndex(ctx context.Context, publicKey [48]byte) error {
	return nil
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
