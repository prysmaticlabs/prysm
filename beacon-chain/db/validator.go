package db

import (
	"github.com/prysmaticlabs/prysm/bazel-prysm/external/go_sdk/src/fmt"
	"strconv"

	"github.com/boltdb/bolt"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

// SaveValidatorIndex accepts a public key and validator index and writes them to disk.
func (db *BeaconDB) SaveValidatorIndex(pubKey []byte, index int) error {
	h := hashutil.Hash(pubKey)

	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorBucket)

		return bucket.Put(h[:], []byte(strconv.Itoa(index)))
	})
}

// ValidatorIndex accepts a public key and returns the corresponding validator index.
// Returns nil if the block does not exist.
func (db *BeaconDB) ValidatorIndex(pubKey []byte) (int, error) {
	if !db.HasValidator(pubKey) {
		return -1, fmt.Errorf("validator %#x does not exist", pubKey)
	}

	var index int
	h := hashutil.Hash(pubKey)

	err := db.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorBucket)

		enc := bucket.Get(h[:])
		if enc == nil {
			return nil
		}
		var err error
		index, err = strconv.Atoi(string(enc))
		return err
	})

	return index, err
}

// DeleteValidatorIndex deletes the validator index record.
func (db *BeaconDB) DeleteValidatorIndex(pubKey []byte) error {
	h := hashutil.Hash(pubKey)

	return db.update(func(tx *bolt.Tx) error {
		a := tx.Bucket(validatorBucket)

		return a.Delete(h[:])
	})
}

// HasValidator checks if a validator exists.
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
