package kv

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
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
// The finalized checkpoint contains the epoch which was finalized and the highest beacon block
// root where block.slot <= start_slot(epoch). As a result, we cannot index the finalized canonical
// beacon block chain using the finalized root alone as this would exclude all other blocks in the
// finalized epoch from being indexed as "final and canonical".
//
// The main part of the algorithm traverses parent->child block relationships in the
// `blockParentRootIndicesBucket` bucket to find the path between the last finalized checkpoint
// and the current finalized checkpoint. It relies on the invariant that there is a unique path
// between two finalized checkpoints.
func (s *Store) updateFinalizedBlockRoots(ctx context.Context, tx *bolt.Tx, checkpoint *ethpb.Checkpoint) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.updateFinalizedBlockRoots")
	defer span.End()

	finalizedBkt := tx.Bucket(finalizedBlockRootsIndexBucket)
	previousFinalizedCheckpoint := &ethpb.Checkpoint{}
	if b := finalizedBkt.Get(previousFinalizedCheckpointKey); b != nil {
		if err := decode(ctx, b, previousFinalizedCheckpoint); err != nil {
			tracing.AnnotateError(span, err)
			return err
		}
	}

	// Handle the case of checkpoint sync.
	if previousFinalizedCheckpoint.Root == nil && bytes.Equal(checkpoint.Root, tx.Bucket(blocksBucket).Get(originCheckpointBlockRootKey)) {
		container := &ethpb.FinalizedBlockRootContainer{}
		enc, err := encode(ctx, container)
		if err != nil {
			tracing.AnnotateError(span, err)
			return err
		}
		if err = finalizedBkt.Put(checkpoint.Root, enc); err != nil {
			tracing.AnnotateError(span, err)
			return err
		}
		return updatePrevFinalizedCheckpoint(ctx, span, finalizedBkt, checkpoint)
	}

	var finalized [][]byte
	if previousFinalizedCheckpoint.Root == nil {
		genesisRoot := tx.Bucket(blocksBucket).Get(genesisBlockRootKey)
		_, finalized = pathToFinalizedCheckpoint(ctx, [][]byte{genesisRoot}, checkpoint.Root, tx)
	} else {
		if err := updateChildOfPrevFinalizedCheckpoint(
			ctx,
			span,
			finalizedBkt,
			tx.Bucket(blockParentRootIndicesBucket), previousFinalizedCheckpoint.Root,
		); err != nil {
			return err
		}
		_, finalized = pathToFinalizedCheckpoint(ctx, [][]byte{previousFinalizedCheckpoint.Root}, checkpoint.Root, tx)
	}

	for i, r := range finalized {
		var container *ethpb.FinalizedBlockRootContainer
		switch i {
		case 0:
			container = &ethpb.FinalizedBlockRootContainer{
				ParentRoot: previousFinalizedCheckpoint.Root,
			}
			if len(finalized) > 1 {
				container.ChildRoot = finalized[i+1]
			}
		case len(finalized) - 1:
			// We don't know the finalized child of the new finalized checkpoint.
			// It will be filled out in the next function call.
			container = &ethpb.FinalizedBlockRootContainer{}
			if len(finalized) > 1 {
				container.ParentRoot = finalized[i-1]
			}
		default:
			container = &ethpb.FinalizedBlockRootContainer{
				ParentRoot: finalized[i-1],
				ChildRoot:  finalized[i+1],
			}
		}

		enc, err := encode(ctx, container)
		if err != nil {
			tracing.AnnotateError(span, err)
			return err
		}
		if err = finalizedBkt.Put(r, enc); err != nil {
			tracing.AnnotateError(span, err)
			return err
		}
	}

	return updatePrevFinalizedCheckpoint(ctx, span, finalizedBkt, checkpoint)
}

