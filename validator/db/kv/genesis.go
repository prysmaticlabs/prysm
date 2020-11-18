package kv

import (
	"context"

	bolt "go.etcd.io/bbolt"
)

// SaveGenesisValidatorRoot saves the genesis validator root to db.
func (s *Store) SaveGenesisValidatorRoot(ctx context.Context, data []byte) error {
	err := s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(genesisInfoBucket)
		return bkt.Put(genesisValidatorsRootKey, data)
	})
	return err
}

// GenesisValidatorRoot retrieves the genesis validator root from db.
func (s *Store) GenesisValidatorRoot(ctx context.Context) ([]byte, error) {
	var data []byte
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(genesisInfoBucket)
		enc := bkt.Get(genesisValidatorsRootKey)
		if len(enc) == 0 {
			return nil
		}
		data = enc
		return nil
	})
	return data, err
}
