package kv

import (
	"context"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// LastArchivedSlot from the db.
func (kv *Store) LastArchivedSlot(ctx context.Context) (uint64, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.LastArchivedSlot")
	defer span.End()
	var index uint64
	err := kv.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(stateSlotIndicesBucket)
		b, _ := bkt.Cursor().Last()
		index = bytesutil.BytesToUint64BigEndian(b)
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
		bkt := tx.Bucket(stateSlotIndicesBucket)
		_, blockRoot = bkt.Cursor().Last()
		return nil
	}); err != nil { // This view never returns an error, but we'll handle anyway for sanity.
		panic(err)
	}

	return bytesutil.ToBytes32(blockRoot)
}

// ArchivedPointRoot returns the block root of an archived point from the DB.
// This is essential for cold state management and to restore a cold state.
func (kv *Store) ArchivedPointRoot(ctx context.Context, slot uint64) [32]byte {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.ArchivedPointRoot")
	defer span.End()

	var blockRoot []byte
	if err := kv.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(stateSlotIndicesBucket)
		blockRoot = bucket.Get(bytesutil.Uint64ToBytesBigEndian(slot))
		return nil
	}); err != nil { // This view never returns an error, but we'll handle anyway for sanity.
		panic(err)
	}

	return bytesutil.ToBytes32(blockRoot)
}

// HasArchivedPoint returns true if an archived point exists in DB.
func (kv *Store) HasArchivedPoint(ctx context.Context, slot uint64) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HasArchivedPoint")
	defer span.End()
	var exists bool
	if err := kv.db.View(func(tx *bolt.Tx) error {
		iBucket := tx.Bucket(stateSlotIndicesBucket)
		exists = iBucket.Get(bytesutil.Uint64ToBytesBigEndian(slot)) != nil
		return nil
	}); err != nil { // This view never returns an error, but we'll handle anyway for sanity.
		panic(err)
	}
	return exists
}
