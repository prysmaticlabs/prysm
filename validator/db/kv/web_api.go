package kv

import (
	"context"

	bolt "go.etcd.io/bbolt"
)

// SaveHashedPasswordForAPI --
func (store *Store) SaveHashedPasswordForAPI(ctx context.Context, hashedPassword []byte) error {
	return store.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorAPIBucket)
		return bucket.Put(apiHashedPasswordKey, hashedPassword)
	})
}

// HashedPasswordForAPI --
func (store *Store) HashedPasswordForAPI(ctx context.Context) ([]byte, error) {
	var err error
	var hashedPassword []byte
	err = store.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorAPIBucket)
		hashedPassword = bucket.Get(apiHashedPasswordKey)
		return nil
	})
	return hashedPassword, err
}
