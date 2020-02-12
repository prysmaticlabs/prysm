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
func (k *Store) HotStateSummary(ctx context.Context, blockRoot []byte) (*pb.HotStateSummary, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HotStateSummary")
	defer span.End()

	var summary *pb.HotStateSummary
	err := k.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(hotStateSummaryBucket)
		enc := bucket.Get(blockRoot)
		if enc == nil {
			return nil
		}
		summary = &pb.HotStateSummary{}
		return decode(enc, summary)
	})

	return summary, err
}
