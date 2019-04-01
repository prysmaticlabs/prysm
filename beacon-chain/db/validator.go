package db

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/boltdb/bolt"
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
func (db *BeaconDB) ValidatorIndex(pubKey []byte) (uint64, error) {
	if !db.HasValidator(pubKey) {
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

// ValidatorIndices accepts a slice of public keys and returns the corresponding map of validators indexes and public keys.
func (db *BeaconDB) ValidatorIndices(pubKeys [][]byte) (map[uint64][]byte, error) {
	m := make(map[uint64][]byte)
	if !db.HasValidators(pubKeys) {
		return m, fmt.Errorf("one or more of the validators\n%#x\ndoes not exist", pubKeys)
	}

	err := db.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorBucket)
		for _, pk := range pubKeys {
			h := hashutil.Hash(pk)
			enc := bucket.Get(h[:])
			buf := bytes.NewBuffer(enc)
			index, err := binary.ReadUvarint(buf)
			if err != nil {
				return err
			}
			m[index] = pk
		}
		return nil
	})

	return m, err
}

// DeleteValidatorIndex deletes the validator index map record.
func (db *BeaconDB) DeleteValidatorIndex(pubKey []byte) error {
	h := hashutil.Hash(pubKey)

	return db.update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(validatorBucket)

		return bkt.Delete(h[:])
	})
}

// HasValidators checks if a validator index map exists out of a list of public keys.
func (db *BeaconDB) HasValidators(pubKeys [][]byte) bool {
	exists := false
	// #nosec G104, similar to HasBlock, HasAttestation... etc
	db.view(func(tx *bolt.Tx) error {
		a := tx.Bucket(validatorBucket)
		for _, pk := range pubKeys {
			h := hashutil.Hash(pk)
			exists = a.Get(h[:]) != nil
			if !exists {
				break
			}
		}
		return nil
	})

	return exists
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

// HasAllValidators returns true if all validators in a list of public keys
// are in the bucket.
func (db *BeaconDB) HasAllValidators(pubKeys [][]byte) bool {
	return db.hasValidators(pubKeys, true /* requireAll */)
}

// HasAnyValidators returns true if any validator in a list of public keys
// are in the bucket.
func (db *BeaconDB) HasAnyValidators(pubKeys [][]byte) bool {
	return db.hasValidators(pubKeys, false /* requireAll */)
}

func (db *BeaconDB) hasValidators(pubKeys [][]byte, requireAll bool) bool {
	exists := false
	// #nosec G104, similar to HasBlock, HasAttestation... etc
	db.view(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(validatorBucket)
		for _, pk := range pubKeys {
			h := hashutil.Hash(pk)
			exists = bkt.Get(h[:]) != nil
			if !exists && requireAll {
				break
			} else if exists && !requireAll {
				break
			}
		}
		return nil
	})

	return exists
}
