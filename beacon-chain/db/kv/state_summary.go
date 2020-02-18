package kv

import (
	"context"

	"github.com/boltdb/bolt"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"go.opencensus.io/trace"
)

// SaveHotStateSummary saves a hot state summary to the DB.
func (k *Store) SaveHotStateSummary(ctx context.Context, summary *pb.HotStateSummary) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveHotStateSummary")
	defer span.End()

	enc, err := encode(summary)
	if err != nil {
		return err
	}
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(hotStateSummaryBucket)
		return bucket.Put(summary.LatestRoot, enc)
	})
}

// HotStateSummary returns the hot state summary using input block root from DB.
func (k *Store) HotStateSummary(ctx context.Context, blockRoot [32]byte) (*pb.HotStateSummary, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HotStateSummary")
	defer span.End()

	var summary *pb.HotStateSummary
	err := k.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(hotStateSummaryBucket)
		enc := bucket.Get(blockRoot[:])
		if enc == nil {
			return nil
		}
		summary = &pb.HotStateSummary{}
		return decode(enc, summary)
	})

	return summary, err
}

// DeleteHotStateSummary deletes the hot state summary using input block root from DB.
func (k *Store) DeleteHotStateSummary(ctx context.Context, blockRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DeleteHotStateSummary")
	defer span.End()

	return k.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(hotStateSummaryBucket)
		return bucket.Delete(blockRoot[:])
	})
}

// SaveColdStateSummary saves a cold state summary to the DB.
func (k *Store) SaveColdStateSummary(ctx context.Context, blockRoot [32]byte, summary *pb.ColdStateSummary) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveColdStateSummary")
	defer span.End()

	enc, err := encode(summary)
	if err != nil {
		return err
	}
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(coldStateSummaryBucket)
		return bucket.Put(blockRoot[:], enc)
	})
}

// ColdStateSummary returns the cold state summary using input block root from DB.
func (k *Store) ColdStateSummary(ctx context.Context, blockRoot [32]byte) (*pb.ColdStateSummary, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.ColdStateSummary")
	defer span.End()

	var summary *pb.ColdStateSummary
	err := k.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(coldStateSummaryBucket)
		enc := bucket.Get(blockRoot[:])
		if enc == nil {
			return nil
		}
		summary = &pb.ColdStateSummary{}
		return decode(enc, summary)
	})

	return summary, err
}

// HasColdStateSummary returns true if a cold state summary exists in DB.
func (k *Store) HasColdStateSummary(ctx context.Context, blockRoot [32]byte) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HasColdStateSummary")
	defer span.End()
	var exists bool
	// #nosec G104. Always returns nil.
	k.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(coldStateSummaryBucket)
		exists = bucket.Get(blockRoot[:]) != nil
		return nil
	})
	return exists
}
