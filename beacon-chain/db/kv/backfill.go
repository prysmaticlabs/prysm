package kv

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/proto/dbval"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
	"google.golang.org/protobuf/proto"
)

func (s *Store) SaveBackfillStatus(ctx context.Context, bf *dbval.BackfillStatus) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveBackfillStatus")
	defer span.End()
	bfb, err := proto.Marshal(bf)
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blocksBucket)
		return bucket.Put(backfillStatusKey, bfb)
	})
}

func (s *Store) BackfillStatus(ctx context.Context) (*dbval.BackfillStatus, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveBackfillStatus")
	defer span.End()
	bf := &dbval.BackfillStatus{}
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blocksBucket)
		bs := bucket.Get(backfillStatusKey)
		if len(bs) == 0 {
			return errors.Wrap(ErrNotFound, "BackfillStatus not found")
		}
		return proto.Unmarshal(bs, bf)
	})
	return bf, err
}
