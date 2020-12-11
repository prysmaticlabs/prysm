package kv

import (
	"context"
	"errors"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// SaveStateSummaryInCache saves a state summary object to the state summary cache.
func (s *Store) SaveStateSummaryInCache(ctx context.Context, summary *pb.StateSummary) {
	s.stateSummaryCache.Put(bytesutil.ToBytes32(summary.Root), summary)
}

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

// SaveStateSummariesFromCacheToDB saves state summary objects from cache to the DB.
func (s *Store) SaveStateSummariesFromCacheToDB(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveStateSummaries")
	defer span.End()

	summaries := s.stateSummaryCache.GetAll()

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

// HasStateSummary returns true if a state summary exists in DB or cache.
func (s *Store) HasStateSummary(ctx context.Context, blockRoot [32]byte) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HasStateSummary")
	defer span.End()

	if s.stateSummaryCache.Has(blockRoot) {
		return true
	}

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

// RecoverStateSummary recovers state summary object of a given block root by using the saved block in DB.
func (s *Store) RecoverStateSummary(ctx context.Context, blockRoot [32]byte) (*pb.StateSummary, error) {
	if s.HasBlock(ctx, blockRoot) {
		b, err := s.Block(ctx, blockRoot)
		if err != nil {
			return nil, err
		}
		summary := &pb.StateSummary{Slot: b.Block.Slot, Root: blockRoot[:]}
		if err := s.SaveStateSummary(ctx, summary); err != nil {
			return nil, err
		}
		return summary, nil
	}
	return nil, errors.New("could not find block in DB")
}
