package kv

import (
	"context"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// SaveStateSummary saves a state summary object to the DB.
func (s *Store) SaveStateSummary(ctx context.Context, summary *pb.StateSummary) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveStateSummary")
	defer span.End()

	return s.SaveStateSummaries(ctx, []*pb.StateSummary{summary})
}

// SaveStateSummaries saves state summary objects to the DB.
func (s *Store) SaveStateSummaries(ctx context.Context, summaries []*pb.StateSummary) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveStateSummaries")
	defer span.End()

	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(stateSummaryBucket)
		for _, summary := range summaries {
			enc, err := encode(ctx, summary)
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
func (s *Store) StateSummary(ctx context.Context, blockRoot [32]byte) (*pb.StateSummary, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.StateSummary")
	defer span.End()
	enc, err := s.stateSummaryBytes(ctx, blockRoot)
	if err != nil {
		return nil, err
	}
	if len(enc) == 0 {
		return nil, nil
	}
	summary := &pb.StateSummary{}
	if err := decode(ctx, enc, summary); err != nil {
		return nil, err
	}
	return summary, nil
}

// HasStateSummary returns true if a state summary exists in DB.
func (s *Store) HasStateSummary(ctx context.Context, blockRoot [32]byte) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HasStateSummary")
	defer span.End()
	enc, err := s.stateSummaryBytes(ctx, blockRoot)
	if err != nil {
		panic(err)
	}
	return len(enc) > 0
}

func (s *Store) stateSummaryBytes(ctx context.Context, blockRoot [32]byte) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.stateSummaryBytes")
	defer span.End()

	var enc []byte
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(stateSummaryBucket)
		enc = bucket.Get(blockRoot[:])
		return nil
	})

	return enc, err
}
