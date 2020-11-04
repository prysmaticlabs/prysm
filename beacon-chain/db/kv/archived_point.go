package kv

import (
	"bytes"
	"context"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// LastArchivedSlot from the db.
func (s *Store) LastArchivedSlot(ctx context.Context) (uint64, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.LastArchivedSlot")
	defer span.End()
	var index uint64
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(stateSlotIndicesBucket)
		b, _ := bkt.Cursor().Last()
		index = bytesutil.BytesToUint64BigEndian(b)
		return nil
	})

	return index, err
}

// LastArchivedRoot from the db.
func (s *Store) LastArchivedRoot(ctx context.Context) [32]byte {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.LastArchivedRoot")
	defer span.End()

	var blockRoot []byte
	if err := s.db.View(func(tx *bolt.Tx) error {
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
func (s *Store) ArchivedPointRoot(ctx context.Context, slot uint64) [32]byte {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.ArchivedPointRoot")
	defer span.End()

	var blockRoot []byte
	if err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(stateSlotIndicesBucket)
		blockRoot = bucket.Get(bytesutil.Uint64ToBytesBigEndian(slot))
		return nil
	}); err != nil { // This view never returns an error, but we'll handle anyway for sanity.
		panic(err)
	}

	return bytesutil.ToBytes32(blockRoot)
}

// HasArchivedPoint returns true if an archived point exists in DB.
func (s *Store) HasArchivedPoint(ctx context.Context, slot uint64) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HasArchivedPoint")
	defer span.End()
	var exists bool
	if err := s.db.View(func(tx *bolt.Tx) error {
		iBucket := tx.Bucket(stateSlotIndicesBucket)
		exists = iBucket.Get(bytesutil.Uint64ToBytesBigEndian(slot)) != nil
		return nil
	}); err != nil { // This view never returns an error, but we'll handle anyway for sanity.
		panic(err)
	}
	return exists
}

// CleanUpDirtyStates removes states in DB that falls to under archived point interval rules.
// Only following states would be kept:
// 1.) state_slot % archived_interval == 0. (e.g. archived_interval=2048, states with slot 2048, 4096... etc)
// 2.) archived_interval - slots_per_epoch/2 < state_slot % archived_interval
//   (e.g. archived_interval=2048, slots_per_epoch=32, states with slot 2047, 2046, 2032... etc).
//   This is to tolerate skip slots. Not every state lays on the boundary.
// 3.) state with current finalized root
func (s *Store) CleanUpDirtyStates(ctx context.Context, slotsPerArchivedPoint uint64) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.CleanUp")
	defer span.End()
	f, err := s.FinalizedCheckpoint(ctx)
	if err != nil {
		return err
	}
	deletedRoots := make([][32]byte, 0)

	err = s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(stateSlotIndicesBucket)
		if err := bkt.ForEach(func(k, v []byte) error {
			finalized := bytes.Equal(f.Root, v)
			slot := bytesutil.BytesToUint64BigEndian(k)
			mod := slot % slotsPerArchivedPoint
			// The following conditions cover 1, 2, and 3 above.
			if mod != 0 && mod < slotsPerArchivedPoint-params.BeaconConfig().SlotsPerEpoch/2 && !finalized {
				deletedRoots = append(deletedRoots, bytesutil.ToBytes32(v))
			}
			return nil
		}); err != nil {
			return err
		}
		return nil
	})

	log.WithField("count", len(deletedRoots)).Info("Cleaning up dirty states, this may take a while")
	if err := s.DeleteStates(ctx, deletedRoots); err != nil {
		return err
	}

	return err
}
