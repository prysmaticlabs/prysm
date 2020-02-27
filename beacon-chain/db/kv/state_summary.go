package kv

import (
	"context"

	"github.com/boltdb/bolt"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"go.opencensus.io/trace"
)

// SaveStateSummary saves a state summary to the DB.
func (k *Store) SaveStateSummary(ctx context.Context, summary *pb.StateSummary) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveStateSummary")
	defer span.End()

	enc, err := encode(summary)
	if err != nil {
		return err
	}
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(stateSummaryBucket)
		return bucket.Put(summary.Root, enc)
	})
}

// StateSummary returns the state summary using input block root from DB.
func (k *Store) StateSummary(ctx context.Context, blockRoot [32]byte) (*pb.StateSummary, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.StateSummary")
	defer span.End()

	var summary *pb.StateSummary
	err := k.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(stateSummaryBucket)
		enc := bucket.Get(blockRoot[:])
		if enc == nil {
			return nil
		}
		summary = &pb.StateSummary{}
		return decode(enc, summary)
	})

	return summary, err
}

// HasStateSummary returns true if a state summary exists in DB.
func (k *Store) HasStateSummary(ctx context.Context, blockRoot [32]byte) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HasStateSummary")
	defer span.End()
	var exists bool
	// #nosec G104. Always returns nil.
	k.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(stateSummaryBucket)
		exists = bucket.Get(blockRoot[:]) != nil
		return nil
	})
	return exists
}

// DeleteStateSummary deletes the state summary using input block root from DB.
func (k *Store) DeleteStateSummary(ctx context.Context, blockRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DeleteStateSummary")
	defer span.End()

	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(stateSummaryBucket)
		return bucket.Delete(blockRoot[:])
	})
}
