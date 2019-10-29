package kv

import (
	"bytes"
	"context"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	dbpb "github.com/prysmaticlabs/prysm/proto/beacon/db"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

var errMissingParentBlockInDatabase = errors.New("missing block in database")

func updateFinalizedBlockRoots(ctx context.Context, tx *bolt.Tx, checkpoint *ethpb.Checkpoint) error {
	if !featureconfig.Get().EnableFinalizedBlockRootIndex {
		return nil
	}

	ctx, span := trace.StartSpan(ctx, "BeaconDB.updateFinalizedBlockRoots")
	defer span.End()

	bkt := tx.Bucket(finalizedBlockRootsIndexBucket)
	blocks := tx.Bucket(blocksBucket)

	root := checkpoint.Root
	var previousRoot []byte
	genesisRoot := blocks.Get(genesisBlockRootKey)

	// Walk up the ancestry chain until we reach a block root present in the finalized block roots
	// index bucket or genesis block root.
	var walk = func() error {
		for {
			if bytes.Equal(root, genesisRoot) {
				return nil
			}

			enc := blocks.Get(root)
			if enc == nil {
				traceutil.AnnotateError(span, errMissingParentBlockInDatabase)
				return errMissingParentBlockInDatabase
			}
			block := &ethpb.BeaconBlock{}
			if err := proto.Unmarshal(enc, block); err != nil {
				traceutil.AnnotateError(span, err)
				return err
			}

			container := &dbpb.FinalizedBlockRootContainer{
				ParentRoot: block.ParentRoot,
				ChildRoot:  previousRoot,
			}

			enc, err := proto.Marshal(container)
			if err != nil {
				traceutil.AnnotateError(span, err)
				return err
			}
			if err := bkt.Put(root, enc); err != nil {
				traceutil.AnnotateError(span, err)
				return err
			}
			if parentBytes := bkt.Get(block.ParentRoot); parentBytes != nil {
				parent := &dbpb.FinalizedBlockRootContainer{}
				if err := proto.Unmarshal(parentBytes, parent); err != nil {
					traceutil.AnnotateError(span, err)
					return err
				}
				parent.ChildRoot = root
				enc, err := proto.Marshal(parent)
				if err != nil {
					traceutil.AnnotateError(span, err)
					return err
				}
				return bkt.Put(block.ParentRoot, enc)
			}
			previousRoot = root
			root = block.ParentRoot
		}
	}
	err := walk()
	// All updates must have been successful or the whole transaction rolled back.
	if err != nil {
		tx.Rollback()
	}
	return err
}

// IsFinalizedBlock returns true if the block root is present in the finalized block root index.
func (kv *Store) IsFinalizedBlock(ctx context.Context, blockRoot [32]byte) bool {
	if !featureconfig.Get().EnableFinalizedBlockRootIndex {
		return true
	}

	ctx, span := trace.StartSpan(ctx, "BeaconDB.IsFinalizedBlock")
	defer span.End()

	var exists bool
	kv.db.View(func(tx *bolt.Tx) error {
		exists = tx.Bucket(finalizedBlockRootsIndexBucket).Get(blockRoot[:]) != nil
		return nil
	})
	return exists
}
