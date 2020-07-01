package kv

import (
	"bytes"
	"context"
	"fmt"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	dbpb "github.com/prysmaticlabs/prysm/proto/beacon/db"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

var previousFinalizedCheckpointKey = []byte("previous-finalized-checkpoint")

// Blocks from the recent finalized epoch are not part of the finalized and canonical chain in this
// index. These containers will be removed on the next update of finalized checkpoint. Note that
// these block roots may be considered canonical in the "head view" of the beacon chain, but not so
// in this index.
var containerFinalizedButNotCanonical = []byte("recent block needs reindexing to determine canonical")

// The finalized block roots index tracks beacon blocks which are finalized in the canonical chain.
// The finalized checkpoint contains the the epoch which was finalized and the highest beacon block
// root where block.slot <= start_slot(epoch). As a result, we cannot index the finalized canonical
// beacon block chain using the finalized root alone as this would exclude all other blocks in the
// finalized epoch from being indexed as "final and canonical".
//
// The algorithm for building the index works as follows:
//   - De-index all finalized beacon block roots from previous_finalized_epoch to
//     new_finalized_epoch. (I.e. delete these roots from the index, to be re-indexed.)
//   - Build the canonical finalized chain by walking up the ancestry chain from the finalized block
//     root until a parent is found in the index or the parent is genesis.
//   - Add all block roots in the database where epoch(block.slot) == checkpoint.epoch.
//
// This method ensures that all blocks from the current finalized epoch are considered "final" while
// maintaining only canonical and finalized blocks older than the current finalized epoch.
func (kv *Store) updateFinalizedBlockRoots(ctx context.Context, tx *bolt.Tx, checkpoint *ethpb.Checkpoint) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.updateFinalizedBlockRoots")
	defer span.End()

	bkt := tx.Bucket(finalizedBlockRootsIndexBucket)

	root := checkpoint.Root
	var previousRoot []byte
	genesisRoot := tx.Bucket(blocksBucket).Get(genesisBlockRootKey)

	// De-index recent finalized block roots, to be re-indexed.
	previousFinalizedCheckpoint := &ethpb.Checkpoint{}
	if b := bkt.Get(previousFinalizedCheckpointKey); b != nil {
		if err := decode(ctx, b, previousFinalizedCheckpoint); err != nil {
			traceutil.AnnotateError(span, err)
			return err
		}
	}

	blockRoots, err := kv.BlockRoots(ctx, filters.NewFilter().
		SetStartEpoch(previousFinalizedCheckpoint.Epoch).
		SetEndEpoch(checkpoint.Epoch+1),
	)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return err
	}
	for _, root := range blockRoots {
		if err := bkt.Delete(root[:]); err != nil {
			traceutil.AnnotateError(span, err)
			return err
		}
	}

	// Walk up the ancestry chain until we reach a block root present in the finalized block roots
	// index bucket or genesis block root.
	for {
		if bytes.Equal(root, genesisRoot) {
			break
		}

		signedBlock, err := kv.Block(ctx, bytesutil.ToBytes32(root))
		if err != nil {
			traceutil.AnnotateError(span, err)
			return err
		}
		if signedBlock == nil || signedBlock.Block == nil {
			err := fmt.Errorf("missing block in database: block root=%#x", root)
			traceutil.AnnotateError(span, err)
			return err
		}
		block := signedBlock.Block

		container := &dbpb.FinalizedBlockRootContainer{
			ParentRoot: block.ParentRoot,
			ChildRoot:  previousRoot,
		}

		enc, err := encode(ctx, container)
		if err != nil {
			traceutil.AnnotateError(span, err)
			return err
		}
		if err := bkt.Put(root, enc); err != nil {
			traceutil.AnnotateError(span, err)
			return err
		}

		// Found parent, loop exit condition.
		if parentBytes := bkt.Get(block.ParentRoot); parentBytes != nil {
			parent := &dbpb.FinalizedBlockRootContainer{}
			if err := decode(ctx, parentBytes, parent); err != nil {
				traceutil.AnnotateError(span, err)
				return err
			}
			parent.ChildRoot = root
			enc, err := encode(ctx, parent)
			if err != nil {
				traceutil.AnnotateError(span, err)
				return err
			}
			if err := bkt.Put(block.ParentRoot, enc); err != nil {
				traceutil.AnnotateError(span, err)
				return err
			}
			break
		}
		previousRoot = root
		root = block.ParentRoot
	}

	// Upsert blocks from the current finalized epoch.
	roots, err := kv.BlockRoots(ctx, filters.NewFilter().SetStartEpoch(checkpoint.Epoch).SetEndEpoch(checkpoint.Epoch+1))
	if err != nil {
		traceutil.AnnotateError(span, err)
		return err
	}
	for _, root := range roots {
		root := root[:]
		if bytes.Equal(root, checkpoint.Root) || bkt.Get(root) != nil {
			continue
		}
		if err := bkt.Put(root, containerFinalizedButNotCanonical); err != nil {
			traceutil.AnnotateError(span, err)
			return err
		}
	}

	// Update previous checkpoint
	enc, err := encode(ctx, checkpoint)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return err
	}

	return bkt.Put(previousFinalizedCheckpointKey, enc)
}

// IsFinalizedBlock returns true if the block root is present in the finalized block root index.
// A beacon block root contained exists in this index if it is considered finalized and canonical.
// Note: beacon blocks from the latest finalized epoch return true, whether or not they are
// considered canonical in the "head view" of the beacon node.
func (kv *Store) IsFinalizedBlock(ctx context.Context, blockRoot [32]byte) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.IsFinalizedBlock")
	defer span.End()

	var exists bool
	err := kv.db.View(func(tx *bolt.Tx) error {
		exists = tx.Bucket(finalizedBlockRootsIndexBucket).Get(blockRoot[:]) != nil
		// Check genesis block root.
		if !exists {
			genRoot := tx.Bucket(blocksBucket).Get(genesisBlockRootKey)
			exists = bytesutil.ToBytes32(genRoot) == blockRoot
		}
		return nil
	})
	if err != nil {
		traceutil.AnnotateError(span, err)
	}
	return exists
}
