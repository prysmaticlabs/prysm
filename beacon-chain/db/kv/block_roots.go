package kv

import (
	"context"

	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// BlockRoot returns the block root from the DB using the state root
func (s *Store) BlockRoot(ctx context.Context, stateRoot [32]byte) ([32]byte, error) {
	_, span := trace.StartSpan(ctx, "BeaconDB.BlockRoot")
	defer span.End()

	var blockRoot [32]byte
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blockRootsBucket)
		root := bucket.Get(stateRoot[:])
		if root == nil {
			return nil
		}
		blockRoot = [32]byte(root)
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
	_, span := trace.StartSpan(ctx, "BeaconDB.SaveBlockRoots")
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

// DeleteBlockRoot deletes block root from the DB using derived state root
func (s *Store) DeleteBlockRoot(ctx context.Context, blockRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DeleteBlockRoot")
	defer span.End()

	return s.db.Update(func(tx *bolt.Tx) error {
		if err := checkJustifiedAndFinalized(ctx, tx, blockRoot); err != nil {
			return err
		}
		st, err := s.State(ctx, blockRoot)
		if err != nil {
			return err
		}
		if st != nil {
			stRoot, err := st.HashTreeRoot(ctx)
			if err != nil {
				return errors.Wrap(err, "could not get stateRoot")
			}
			bucket := tx.Bucket(blockRootsBucket)
			return bucket.Delete(stRoot[:])
		}
		return nil
	})
}
