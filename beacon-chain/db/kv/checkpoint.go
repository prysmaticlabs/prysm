package kv

import (
	"context"
	"errors"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

var errMissingStateForCheckpoint = errors.New("missing state summary for finalized root")

// JustifiedCheckpoint returns the latest justified checkpoint in beacon chain.
func (kv *Store) JustifiedCheckpoint(ctx context.Context) (*ethpb.Checkpoint, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.JustifiedCheckpoint")
	defer span.End()
	var checkpoint *ethpb.Checkpoint
	err := kv.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(checkpointBucket)
		enc := bkt.Get(justifiedCheckpointKey)
		if enc == nil {
			checkpoint = &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
			return nil
		}
		checkpoint = &ethpb.Checkpoint{}
		return decode(ctx, enc, checkpoint)
	})
	return checkpoint, err
}

// FinalizedCheckpoint returns the latest finalized checkpoint in beacon chain.
func (kv *Store) FinalizedCheckpoint(ctx context.Context) (*ethpb.Checkpoint, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.FinalizedCheckpoint")
	defer span.End()
	var checkpoint *ethpb.Checkpoint
	err := kv.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(checkpointBucket)
		enc := bkt.Get(finalizedCheckpointKey)
		if enc == nil {
			checkpoint = &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
			return nil
		}
		checkpoint = &ethpb.Checkpoint{}
		return decode(ctx, enc, checkpoint)
	})
	return checkpoint, err
}

// SaveJustifiedCheckpoint saves justified checkpoint in beacon chain.
func (kv *Store) SaveJustifiedCheckpoint(ctx context.Context, checkpoint *ethpb.Checkpoint) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveJustifiedCheckpoint")
	defer span.End()

	enc, err := encode(ctx, checkpoint)
	if err != nil {
		return err
	}
	return kv.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(checkpointBucket)
		hasStateSummaryInDB := tx.Bucket(stateSummaryBucket).Get(checkpoint.Root) != nil
		hasStateSummaryInCache := kv.stateSummaryCache.Has(bytesutil.ToBytes32(checkpoint.Root))
		hasStateInDB := tx.Bucket(stateBucket).Get(checkpoint.Root) != nil
		if !(hasStateInDB || hasStateSummaryInDB || hasStateSummaryInCache) {
			return errMissingStateForCheckpoint
		}
		return bucket.Put(justifiedCheckpointKey, enc)
	})
}

// SaveFinalizedCheckpoint saves finalized checkpoint in beacon chain.
func (kv *Store) SaveFinalizedCheckpoint(ctx context.Context, checkpoint *ethpb.Checkpoint) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveFinalizedCheckpoint")
	defer span.End()

	enc, err := encode(ctx, checkpoint)
	if err != nil {
		return err
	}
	return kv.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(checkpointBucket)
		hasStateSummaryInDB := tx.Bucket(stateSummaryBucket).Get(checkpoint.Root) != nil
		hasStateSummaryInCache := kv.stateSummaryCache.Has(bytesutil.ToBytes32(checkpoint.Root))
		hasStateInDB := tx.Bucket(stateBucket).Get(checkpoint.Root) != nil
		if !(hasStateInDB || hasStateSummaryInDB || hasStateSummaryInCache) {
			return errMissingStateForCheckpoint
		}
		if err := bucket.Put(finalizedCheckpointKey, enc); err != nil {
			return err
		}

		return kv.updateFinalizedBlockRoots(ctx, tx, checkpoint)
	})
}
