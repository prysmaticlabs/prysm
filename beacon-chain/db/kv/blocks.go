package kv

import (
	"bytes"
	"context"
	"fmt"
	"strconv"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// Block retrieval by root.
func (k *Store) Block(ctx context.Context, blockRoot [32]byte) (*ethpb.SignedBeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.Block")
	defer span.End()
	// Return block from cache if it exists.
	if v, ok := k.blockCache.Get(string(blockRoot[:])); v != nil && ok {
		return v.(*ethpb.SignedBeaconBlock), nil
	}
	var block *ethpb.SignedBeaconBlock
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)
		enc := bkt.Get(blockRoot[:])
		if enc == nil {
			return nil
		}
		block = &ethpb.SignedBeaconBlock{}
		return decode(enc, block)
	})
	return block, err
}

// HeadBlock returns the latest canonical block in eth2.
func (k *Store) HeadBlock(ctx context.Context) (*ethpb.SignedBeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HeadBlock")
	defer span.End()
	var headBlock *ethpb.SignedBeaconBlock
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
		headBlock = &ethpb.SignedBeaconBlock{}
		return decode(enc, headBlock)
	})
	return headBlock, err
}

// Blocks retrieves a list of beacon blocks by filter criteria.
func (k *Store) Blocks(ctx context.Context, f *filters.QueryFilter) ([]*ethpb.SignedBeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.Blocks")
	defer span.End()
	blocks := make([]*ethpb.SignedBeaconBlock, 0)
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)

		// If no filter criteria are specified, return an error.
		if f == nil {
			return errors.New("must specify a filter criteria for retrieving blocks")
		}

		// Creates a list of indices from the passed in filter values, such as:
		// []byte("0x2093923") in the parent root indices bucket to be used for looking up
		// block roots that were stored under each of those indices for O(1) lookup.
		indicesByBucket, err := createBlockIndicesFromFilters(f)
		if err != nil {
			return errors.Wrap(err, "could not determine lookup indices")
		}

		// We retrieve block roots that match a filter criteria of slot ranges, if specified.
		filtersMap := f.Filters()
		rootsBySlotRange := fetchBlockRootsBySlotRange(
			tx.Bucket(blockSlotIndicesBucket),
			filtersMap[filters.StartSlot],
			filtersMap[filters.EndSlot],
			filtersMap[filters.StartEpoch],
			filtersMap[filters.EndEpoch],
			filtersMap[filters.SlotStep],
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
			block := &ethpb.SignedBeaconBlock{}
			if err := decode(encoded, block); err != nil {
				return err
			}
			blocks = append(blocks, block)
		}
		return nil
	})
	return blocks, err
}

// BlockRoots retrieves a list of beacon block roots by filter criteria.
func (k *Store) BlockRoots(ctx context.Context, f *filters.QueryFilter) ([][32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.BlockRoots")
	defer span.End()
	blockRoots := make([][32]byte, 0)
	err := k.db.View(func(tx *bolt.Tx) error {
		// If no filter criteria are specified, return an error.
		if f == nil {
			return errors.New("must specify a filter criteria for retrieving block roots")
		}

		// Creates a list of indices from the passed in filter values, such as:
		// []byte("0x2093923") in the parent root indices bucket to be used for looking up
		// block roots that were stored under each of those indices for O(1) lookup.
		indicesByBucket, err := createBlockIndicesFromFilters(f)
		if err != nil {
			return errors.Wrap(err, "could not determine lookup indices")
		}

		// We retrieve block roots that match a filter criteria of slot ranges, if specified.
		filtersMap := f.Filters()
		rootsBySlotRange := fetchBlockRootsBySlotRange(
			tx.Bucket(blockSlotIndicesBucket),
			filtersMap[filters.StartSlot],
			filtersMap[filters.EndSlot],
			filtersMap[filters.StartEpoch],
			filtersMap[filters.EndEpoch],
			filtersMap[filters.SlotStep],
		)

		// Once we have a list of block roots that correspond to each
		// lookup index, we find the intersection across all of them.
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
			blockRoots = append(blockRoots, bytesutil.ToBytes32(keys[i]))
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve block roots")
	}
	return blockRoots, nil
}

// HasBlock checks if a block by root exists in the db.
func (k *Store) HasBlock(ctx context.Context, blockRoot [32]byte) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HasBlock")
	defer span.End()
	if v, ok := k.blockCache.Get(string(blockRoot[:])); v != nil && ok {
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
func (k *Store) DeleteBlock(ctx context.Context, blockRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DeleteBlock")
	defer span.End()
	return k.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)
		enc := bkt.Get(blockRoot[:])
		if enc == nil {
			return nil
		}
		block := &ethpb.SignedBeaconBlock{}
		if err := decode(enc, block); err != nil {
			return err
		}
		indicesByBucket := createBlockIndicesFromBlock(block.Block)
		if err := deleteValueForIndices(indicesByBucket, blockRoot[:], tx); err != nil {
			return errors.Wrap(err, "could not delete root for DB indices")
		}
		k.blockCache.Del(string(blockRoot[:]))
		return bkt.Delete(blockRoot[:])
	})
}

