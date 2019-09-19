package kv

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"go.opencensus.io/trace"
)

// Block retrieval by root.
func (k *Store) Block(ctx context.Context, blockRoot [32]byte) (*ethpb.BeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.Block")
	defer span.End()
	// Return block from cache if it exists.
	if v := k.blockCache.Get(string(blockRoot[:])); v != nil {
		return v.Value().(*ethpb.BeaconBlock), nil
	}
	var block *ethpb.BeaconBlock
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)
		enc := bkt.Get(blockRoot[:])
		if enc == nil {
			return nil
		}
		block = &ethpb.BeaconBlock{}
		return proto.Unmarshal(enc, block)
	})
	return block, err
}

// HeadBlock returns the latest canonical block in eth2.
func (k *Store) HeadBlock(ctx context.Context) (*ethpb.BeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HeadBlock")
	defer span.End()
	var headBlock *ethpb.BeaconBlock
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)
		headRoot := bkt.Get(headBlockRootKey)
		if headRoot == nil {
			return nil
		}
		enc := bkt.Get(headRoot)
		if enc == nil {
			return nil
		}
		headBlock = &ethpb.BeaconBlock{}
		return proto.Unmarshal(enc, headBlock)
	})
	return headBlock, err
}

// Blocks retrieves a list of beacon blocks by filter criteria.
func (k *Store) Blocks(ctx context.Context, f *filters.QueryFilter) ([]*ethpb.BeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.Blocks")
	defer span.End()
	blocks := make([]*ethpb.BeaconBlock, 0)
	err := k.db.Batch(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)

		// If no filter criteria are specified, return all blocks.
		if f == nil {
			return bkt.ForEach(func(k, v []byte) error {
				block := &ethpb.BeaconBlock{}
				if err := proto.Unmarshal(v, block); err != nil {
					return err
				}
				blocks = append(blocks, block)
				return nil
			})
		}

		// Creates a list of indices from the passed in filter values, such as:
		// []byte("0x2093923") in the parent root indices bucket to be used for looking up
		// block roots that were stored under each of those indices for O(1) lookup.
		indicesByBucket, err := createBlockIndicesFromFilters(f)
		if err != nil {
			return errors.Wrap(err, "could not determine block lookup indices")
		}

		// We retrieve block roots that match a filter criteria of slot ranges, if specified.
		filtersMap := f.Filters()
		rootsBySlotRange := fetchBlockRootsBySlotRange(
			tx.Bucket(blockSlotIndicesBucket),
			filtersMap[filters.StartSlot],
			filtersMap[filters.EndSlot],
		)

		// Once we have a list of block roots that correspond to each
		// lookup index, we find the intersection across all of them and use
		// that list of roots to lookup the block. These block will
		// meet the filter criteria.
		indices := lookupValuesForIndices(indicesByBucket, tx)
		keys := rootsBySlotRange
		if len(indices) > 0 {
			// If we have found indices that meet the filter criteria, and there are also
			// block roots that meet the slot range filter criteria, we find the intersection
			// between these two sets of roots.
			if len(rootsBySlotRange) > 0 {
				joined := append([][][]byte{keys}, indices...)
				keys = sliceutil.IntersectionByteSlices(joined...)
			} else {
				// If we have found indices that meet the filter criteria, but there are no block roots
				// that meet the slot range filter criteria, we find the intersection
				// of the regular filter indices.
				keys = sliceutil.IntersectionByteSlices(indices...)
			}
		}
		for i := 0; i < len(keys); i++ {
			encoded := bkt.Get(keys[i])
			block := &ethpb.BeaconBlock{}
			if err := proto.Unmarshal(encoded, block); err != nil {
				return err
			}
			blocks = append(blocks, block)
		}
		return nil
	})
	return blocks, err
}

// BlockRoots retrieves a list of beacon block roots by filter criteria.
func (k *Store) BlockRoots(ctx context.Context, f *filters.QueryFilter) ([][]byte, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.BlockRoots")
	defer span.End()
	blocks, err := k.Blocks(ctx, f)
	if err != nil {
		return nil, err
	}
	roots := make([][]byte, len(blocks))
	for i, b := range blocks {
		root, err := ssz.SigningRoot(b)
		if err != nil {
			return nil, err
		}
		roots[i] = root[:]
	}
	return roots, nil
}

// HasBlock checks if a block by root exists in the db.
func (k *Store) HasBlock(ctx context.Context, blockRoot [32]byte) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HasBlock")
	defer span.End()
	if v := k.blockCache.Get(string(blockRoot[:])); v != nil {
		return true
	}
	exists := false
	// #nosec G104. Always returns nil.
	k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)
		exists = bkt.Get(blockRoot[:]) != nil
		return nil
	})
	return exists
}

// DeleteBlock by block root.
// TODO(#3064): Add the ability for batch deletions.
func (k *Store) DeleteBlock(ctx context.Context, blockRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DeleteBlock")
	defer span.End()
	return k.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)
		enc := bkt.Get(blockRoot[:])
		if enc == nil {
			return nil
		}
		block := &ethpb.BeaconBlock{}
		if err := proto.Unmarshal(enc, block); err != nil {
			return err
		}
		indicesByBucket := createBlockIndicesFromBlock(block, tx)
		if err := deleteValueForIndices(indicesByBucket, blockRoot[:], tx); err != nil {
			return errors.Wrap(err, "could not delete root for DB indices")
		}
		k.blockCache.Delete(string(blockRoot[:]))
		return bkt.Delete(blockRoot[:])
	})
}