// BackfillFinalizedIndex updates the finalized index for a contiguous chain of blocks that are the ancestors of the
// given finalized child root. This is needed to update the finalized index during backfill, because the usual
// updateFinalizedBlockRoots has assumptions that are incompatible with backfill processing.
func (s *Store) BackfillFinalizedIndex(ctx context.Context, blocks []blocks.ROBlock, finalizedChildRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.BackfillFinalizedIndex")
	defer span.End()
	if len(blocks) == 0 {
		return errEmptyBlockSlice
	}

	fbrs := make([]*ethpb.FinalizedBlockRootContainer, len(blocks))
	encs := make([][]byte, len(blocks))
	for i := range blocks {
		pr := blocks[i].Block().ParentRoot()
		fbrs[i] = &ethpb.FinalizedBlockRootContainer{
			ParentRoot: pr[:],
			// ChildRoot: will be filled in on the next iteration when we look at the descendent block.
		}
		if i == 0 {
			continue
		}
		if blocks[i-1].Root() != blocks[i].Block().ParentRoot() {
			return errors.Wrapf(errIncorrectBlockParent, "previous root=%#x, slot=%d; child parent_root=%#x, root=%#x, slot=%d",
				blocks[i-1].Root(), blocks[i-1].Block().Slot(), blocks[i].Block().ParentRoot(), blocks[i].Root(), blocks[i].Block().Slot())
		}

		// We know the previous index is the parent of this one thanks to the assertion above,
		// so we can set the ChildRoot of the previous value to the root of the current value.
		fbrs[i-1].ChildRoot = blocks[i].RootSlice()
		// Now that the value for fbrs[i-1] is complete, perform encoding here to minimize time in Update,
		// which holds the global db lock.
		penc, err := encode(ctx, fbrs[i-1])
		if err != nil {
			tracing.AnnotateError(span, err)
			return err
		}
		encs[i-1] = penc
	}

	// The final element is the parent of finalizedChildRoot. This is checked inside the db transaction using
	// the parent_root value stored in the index data for finalizedChildRoot.
	lastIdx := len(blocks) - 1
	fbrs[lastIdx].ChildRoot = finalizedChildRoot[:]
	// Final element is complete, so it is pre-encoded like the others.
	enc, err := encode(ctx, fbrs[lastIdx])
	if err != nil {
		tracing.AnnotateError(span, err)
		return err
	}
	encs[lastIdx] = enc

	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(finalizedBlockRootsIndexBucket)
		child := bkt.Get(finalizedChildRoot[:])
		if len(child) == 0 {
			return errFinalizedChildNotFound
		}
		fcc := &ethpb.FinalizedBlockRootContainer{}
		if err := decode(ctx, child, fcc); err != nil {
			return errors.Wrapf(err, "unable to decode finalized block root container for root=%#x", finalizedChildRoot)
		}
		// Ensure that the existing finalized chain descends from the new segment.
		if !bytes.Equal(fcc.ParentRoot, blocks[len(blocks)-1].RootSlice()) {
			return errors.Wrapf(errNotConnectedToFinalized, "finalized block root container for root=%#x has parent_root=%#x, not %#x",
				finalizedChildRoot, fcc.ParentRoot, blocks[len(blocks)-1].RootSlice())
		}
		// Update the finalized index with entries for each block in the new segment.
		for i := range fbrs {
			if err := bkt.Put(blocks[i].RootSlice(), encs[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

// IsFinalizedBlock returns true if the block root is present in the finalized block root index.
// A beacon block root contained exists in this index if it is considered finalized and canonical.
func (s *Store) IsFinalizedBlock(ctx context.Context, blockRoot [32]byte) bool {
	_, span := trace.StartSpan(ctx, "BeaconDB.IsFinalizedBlock")
	defer span.End()

	var exists bool
	err := s.db.View(func(tx *bolt.Tx) error {
		exists = tx.Bucket(finalizedBlockRootsIndexBucket).Get(blockRoot[:]) != nil
		// Check genesis block root.
		if !exists {
			genRoot := tx.Bucket(blocksBucket).Get(genesisBlockRootKey)
			exists = bytesutil.ToBytes32(genRoot) == blockRoot
		}
		return nil
	})
	if err != nil {
		tracing.AnnotateError(span, err)
	}
	return exists
}

// FinalizedChildBlock returns the child block of a provided finalized block. If
// no finalized block or its respective child block exists we return with a nil
// block.
func (s *Store) FinalizedChildBlock(ctx context.Context, blockRoot [32]byte) (interfaces.ReadOnlySignedBeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.FinalizedChildBlock")
	defer span.End()

	var blk interfaces.ReadOnlySignedBeaconBlock
	err := s.db.View(func(tx *bolt.Tx) error {
		blkBytes := tx.Bucket(finalizedBlockRootsIndexBucket).Get(blockRoot[:])
		if blkBytes == nil {
			return nil
		}
		if bytes.Equal(blkBytes, containerFinalizedButNotCanonical) {
			return nil
		}
		ctr := &ethpb.FinalizedBlockRootContainer{}
		if err := decode(ctx, blkBytes, ctr); err != nil {
			tracing.AnnotateError(span, err)
			return err
		}
		enc := tx.Bucket(blocksBucket).Get(ctr.ChildRoot)
		if enc == nil {
			return nil
		}
		var err error
		blk, err = unmarshalBlock(ctx, enc)
		return err
	})
	tracing.AnnotateError(span, err)
	return blk, err
}

func pathToFinalizedCheckpoint(ctx context.Context, roots [][]byte, checkpointRoot []byte, tx *bolt.Tx) (bool, [][]byte) {
	if len(roots) == 0 || (len(roots) == 1 && roots[0] == nil) {
		return false, nil
	}

	for _, r := range roots {
		if bytes.Equal(r, checkpointRoot) {
			return true, [][]byte{r}
		}
		children := lookupValuesForIndices(ctx, map[string][]byte{string(blockParentRootIndicesBucket): r}, tx)
		if len(children) == 0 {
			children = [][][]byte{nil}
		}
		isPath, path := pathToFinalizedCheckpoint(ctx, children[0], checkpointRoot, tx)
		if isPath {
			return true, append([][]byte{r}, path...)
		}
	}

	return false, nil
}

func updatePrevFinalizedCheckpoint(ctx context.Context, span *trace.Span, finalizedBkt *bolt.Bucket, checkpoint *ethpb.Checkpoint) error {
	enc, err := encode(ctx, checkpoint)
	if err != nil {
		tracing.AnnotateError(span, err)
		return err
	}
	return finalizedBkt.Put(previousFinalizedCheckpointKey, enc)
}

func updateChildOfPrevFinalizedCheckpoint(ctx context.Context, span *trace.Span, finalizedBkt, parentBkt *bolt.Bucket, checkpointRoot []byte) error {
	container := &ethpb.FinalizedBlockRootContainer{}
	if err := decode(ctx, finalizedBkt.Get(checkpointRoot), container); err != nil {
		tracing.AnnotateError(span, err)
		return err
	}
	container.ChildRoot = parentBkt.Get(checkpointRoot)
	enc, err := encode(ctx, container)
	if err != nil {
		tracing.AnnotateError(span, err)
		return err
	}
	if err = finalizedBkt.Put(checkpointRoot, enc); err != nil {
		tracing.AnnotateError(span, err)
		return err
	}
	return nil
}
