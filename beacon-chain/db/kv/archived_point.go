package kv

import (
	"context"
	"encoding/binary"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// SaveArchivedPointRoot saves an archived point root to the DB. This is used for cold state management.
func (kv *Store) SaveArchivedPointRoot(ctx context.Context, blockRoot [32]byte, slot uint64) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveArchivedPointRoot")
	defer span.End()

	return kv.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(archivedIndexRootBucket)
		if err := bucket.Put(bytesutil.Uint64ToBytes(slot), blockRoot[:]); err != nil {
			return err
		}
		// Update last archived state, if this state is higher than the the previous.
		last := bucket.Get(lastArchivedIndexKey)
		if slot > bytesutil.BytesToUint64(last) {
			return bucket.Put(lastArchivedIndexKey, bytesutil.Uint64ToBytes(slot))
		}
		return nil
	})
}

// LastArchivedSlot from the db.
func (kv *Store) LastArchivedSlot(ctx context.Context) (uint64, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.LastArchivedSlot")
	defer span.End()
	var index uint64
	err := kv.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(archivedIndexRootBucket)
		b := bucket.Get(lastArchivedIndexKey)
		if b == nil {
			return nil
		}
		index = binary.LittleEndian.Uint64(b)
		return nil
	})

	return index, err
}

// LastArchivedRoot from the db.
func (kv *Store) LastArchivedRoot(ctx context.Context) [32]byte {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.LastArchivedRoot")
	defer span.End()

	var blockRoot []byte
	if err := kv.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(archivedIndexRootBucket)
		lastArchivedIndex := bucket.Get(lastArchivedIndexKey)
		if lastArchivedIndex == nil {
			return nil
		}
		blockRoot = bucket.Get(lastArchivedIndex)
		return nil
	}); err != nil { // This view never returns an error, but we'll handle anyway for sanity.
		panic(err)
	}

	return bytesutil.ToBytes32(blockRoot)
}

// ArchivedPointRoot returns the block root of an archived point from the DB.
// This is essential for cold state management and to restore a cold state.
func (kv *Store) ArchivedPointRoot(ctx context.Context, index uint64) [32]byte {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.ArchivedPointRoot")
	defer span.End()

	var blockRoot []byte
	if err := kv.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(archivedIndexRootBucket)
		blockRoot = bucket.Get(bytesutil.Uint64ToBytes(index))
		return nil
	}); err != nil { // This view never returns an error, but we'll handle anyway for sanity.
		panic(err)
	}

	return bytesutil.ToBytes32(blockRoot)
}

// HasArchivedPoint returns true if an archived point exists in DB.
func (kv *Store) HasArchivedPoint(ctx context.Context, index uint64) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HasArchivedPoint")
	defer span.End()
	var exists bool
	if err := kv.db.View(func(tx *bolt.Tx) error {
		iBucket := tx.Bucket(archivedIndexRootBucket)
		exists = iBucket.Get(bytesutil.Uint64ToBytes(index)) != nil
		return nil
	}); err != nil { // This view never returns an error, but we'll handle anyway for sanity.
		panic(err)
	}
	return exists
}