// SaveBlock to the db.
func (k *Store) SaveBlock(ctx context.Context, block *ethpb.BeaconBlock) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveBlock")
	defer span.End()
	blockRoot, err := ssz.SigningRoot(block)
	if err != nil {
		return err
	}

	if v := k.blockCache.Get(string(blockRoot[:])); v != nil {
		return nil
	}

	enc, err := proto.Marshal(block)
	if err != nil {
		return err
	}
	return k.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)
		indicesByBucket := createBlockIndicesFromBlock(block, tx)
		if err := updateValueForIndices(indicesByBucket, blockRoot[:], tx); err != nil {
			return errors.Wrap(err, "could not update DB indices")
		}
		k.blockCache.Set(string(blockRoot[:]), block, time.Hour)
		return bkt.Put(blockRoot[:], enc)
	})
}

// SaveBlocks via batch updates to the db.
func (k *Store) SaveBlocks(ctx context.Context, blocks []*ethpb.BeaconBlock) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveBlocks")
	defer span.End()
	encodedValues := make([][]byte, len(blocks))
	keys := make([][]byte, len(blocks))
	for i := 0; i < len(blocks); i++ {
		enc, err := proto.Marshal(blocks[i])
		if err != nil {
			return err
		}
		key, err := ssz.SigningRoot(blocks[i])
		if err != nil {
			return err
		}
		encodedValues[i] = enc
		keys[i] = key[:]
	}
	return k.db.Batch(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blocksBucket)
		for i := 0; i < len(blocks); i++ {
			indicesByBucket := createBlockIndicesFromBlock(blocks[i], tx)
			if err := updateValueForIndices(indicesByBucket, keys[i], tx); err != nil {
				return errors.Wrap(err, "could not update DB indices")
			}
			k.blockCache.Set(string(keys[i]), blocks[i], time.Hour)
			if err := bucket.Put(keys[i], encodedValues[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

// SaveHeadBlockRoot to the db.
func (k *Store) SaveHeadBlockRoot(ctx context.Context, blockRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveHeadBlockRoot")
	defer span.End()
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blocksBucket)
		return bucket.Put(headBlockRootKey, blockRoot[:])
	})
}

// SaveGenesisBlockRoot to the db.
func (k *Store) SaveGenesisBlockRoot(ctx context.Context, blockRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveGenesisBlockRoot")
	defer span.End()
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blocksBucket)
		return bucket.Put(genesisBlockRootKey, blockRoot[:])
	})
}

// fetchBlockRootsBySlotRange looks into a boltDB bucket and performs a binary search
// range scan using sorted left-padded byte keys using a start slot and an end slot.
// If both the start and end slot are the same, and are 0, the function returns nil.
func fetchBlockRootsBySlotRange(bkt *bolt.Bucket, startSlotEncoded, endSlotEncoded interface{}) [][]byte {
	var startSlot, endSlot uint64
	var ok bool
	if startSlot, ok = startSlotEncoded.(uint64); !ok {
		startSlot = 0
	}
	if endSlot, ok = endSlotEncoded.(uint64); !ok {
		endSlot = 0
	}
	if startSlot == endSlot && startSlot == 0 {
		return nil
	}
	min := []byte(fmt.Sprintf("%07d", startSlot))
	max := []byte(fmt.Sprintf("%07d", endSlot))
	var conditional func(key, max []byte) bool
	if endSlot == 0 {
		conditional = func(k, max []byte) bool {
			return k != nil
		}
	} else {
		conditional = func(k, max []byte) bool {
			return k != nil && bytes.Compare(k, max) <= 0
		}
	}
	roots := make([][]byte, 0)
	c := bkt.Cursor()
	for k, v := c.Seek(min); conditional(k, max); k, v = c.Next() {
		splitRoots := make([][]byte, 0)
		for i := 0; i < len(v); i += 32 {
			splitRoots = append(splitRoots, v[i:i+32])
		}
		roots = append(roots, splitRoots...)
	}
	return roots
}

// createBlockIndicesFromBlock takes in a beacon block and returns
// a map of bolt DB index buckets corresponding to each particular key for indices for
// data, such as (shard indices bucket -> shard 5).
func createBlockIndicesFromBlock(block *ethpb.BeaconBlock, tx *bolt.Tx) map[string][]byte {
	indicesByBucket := make(map[string][]byte)
	// Every index has a unique bucket for fast, binary-search
	// range scans for filtering across keys.
	buckets := [][]byte{
		blockSlotIndicesBucket,
	}
	indices := [][]byte{
		[]byte(fmt.Sprintf("%07d", block.Slot)),
	}
	if block.ParentRoot != nil && len(block.ParentRoot) > 0 {
		buckets = append(buckets, blockParentRootIndicesBucket)
		indices = append(indices, block.ParentRoot)
	}
	for i := 0; i < len(buckets); i++ {
		indicesByBucket[string(buckets[i])] = indices[i]
	}
	return indicesByBucket
}

// createBlockFiltersFromIndices takes in filter criteria and returns
// a list of of byte keys used to retrieve the values stored
// for the indices from the DB.
//
// For blocks, these are list of signing roots of block
// objects. If a certain filter criterion does not apply to
// blocks, an appropriate error is returned.
func createBlockIndicesFromFilters(f *filters.QueryFilter) (map[string][]byte, error) {
	indicesByBucket := make(map[string][]byte)
	for k, v := range f.Filters() {
		switch k {
		case filters.ParentRoot:
			parentRoot := v.([]byte)
			indicesByBucket[string(blockParentRootIndicesBucket)] = parentRoot
		case filters.StartSlot:
		case filters.EndSlot:
		default:
			return nil, fmt.Errorf("filter criterion %v not supported for blocks", k)
		}
	}
	return indicesByBucket, nil
}