// DeleteBlocks by block roots.
func (k *Store) DeleteBlocks(ctx context.Context, blockRoots [][32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DeleteBlocks")
	defer span.End()

	return k.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)
		for _, blockRoot := range blockRoots {
			enc := bkt.Get(blockRoot[:])
			if enc == nil {
				return nil
			}
			block := &ethpb.SignedBeaconBlock{}
			if err := decode(enc, block); err != nil {
				return err
			}
			indicesByBucket := createBlockIndicesFromBlock(block.Block)
			if err := deleteValueForIndices(indicesByBucket, blockRoot[:], tx); err != nil {
				return errors.Wrap(err, "could not delete root for DB indices")
			}
			k.blockCache.Del(string(blockRoot[:]))
			if err := bkt.Delete(blockRoot[:]); err != nil {
				return err
			}
		}
		return nil
	})
}

// SaveBlock to the db.
func (k *Store) SaveBlock(ctx context.Context, signed *ethpb.SignedBeaconBlock) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveBlock")
	defer span.End()
	blockRoot, err := ssz.HashTreeRoot(signed.Block)
	if err != nil {
		return err
	}
	if v, ok := k.blockCache.Get(string(blockRoot[:])); v != nil && ok {
		return nil
	}
	return k.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)
		if existingBlock := bkt.Get(blockRoot[:]); existingBlock != nil {
			return nil
		}
		enc, err := encode(signed)
		if err != nil {
			return err
		}
		indicesByBucket := createBlockIndicesFromBlock(signed.Block)
		if err := updateValueForIndices(indicesByBucket, blockRoot[:], tx); err != nil {
			return errors.Wrap(err, "could not update DB indices")
		}
		k.blockCache.Set(string(blockRoot[:]), signed, int64(len(enc)))
		return bkt.Put(blockRoot[:], enc)
	})
}

// SaveBlocks via bulk updates to the db.
func (k *Store) SaveBlocks(ctx context.Context, blocks []*ethpb.SignedBeaconBlock) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveBlocks")
	defer span.End()

	return k.db.Update(func(tx *bolt.Tx) error {
		for _, block := range blocks {
			blockRoot, err := ssz.HashTreeRoot(block.Block)
			if err != nil {
				return err
			}
			bkt := tx.Bucket(blocksBucket)
			if existingBlock := bkt.Get(blockRoot[:]); existingBlock != nil {
				return nil
			}
			enc, err := encode(block)
			if err != nil {
				return err
			}
			indicesByBucket := createBlockIndicesFromBlock(block.Block)
			if err := updateValueForIndices(indicesByBucket, blockRoot[:], tx); err != nil {
				return errors.Wrap(err, "could not update DB indices")
			}
			k.blockCache.Set(string(blockRoot[:]), block, int64(len(enc)))
			if err := bkt.Put(blockRoot[:], enc); err != nil {
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
		if tx.Bucket(stateBucket).Get(blockRoot[:]) == nil {
			return errors.New("no state found with head block root")
		}
		bucket := tx.Bucket(blocksBucket)
		return bucket.Put(headBlockRootKey, blockRoot[:])
	})
}

// GenesisBlock retrieves the genesis block of the beacon chain.
func (k *Store) GenesisBlock(ctx context.Context) (*ethpb.SignedBeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.GenesisBlock")
	defer span.End()
	var block *ethpb.SignedBeaconBlock
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)
		root := bkt.Get(genesisBlockRootKey)
		enc := bkt.Get(root)
		if enc == nil {
			return nil
		}
		block = &ethpb.SignedBeaconBlock{}
		return decode(enc, block)
	})
	return block, err
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
func fetchBlockRootsBySlotRange(
	bkt *bolt.Bucket,
	startSlotEncoded interface{},
	endSlotEncoded interface{},
	startEpochEncoded interface{},
	endEpochEncoded interface{},
	slotStepEncoded interface{},
) [][]byte {
	var startSlot, endSlot, step uint64
	var ok bool
	if startSlot, ok = startSlotEncoded.(uint64); !ok {
		startSlot = 0
	}
	if endSlot, ok = endSlotEncoded.(uint64); !ok {
		endSlot = 0
	}
	if step, ok = slotStepEncoded.(uint64); !ok || step == 0 {
		step = 1
	}
	startEpoch, startEpochOk := startEpochEncoded.(uint64)
	endEpoch, endEpochOk := endEpochEncoded.(uint64)
	if startEpochOk && endEpochOk {
		startSlot = helpers.StartSlot(startEpoch)
		endSlot = helpers.StartSlot(endEpoch) + params.BeaconConfig().SlotsPerEpoch - 1
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
	rootsRange := (endSlot - startSlot) / step
	if endSlot < startSlot {
		rootsRange = 0
	}
	roots := make([][]byte, 0, rootsRange)
	c := bkt.Cursor()
	for k, v := c.Seek(min); conditional(k, max); k, v = c.Next() {
		if step > 1 {
			slot, err := strconv.ParseUint(string(k), 10, 64)
			if err != nil {
				log.WithError(err).Error("Cannot parse key to uint")
				continue
			}
			if (slot-startSlot)%step != 0 {
				continue
			}
		}
		numOfRoots := len(v) / 32
		splitRoots := make([][]byte, 0, numOfRoots)
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
func createBlockIndicesFromBlock(block *ethpb.BeaconBlock) map[string][]byte {
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
		case filters.StartEpoch:
		case filters.EndEpoch:
		case filters.SlotStep:
		default:
			return nil, fmt.Errorf("filter criterion %v not supported for blocks", k)
		}
	}
	return indicesByBucket, nil
}
