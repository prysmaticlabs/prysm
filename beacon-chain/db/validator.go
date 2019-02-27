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

// DeleteValidatorIndex deletes the validator index map record.
func (db *BeaconDB) DeleteValidatorIndex(pubKey []byte) error {
	h := hashutil.Hash(pubKey)

	return db.update(func(tx *bolt.Tx) error {
		a := tx.Bucket(validatorBucket)

		return a.Delete(h[:])
	})
}

// HasValidator checks if a validator index map exists.
func (db *BeaconDB) HasValidator(pubKey []byte) bool {
	exists := false
	h := hashutil.Hash(pubKey)
	// #nosec G104, similar to HasBlock, HasAttestation... etc
	db.view(func(tx *bolt.Tx) error {
		a := tx.Bucket(validatorBucket)

		exists = a.Get(h[:]) != nil
		return nil
	})
	return exists
}
