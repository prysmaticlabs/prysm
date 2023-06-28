package kv

import (
	"context"

	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// BlockRoot returns the block root from the DB using the state root
func (s *Store) BlockRoot(ctx context.Context, stateRoot [32]byte) ([32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.BlockRoot")
	defer span.End()

	var blockRoot [32]byte
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blockRootsBucket)
		blockRoot = [32]byte(bucket.Get(stateRoot[:]))
		return nil
	})

	return blockRoot, err
}

// SaveBlockRoot saves state root to block root mapping into DB
func (s *Store) SaveBlockRoot(ctx context.Context, stateRoot [32]byte, blockRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveBlockRoot")
	defer span.End()
	return s.SaveBlockRoots(ctx, [][32]byte{stateRoot}, [][32]byte{blockRoot})
}

// SaveBlockRoots bulk saves list of state root to block root mappings into DB
func (s *Store) SaveBlockRoots(ctx context.Context, stateRoots [][32]byte, blockRoots [][32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveBlockRoots")
	defer span.End()

	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blockRootsBucket)
		for i, stateRoot := range stateRoots {
			if err := bucket.Put(stateRoot[:], blockRoots[i][:]); err != nil {
				return err
			}
		}
		return nil
	})
}

// DeleteBlockRoot deletes block root from the DB using state root
func (s *Store) DeleteBlockRoot(ctx context.Context, stateRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DeleteBlockRoot")
	defer span.End()

	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blockRootsBucket)
		return bucket.Delete(stateRoot[:])
	})
}
