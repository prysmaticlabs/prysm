package kv

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// LastValidatedCheckpoint returns the latest fully validated checkpoint in beacon chain.
func (s *Store) LastValidatedCheckpoint(ctx context.Context) (*ethpb.Checkpoint, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.LastValidatedCheckpoint")
	defer span.End()
	var checkpoint *ethpb.Checkpoint
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(checkpointBucket)
		enc := bkt.Get(lastValidatedCheckpointKey)
		if enc == nil {
			var finErr error
			checkpoint, finErr = s.FinalizedCheckpoint(ctx)
			return finErr
		}
		checkpoint = &ethpb.Checkpoint{}
		return decode(ctx, enc, checkpoint)
	})
	return checkpoint, err
}

// SaveLastValidatedCheckpoint saves the last validated checkpoint in beacon chain.
func (s *Store) SaveLastValidatedCheckpoint(ctx context.Context, checkpoint *ethpb.Checkpoint) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveLastValidatedCheckpoint")
	defer span.End()

	enc, err := encode(ctx, checkpoint)
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(checkpointBucket)
		hasStateSummary := s.hasStateSummaryBytes(tx, bytesutil.ToBytes32(checkpoint.Root))
		hasStateInDB := tx.Bucket(stateBucket).Get(checkpoint.Root) != nil
		if !(hasStateInDB || hasStateSummary) {
			log.Warnf("Recovering state summary for last validated root: %#x", bytesutil.Trunc(checkpoint.Root))
			if err := recoverStateSummary(ctx, tx, checkpoint.Root); err != nil {
				return errors.Wrapf(errMissingStateForCheckpoint, "could not save finalized checkpoint, last validated root: %#x", bytesutil.Trunc(checkpoint.Root))
			}
		}
		return bucket.Put(lastValidatedCheckpointKey, enc)
	})
}
