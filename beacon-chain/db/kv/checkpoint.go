package kv

import (
	"context"
	"errors"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

var errMissingStateForFinalizedCheckpoint = errors.New("no state exists with checkpoint root")

// JustifiedCheckpoint returns the latest justified checkpoint in beacon chain.
func (k *Store) JustifiedCheckpoint(ctx context.Context) (*ethpb.Checkpoint, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.JustifiedCheckpoint")
	defer span.End()
	var checkpoint *ethpb.Checkpoint
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(checkpointBucket)
		enc := bkt.Get(justifiedCheckpointKey)
		if enc == nil {
			blockBucket := tx.Bucket(blocksBucket)
			genesisRoot := blockBucket.Get(genesisBlockRootKey)
			checkpoint = &ethpb.Checkpoint{Root: genesisRoot}
			return nil
		}
		checkpoint = &ethpb.Checkpoint{}
		return proto.Unmarshal(enc, checkpoint)
	})
	return checkpoint, err
}

// FinalizedCheckpoint returns the latest finalized checkpoint in beacon chain.
func (k *Store) FinalizedCheckpoint(ctx context.Context) (*ethpb.Checkpoint, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.FinalizedCheckpoint")
	defer span.End()
	var checkpoint *ethpb.Checkpoint
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(checkpointBucket)
		enc := bkt.Get(finalizedCheckpointKey)
		if enc == nil {
			blockBucket := tx.Bucket(blocksBucket)
			genesisRoot := blockBucket.Get(genesisBlockRootKey)
			checkpoint = &ethpb.Checkpoint{Root: genesisRoot}
			return nil
		}
		checkpoint = &ethpb.Checkpoint{}
		return proto.Unmarshal(enc, checkpoint)
	})
	return checkpoint, err
}

// SaveJustifiedCheckpoint saves justified checkpoint in beacon chain.
func (k *Store) SaveJustifiedCheckpoint(ctx context.Context, checkpoint *ethpb.Checkpoint) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveJustifiedCheckpoint")
	defer span.End()

	enc, err := proto.Marshal(checkpoint)
	if err != nil {
		return err
	}
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(checkpointBucket)
		return bucket.Put(justifiedCheckpointKey, enc)
	})
}

// SaveFinalizedCheckpoint saves finalized checkpoint in beacon chain.
func (k *Store) SaveFinalizedCheckpoint(ctx context.Context, checkpoint *ethpb.Checkpoint) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveFinalizedCheckpoint")
	defer span.End()

	enc, err := proto.Marshal(checkpoint)
	if err != nil {
		return err
	}
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(checkpointBucket)
		// The corresponding state must exist or there is a risk that the beacondb enters a state
		// where the finalized beaconState is missing. This would be a fatal condition requiring
		// a new sync from genesis.
		if tx.Bucket(stateBucket).Get(checkpoint.Root) == nil {
			traceutil.AnnotateError(span, errMissingStateForFinalizedCheckpoint)
			return errMissingStateForFinalizedCheckpoint
		}
		return bucket.Put(finalizedCheckpointKey, enc)
	})
}
