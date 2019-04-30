package db

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"github.com/boltdb/bolt"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

// SaveValidatorIndex accepts a public key and validator index and writes them to disk.
func (db *BeaconDB) SaveValidatorIndex(pubKey []byte, index int) error {
	h := hashutil.Hash(pubKey)

	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorBucket)

		buf := make([]byte, binary.MaxVarintLen64)
		n := binary.PutUvarint(buf, uint64(index))

		return bucket.Put(h[:], buf[:n])
	})
}

// SaveValidatorIndexBatch accepts a public key and validator index and writes them to disk.
func (db *BeaconDB) SaveValidatorIndexBatch(pubKey []byte, index int) error {
	h := hashutil.Hash(pubKey)

	return db.batch(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorBucket)
		buf := make([]byte, binary.MaxVarintLen64)
		n := binary.PutUvarint(buf, uint64(index))
		return bucket.Put(h[:], buf[:n])
	})

}

// ValidatorIndex accepts a public key and returns the corresponding validator index.
// If the validator index is not found in DB, as a fail over, it searches the state and
// saves it to the DB when found.
func (db *BeaconDB) ValidatorIndex(pubKey []byte) (uint64, error) {
	if !db.HasValidator(pubKey) {
		state, err := db.HeadState(context.Background())
		if err != nil {
			return 0, err
		}
		for i := 0; i < len(state.ValidatorRegistry); i++ {
			v := state.ValidatorRegistry[i]
			if bytes.Equal(v.Pubkey, pubKey) {
				if err := db.SaveValidatorIndex(pubKey, i); err != nil {
					return 0, err
				}
				return uint64(i), nil
			}
		}
		return 0, fmt.Errorf("validator %#x does not exist", pubKey)
	}

	var index uint64
	h := hashutil.Hash(pubKey)
	err := db.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorBucket)

		enc := bucket.Get(h[:])
		if enc == nil {
			return nil
		}
		var err error
		buf := bytes.NewBuffer(enc)
		index, err = binary.ReadUvarint(buf)
		return err
	})

	return index, err
}

// DeleteValidatorIndex deletes the validator index map record.
func (db *BeaconDB) DeleteValidatorIndex(pubKey []byte) error {
	h := hashutil.Hash(pubKey)

	return db.update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(validatorBucket)

		return bkt.Delete(h[:])
	})
}

// HasValidator checks if a validator index map exists.
func (db *BeaconDB) HasValidator(pubKey []byte) bool {
	exists := false
	h := hashutil.Hash(pubKey)
	// #nosec G104, similar to HasBlock, HasAttestation... etc
	db.view(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(validatorBucket)

		exists = bkt.Get(h[:]) != nil
		return nil
	})
	return exists
}

// HasAnyValidators returns true if any validator in a list of public keys
// are in the bucket.
func (db *BeaconDB) HasAnyValidators(state *pb.BeaconState, pubKeys [][]byte) (bool, error) {
	exists := false
	// #nosec G104, similar to HasBlock, HasAttestation... etc
	db.view(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(validatorBucket)
		for _, pk := range pubKeys {
			h := hashutil.Hash(pk)
			exists = bkt.Get(h[:]) != nil
			break
		}
		return nil
	})

	if !exists {
		for _, pubKey := range pubKeys {
			for i := 0; i < len(state.ValidatorRegistry); i++ {
				v := state.ValidatorRegistry[i]
				if bytes.Equal(v.Pubkey, pubKey) {
					if err := db.SaveValidatorIndex(pubKey, i); err != nil {
						return false, err
					}
					exists = true
				}
			}
		}
	}
	return exists, nil
}
