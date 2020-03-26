package kv

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// SaveArchivedPointState saves an archived point state to the DB. This is used for cold state management.
// An archive point index is `slot / slots_per_archive_point`.
func (k *Store) SaveArchivedPointState(ctx context.Context, state *state.BeaconState, index uint64) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveArchivedPointState")
	defer span.End()
	if state == nil {
		return errors.New("nil state")
	}
	enc, err := encode(state.InnerStateUnsafe())
	if err != nil {
		return err
	}

	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(archivedIndexStateBucket)
		return bucket.Put(uint64ToBytes(index), enc)
	})
}

// SaveArchivedPointRoot saves an archived point root to the DB. This is used for cold state management.
func (k *Store) SaveArchivedPointRoot(ctx context.Context, blockRoot [32]byte, index uint64) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveArchivedPointRoot")
	defer span.End()

	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(archivedIndexRootBucket)
		return bucket.Put(uint64ToBytes(index), blockRoot[:])
	})
}

// SaveLastArchivedIndex to the db.
func (k *Store) SaveLastArchivedIndex(ctx context.Context, index uint64) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveHeadBlockRoot")
	defer span.End()
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(archivedIndexRootBucket)
		return bucket.Put(lastArchivedIndexKey, uint64ToBytes(index))
	})
}

// LastArchivedIndexRoot from the db.
func (k *Store) LastArchivedIndexRoot(ctx context.Context) [32]byte {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.LastArchivedIndexRoot")
	defer span.End()

	var blockRoot []byte
	// #nosec G104. Always returns nil.
	k.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(archivedIndexRootBucket)
		lastArchivedIndex := bucket.Get(lastArchivedIndexKey)
		if lastArchivedIndex == nil {
			return nil
		}
		blockRoot = bucket.Get(lastArchivedIndex)
		return nil
	})

	return bytesutil.ToBytes32(blockRoot)
}

// LastArchivedIndexState from the db.
func (k *Store) LastArchivedIndexState(ctx context.Context) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.LastArchivedIndexState")
	defer span.End()

	var s *pb.BeaconState
	err := k.db.View(func(tx *bolt.Tx) error {
		indexRootBucket := tx.Bucket(archivedIndexRootBucket)
		lastArchivedIndex := indexRootBucket.Get(lastArchivedIndexKey)
		if lastArchivedIndex == nil {
			return nil
		}
		indexStateBucket := tx.Bucket(archivedIndexStateBucket)
		enc := indexStateBucket.Get(lastArchivedIndex)
		if enc == nil {
			return nil
		}

		var err error
		s, err = createState(enc)
		return err
	})
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, nil
	}
	return state.InitializeFromProtoUnsafe(s)
}

// ArchivedPointState returns the state of an archived point from the DB.
// This is essential for cold state management and to restore a cold state.
func (k *Store) ArchivedPointState(ctx context.Context, index uint64) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.ArchivedPointState")
	defer span.End()
	var s *pb.BeaconState
	err := k.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(archivedIndexStateBucket)
		enc := bucket.Get(uint64ToBytes(index))
		if enc == nil {
			return nil
		}

		var err error
		s, err = createState(enc)
		return err
	})
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, nil
	}
	return state.InitializeFromProtoUnsafe(s)
}

// ArchivedPointRoot returns the block root of an archived point from the DB.
// This is essential for cold state management and to restore a cold state.
func (k *Store) ArchivedPointRoot(ctx context.Context, index uint64) [32]byte {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.ArchivePointRoot")
	defer span.End()

	var blockRoot []byte
	// #nosec G104. Always returns nil.
	k.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(archivedIndexRootBucket)
		blockRoot = bucket.Get(uint64ToBytes(index))
		return nil
	})

	return bytesutil.ToBytes32(blockRoot)
}

// HasArchivedPoint returns true if an archived point exists in DB.
func (k *Store) HasArchivedPoint(ctx context.Context, index uint64) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HasArchivedPoint")
	defer span.End()
	var exists bool
	// #nosec G104. Always returns nil.
	k.db.View(func(tx *bolt.Tx) error {
		iBucket := tx.Bucket(archivedIndexRootBucket)
		exists = iBucket.Get(uint64ToBytes(index)) != nil
		return nil
	})
	return exists
}
