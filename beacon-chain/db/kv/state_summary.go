package kv

import (
	"context"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// SaveStateSummary saves a state summary object to the DB.
func (kv *Store) SaveStateSummary(ctx context.Context, summary *pb.StateSummary) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveStateSummary")
	defer span.End()

	enc, err := encode(summary)
	if err != nil {
		return err
	}
	kv.stateSummaryCache.Put(bytesutil.ToBytes32(summary.Root), summary)
	return kv.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(stateSummaryBucket)
		return bucket.Put(summary.Root, enc)
	})
}

// SaveStateSummaries saves state summary objects to the DB.
func (kv *Store) SaveStateSummaries(ctx context.Context, summaries []*pb.StateSummary) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveStateSummaries")
	defer span.End()
	for _, summary := range summaries {
		kv.stateSummaryCache.Put(bytesutil.ToBytes32(summary.Root), summary)
	}
	return kv.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(stateSummaryBucket)
		for _, summary := range summaries {
			enc, err := encode(summary)
			if err != nil {
				return err
			}
			if err := bucket.Put(summary.Root, enc); err != nil {
				return err
			}
		}
		return nil
	})
}

// StateSummary returns the state summary object from the db using input block root.
func (kv *Store) StateSummary(ctx context.Context, blockRoot [32]byte) (*pb.StateSummary, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.StateSummary")
	defer span.End()

	var summary *pb.StateSummary
	err := kv.db.View(func(tx *bolt.Tx) error {
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
func (kv *Store) HasStateSummary(ctx context.Context, blockRoot [32]byte) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HasStateSummary")
	defer span.End()
	var exists bool
	if err := kv.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(stateSummaryBucket)
		exists = bucket.Get(blockRoot[:]) != nil
		return nil
	}); err != nil { // This view never returns an error, but we'll handle anyway for sanity.
		panic(err)
	}
	return exists
}
