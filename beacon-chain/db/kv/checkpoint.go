package kv

import (
	"context"
	"errors"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
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
			blockBucket := tx.Bucket(blocksBucket)
			genesisRoot := blockBucket.Get(genesisBlockRootKey)
			checkpoint = &ethpb.Checkpoint{Root: genesisRoot}
			return nil
		}
		checkpoint = &ethpb.Checkpoint{}
		return decode(enc, checkpoint)
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
			blockBucket := tx.Bucket(blocksBucket)
			genesisRoot := blockBucket.Get(genesisBlockRootKey)
			checkpoint = &ethpb.Checkpoint{Root: genesisRoot}
			return nil
		}
		checkpoint = &ethpb.Checkpoint{}
		return decode(enc, checkpoint)
	})
	return checkpoint, err
}

// SaveJustifiedCheckpoint saves justified checkpoint in beacon chain.
func (kv *Store) SaveJustifiedCheckpoint(ctx context.Context, checkpoint *ethpb.Checkpoint) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveJustifiedCheckpoint")
	defer span.End()

	enc, err := encode(checkpoint)
	if err != nil {
		return err
	}
	return kv.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(checkpointBucket)
		if featureconfig.Get().NewStateMgmt {
			hasStateSummaryInDB := tx.Bucket(stateSummaryBucket).Get(checkpoint.Root) != nil
			hasStateSummaryInCache := kv.stateSummaryCache.Has(bytesutil.ToBytes32(checkpoint.Root))
			hasStateInDB := tx.Bucket(stateBucket).Get(checkpoint.Root) != nil
			if !(hasStateInDB || hasStateSummaryInDB || hasStateSummaryInCache) {
				return errors.New("missing state summary for finalized root")
			}
		} else {
			// The corresponding state must exist or there is a risk that the beacondb enters a state
			// where the justified beaconState is missing. This may be a fatal condition requiring
			// a new sync from genesis.
			if tx.Bucket(stateBucket).Get(checkpoint.Root) == nil {
				traceutil.AnnotateError(span, errMissingStateForCheckpoint)
				return errMissingStateForCheckpoint
			}
		}
		return bucket.Put(justifiedCheckpointKey, enc)
	})
}

// SaveFinalizedCheckpoint saves finalized checkpoint in beacon chain.
func (kv *Store) SaveFinalizedCheckpoint(ctx context.Context, checkpoint *ethpb.Checkpoint) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveFinalizedCheckpoint")
	defer span.End()

	enc, err := encode(checkpoint)
	if err != nil {
		return err
	}
	return kv.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(checkpointBucket)
		if featureconfig.Get().NewStateMgmt {
			hasStateSummaryInDB := tx.Bucket(stateSummaryBucket).Get(checkpoint.Root) != nil
			hasStateSummaryInCache := kv.stateSummaryCache.Has(bytesutil.ToBytes32(checkpoint.Root))
			hasStateInDB := tx.Bucket(stateBucket).Get(checkpoint.Root) != nil
			if !(hasStateInDB || hasStateSummaryInDB || hasStateSummaryInCache) {
				return errors.New("missing state summary for finalized root")
			}
		} else {
			// The corresponding state must exist or there is a risk that the beacondb enters a state
			// where the finalized beaconState is missing. This would be a fatal condition requiring
			// a new sync from genesis.
			if tx.Bucket(stateBucket).Get(checkpoint.Root) == nil {
				traceutil.AnnotateError(span, errMissingStateForCheckpoint)
				return errMissingStateForCheckpoint
			}
		}

		if err := bucket.Put(finalizedCheckpointKey, enc); err != nil {
			return err
		}

		return kv.updateFinalizedBlockRoots(ctx, tx, checkpoint)
	})
}
