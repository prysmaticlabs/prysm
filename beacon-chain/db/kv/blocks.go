package kv

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	ssz "github.com/ferranbt/fastssz"
	"github.com/golang/snappy"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/container/slice"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/prysmaticlabs/prysm/time/slots"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// used to represent errors for inconsistent slot ranges.
var errInvalidSlotRange = errors.New("invalid end slot and start slot provided")

// Block retrieval by root.
func (s *Store) Block(ctx context.Context, blockRoot [32]byte) (block.SignedBeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.Block")
	defer span.End()
	// Return block from cache if it exists.
	if v, ok := s.blockCache.Get(string(blockRoot[:])); v != nil && ok {
		return v.(block.SignedBeaconBlock), nil
	}
	var blk block.SignedBeaconBlock
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)
		enc := bkt.Get(blockRoot[:])
		if enc == nil {
			return nil
		}
		var err error
		blk, err = unmarshalBlock(ctx, enc)
		return err
	})
	return blk, err
}

// OriginCheckpointBlockRoot returns the value written to the db in SaveOriginCheckpointBlockRoot
// This is the root of a finalized block within the weak subjectivity period
// at the time the chain was started, used to initialize the database and chain
// without syncing from genesis.
func (s *Store) OriginCheckpointBlockRoot(ctx context.Context) ([32]byte, error) {
	_, span := trace.StartSpan(ctx, "BeaconDB.OriginCheckpointBlockRoot")
	defer span.End()

	var root [32]byte
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)
		rootSlice := bkt.Get(originCheckpointBlockRootKey)
		if rootSlice == nil {
			return ErrNotFoundOriginBlockRoot
		}
		copy(root[:], rootSlice)
		return nil
	})

	return root, err
}

// BackfillBlockRoot keeps track of the highest block available before the OriginCheckpointBlockRoot
func (s *Store) BackfillBlockRoot(ctx context.Context) ([32]byte, error) {
	_, span := trace.StartSpan(ctx, "BeaconDB.BackfillBlockRoot")
	defer span.End()

	var root [32]byte
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)
		rootSlice := bkt.Get(backfillBlockRootKey)
		if len(rootSlice) == 0 {
			return ErrNotFoundBackfillBlockRoot
		}
		root = bytesutil.ToBytes32(rootSlice)
		return nil
	})

	return root, err
}

// HeadBlock returns the latest canonical block in the Ethereum Beacon Chain.
func (s *Store) HeadBlock(ctx context.Context) (block.SignedBeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HeadBlock")
	defer span.End()
	var headBlock block.SignedBeaconBlock
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)
		headRoot := bkt.Get(headBlockRootKey)
		if headRoot == nil {
			return nil
		}
		enc := bkt.Get(headRoot)
		if enc == nil {
			return nil
		}
		var err error
		headBlock, err = unmarshalBlock(ctx, enc)
		return err
	})
	return headBlock, err
}

// Blocks retrieves a list of beacon blocks and its respective roots by filter criteria.
func (s *Store) Blocks(ctx context.Context, f *filters.QueryFilter) ([]block.SignedBeaconBlock, [][32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.Blocks")
	defer span.End()
	blocks := make([]block.SignedBeaconBlock, 0)
	blockRoots := make([][32]byte, 0)

	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)

		keys, err := blockRootsByFilter(ctx, tx, f)
		if err != nil {
			return err
		}

		for i := 0; i < len(keys); i++ {
			encoded := bkt.Get(keys[i])
			blk, err := unmarshalBlock(ctx, encoded)
			if err != nil {
				return err
			}
			blocks = append(blocks, blk)
			blockRoots = append(blockRoots, bytesutil.ToBytes32(keys[i]))
		}
		return nil
	})
	return blocks, blockRoots, err
}

// BlockRoots retrieves a list of beacon block roots by filter criteria. If the caller
// requires both the blocks and the block roots for a certain filter they should instead
// use the Blocks function rather than use BlockRoots. During periods of non finality
// there are potential race conditions which leads to differing roots when calling the db
// multiple times for the same filter.
func (s *Store) BlockRoots(ctx context.Context, f *filters.QueryFilter) ([][32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.BlockRoots")
	defer span.End()
	blockRoots := make([][32]byte, 0)
	err := s.db.View(func(tx *bolt.Tx) error {
		keys, err := blockRootsByFilter(ctx, tx, f)
		if err != nil {
			return err
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
func (s *Store) HasBlock(ctx context.Context, blockRoot [32]byte) bool {
	_, span := trace.StartSpan(ctx, "BeaconDB.HasBlock")
	defer span.End()
	if v, ok := s.blockCache.Get(string(blockRoot[:])); v != nil && ok {
		return true
	}
	exists := false
	if err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)
		exists = bkt.Get(blockRoot[:]) != nil
		return nil
	}); err != nil { // This view never returns an error, but we'll handle anyway for sanity.
		panic(err)
	}
	return exists
}

// BlocksBySlot retrieves a list of beacon blocks and its respective roots by slot.
func (s *Store) BlocksBySlot(ctx context.Context, slot types.Slot) (bool, []block.SignedBeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.BlocksBySlot")
	defer span.End()
	blocks := make([]block.SignedBeaconBlock, 0)

	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)

		keys := blockRootsBySlot(ctx, tx, slot)
		for i := 0; i < len(keys); i++ {
			encoded := bkt.Get(keys[i])
			blk, err := unmarshalBlock(ctx, encoded)
			if err != nil {
				return err
			}
			blocks = append(blocks, blk)
		}
		return nil
	})
	return len(blocks) > 0, blocks, err
}

// BlockRootsBySlot retrieves a list of beacon block roots by slot
func (s *Store) BlockRootsBySlot(ctx context.Context, slot types.Slot) (bool, [][32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.BlockRootsBySlot")
	defer span.End()
	blockRoots := make([][32]byte, 0)
	err := s.db.View(func(tx *bolt.Tx) error {
		keys := blockRootsBySlot(ctx, tx, slot)
		for i := 0; i < len(keys); i++ {
			blockRoots = append(blockRoots, bytesutil.ToBytes32(keys[i]))
		}
		return nil
	})
	if err != nil {
		return false, nil, errors.Wrap(err, "could not retrieve block roots by slot")
	}
	return len(blockRoots) > 0, blockRoots, nil
}

// DeleteBlock from the db
// This deletes the root entry from all buckets in the blocks DB
// If the block is finalized this function returns an error
func (s *Store) DeleteBlock(ctx context.Context, root [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DeleteBlock")
	defer span.End()

	if err := s.DeleteState(ctx, root); err != nil {
		return errDeleteFinalized
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(finalizedBlockRootsIndexBucket)
		if b := bkt.Get(root[:]); b != nil {
			return errDeleteFinalized
		}

		if err := tx.Bucket(blocksBucket).Delete(root[:]); err != nil {
			return err
		}
		if err := tx.Bucket(blockParentRootIndicesBucket).Delete(root[:]); err != nil {
			return err
		}
		s.blockCache.Del(string(root[:]))
		return nil
	})
}

// SaveBlock to the db.
func (s *Store) SaveBlock(ctx context.Context, signed block.SignedBeaconBlock) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveBlock")
	defer span.End()
	blockRoot, err := signed.Block().HashTreeRoot()
	if err != nil {
		return err
	}
	if v, ok := s.blockCache.Get(string(blockRoot[:])); v != nil && ok {
		return nil
	}
	return s.SaveBlocks(ctx, []block.SignedBeaconBlock{signed})
}

// SaveBlocks via bulk updates to the db.
func (s *Store) SaveBlocks(ctx context.Context, blocks []block.SignedBeaconBlock) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveBlocks")
	defer span.End()

	// Performing marshaling, hashing, and indexing outside the bolt transaction
	// to minimize the time we hold the DB lock.
	blockRoots := make([][]byte, len(blocks))
	encodedBlocks := make([][]byte, len(blocks))
	indicesForBlocks := make([]map[string][]byte, len(blocks))
	for i, blk := range blocks {
		blockRoot, err := blk.Block().HashTreeRoot()
		if err != nil {
			return err
		}
		enc, err := marshalBlock(ctx, blk)
		if err != nil {
			return err
		}
		blockRoots[i] = blockRoot[:]
		encodedBlocks[i] = enc
		indicesByBucket := createBlockIndicesFromBlock(ctx, blk.Block())
		indicesForBlocks[i] = indicesByBucket
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)
		for i, blk := range blocks {
			if existingBlock := bkt.Get(blockRoots[i]); existingBlock != nil {
				continue
			}
			if err := updateValueForIndices(ctx, indicesForBlocks[i], blockRoots[i], tx); err != nil {
				return errors.Wrap(err, "could not update DB indices")
			}
			s.blockCache.Set(string(blockRoots[i]), blk, int64(len(encodedBlocks[i])))
			if err := bkt.Put(blockRoots[i], encodedBlocks[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

// SaveHeadBlockRoot to the db.
func (s *Store) SaveHeadBlockRoot(ctx context.Context, blockRoot [32]byte) error {
	_, span := trace.StartSpan(ctx, "BeaconDB.SaveHeadBlockRoot")
	defer span.End()
	return s.db.Update(func(tx *bolt.Tx) error {
		hasStateSummary := s.hasStateSummaryBytes(tx, blockRoot)
		hasStateInDB := tx.Bucket(stateBucket).Get(blockRoot[:]) != nil
		if !(hasStateInDB || hasStateSummary) {
			return errors.New("no state or state summary found with head block root")
		}

		bucket := tx.Bucket(blocksBucket)
		return bucket.Put(headBlockRootKey, blockRoot[:])
	})
}

// GenesisBlock retrieves the genesis block of the beacon chain.
func (s *Store) GenesisBlock(ctx context.Context) (block.SignedBeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.GenesisBlock")
	defer span.End()
	var blk block.SignedBeaconBlock
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)
		root := bkt.Get(genesisBlockRootKey)
		enc := bkt.Get(root)
		if enc == nil {
			return nil
		}
		var err error
		blk, err = unmarshalBlock(ctx, enc)
		return err
	})
	return blk, err
}

func (s *Store) GenesisBlockRoot(ctx context.Context) ([32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.GenesisBlockRoot")
	defer span.End()
	var root [32]byte
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)
		r := bkt.Get(genesisBlockRootKey)
		if len(r) == 0 {
			return ErrNotFoundGenesisBlockRoot
		}
		root = bytesutil.ToBytes32(r)
		return nil
	})
	return root, err
}

// SaveGenesisBlockRoot to the db.
func (s *Store) SaveGenesisBlockRoot(ctx context.Context, blockRoot [32]byte) error {
	_, span := trace.StartSpan(ctx, "BeaconDB.SaveGenesisBlockRoot")
	defer span.End()
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blocksBucket)
		return bucket.Put(genesisBlockRootKey, blockRoot[:])
	})
}

// SaveOriginCheckpointBlockRoot is used to keep track of the block root used for syncing from a checkpoint origin.
// This should be a finalized block from within the current weak subjectivity period.
// This value is used by a running beacon chain node to locate the state at the beginning
// of the chain history, in places where genesis would typically be used.
func (s *Store) SaveOriginCheckpointBlockRoot(ctx context.Context, blockRoot [32]byte) error {
	_, span := trace.StartSpan(ctx, "BeaconDB.SaveOriginCheckpointBlockRoot")
	defer span.End()
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blocksBucket)
		return bucket.Put(originCheckpointBlockRootKey, blockRoot[:])
	})
}

// SaveBackfillBlockRoot is used to keep track of the most recently backfilled block root when
// the node was initialized via checkpoint sync.
func (s *Store) SaveBackfillBlockRoot(ctx context.Context, blockRoot [32]byte) error {
	_, span := trace.StartSpan(ctx, "BeaconDB.SaveBackfillBlockRoot")
	defer span.End()
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blocksBucket)
		return bucket.Put(backfillBlockRootKey, blockRoot[:])
	})
}

// HighestSlotBlocksBelow returns the block with the highest slot below the input slot from the db.
func (s *Store) HighestSlotBlocksBelow(ctx context.Context, slot types.Slot) ([]block.SignedBeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HighestSlotBlocksBelow")
	defer span.End()

	var best []byte
	if err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blockSlotIndicesBucket)
		// Iterate through the index, which is in byte sorted order.
		c := bkt.Cursor()
		for s, root := c.First(); s != nil; s, root = c.Next() {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			key := bytesutil.BytesToSlotBigEndian(s)
			if root == nil {
				continue
			}
			if key >= slot {
				break
			}
			best = root
		}
		return nil
	}); err != nil {
		return nil, err
	}

	var blk block.SignedBeaconBlock
	var err error
	if best != nil {
		blk, err = s.Block(ctx, bytesutil.ToBytes32(best))
		if err != nil {
			return nil, err
		}
	}
	if blk == nil || blk.IsNil() {
		blk, err = s.GenesisBlock(ctx)
		if err != nil {
			return nil, err
		}
	}

	return []block.SignedBeaconBlock{blk}, nil
}

// FeeRecipientByValidatorID returns the fee recipient for a validator id.
// `ErrNotFoundFeeRecipient` is returned if the validator id is not found.
func (s *Store) FeeRecipientByValidatorID(ctx context.Context, id types.ValidatorIndex) (common.Address, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.FeeRecipientByValidatorID")
	defer span.End()
	var addr []byte
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(feeRecipientBucket)
		addr = bkt.Get(bytesutil.Uint64ToBytesBigEndian(uint64(id)))
		if addr == nil {
			return errors.Wrapf(ErrNotFoundFeeRecipient, "validator id %d", id)
		}
		return nil
	})
	return common.BytesToAddress(addr), err
}

// SaveFeeRecipientsByValidatorIDs saves the fee recipients for validator ids.
// Error is returned if `ids` and `recipients` are not the same length.
func (s *Store) SaveFeeRecipientsByValidatorIDs(ctx context.Context, ids []types.ValidatorIndex, feeRecipients []common.Address) error {
	_, span := trace.StartSpan(ctx, "BeaconDB.SaveFeeRecipientByValidatorID")
	defer span.End()

	if len(ids) != len(feeRecipients) {
		return errors.New("validatorIDs and feeRecipients must be the same length")
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(feeRecipientBucket)
		for i, id := range ids {
			if err := bkt.Put(bytesutil.Uint64ToBytesBigEndian(uint64(id)), feeRecipients[i].Bytes()); err != nil {
				return err
			}
		}
		return nil
	})
}

// blockRootsByFilter retrieves the block roots given the filter criteria.
func blockRootsByFilter(ctx context.Context, tx *bolt.Tx, f *filters.QueryFilter) ([][]byte, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.blockRootsByFilter")
	defer span.End()

	// If no filter criteria are specified, return an error.
	if f == nil {
		return nil, errors.New("must specify a filter criteria for retrieving blocks")
	}

	// Creates a list of indices from the passed in filter values, such as:
	// []byte("0x2093923") in the parent root indices bucket to be used for looking up
	// block roots that were stored under each of those indices for O(1) lookup.
	indicesByBucket, err := createBlockIndicesFromFilters(ctx, f)
	if err != nil {
		return nil, errors.Wrap(err, "could not determine lookup indices")
	}

	// We retrieve block roots that match a filter criteria of slot ranges, if specified.
	filtersMap := f.Filters()
	rootsBySlotRange, err := blockRootsBySlotRange(
		ctx,
		tx.Bucket(blockSlotIndicesBucket),
		filtersMap[filters.StartSlot],
		filtersMap[filters.EndSlot],
		filtersMap[filters.StartEpoch],
		filtersMap[filters.EndEpoch],
		filtersMap[filters.SlotStep],
	)
	if err != nil {
		return nil, err
	}

	// Once we have a list of block roots that correspond to each
	// lookup index, we find the intersection across all of them and use
	// that list of roots to lookup the block. These block will
	// meet the filter criteria.
	indices := lookupValuesForIndices(ctx, indicesByBucket, tx)
	keys := rootsBySlotRange
	if len(indices) > 0 {
		// If we have found indices that meet the filter criteria, and there are also
		// block roots that meet the slot range filter criteria, we find the intersection
		// between these two sets of roots.
		if len(rootsBySlotRange) > 0 {
			joined := append([][][]byte{keys}, indices...)
			keys = slice.IntersectionByteSlices(joined...)
		} else {
			// If we have found indices that meet the filter criteria, but there are no block roots
			// that meet the slot range filter criteria, we find the intersection
			// of the regular filter indices.
			keys = slice.IntersectionByteSlices(indices...)
		}
	}

	return keys, nil
}

// blockRootsBySlotRange looks into a boltDB bucket and performs a binary search
// range scan using sorted left-padded byte keys using a start slot and an end slot.
// However, if step is one, the implemented logic won’t skip half of the slots in the range.
func blockRootsBySlotRange(
	ctx context.Context,
	bkt *bolt.Bucket,
	startSlotEncoded, endSlotEncoded, startEpochEncoded, endEpochEncoded, slotStepEncoded interface{},
) ([][]byte, error) {
	_, span := trace.StartSpan(ctx, "BeaconDB.blockRootsBySlotRange")
	defer span.End()

	// Return nothing when all slot parameters are missing
	if startSlotEncoded == nil && endSlotEncoded == nil && startEpochEncoded == nil && endEpochEncoded == nil {
		return [][]byte{}, nil
	}

	var startSlot, endSlot types.Slot
	var step uint64
	var ok bool
	if startSlot, ok = startSlotEncoded.(types.Slot); !ok {
		startSlot = 0
	}
	if endSlot, ok = endSlotEncoded.(types.Slot); !ok {
		endSlot = 0
	}
	if step, ok = slotStepEncoded.(uint64); !ok || step == 0 {
		step = 1
	}
	startEpoch, startEpochOk := startEpochEncoded.(types.Epoch)
	endEpoch, endEpochOk := endEpochEncoded.(types.Epoch)
	var err error
	if startEpochOk && endEpochOk {
		startSlot, err = slots.EpochStart(startEpoch)
		if err != nil {
			return nil, err
		}
		endSlot, err = slots.EpochStart(endEpoch)
		if err != nil {
			return nil, err
		}
		endSlot = endSlot + params.BeaconConfig().SlotsPerEpoch - 1
	}
	min := bytesutil.SlotToBytesBigEndian(startSlot)
	max := bytesutil.SlotToBytesBigEndian(endSlot)

	conditional := func(key, max []byte) bool {
		return key != nil && bytes.Compare(key, max) <= 0
	}
	if endSlot < startSlot {
		return nil, errInvalidSlotRange
	}
	rootsRange := endSlot.SubSlot(startSlot).Div(step)
	roots := make([][]byte, 0, rootsRange)
	c := bkt.Cursor()
	for k, v := c.Seek(min); conditional(k, max); k, v = c.Next() {
		if step > 1 {
			slot := bytesutil.BytesToSlotBigEndian(k)
			if slot.SubSlot(startSlot).Mod(step) != 0 {
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
	return roots, nil
}

// blockRootsBySlot retrieves the block roots by slot
func blockRootsBySlot(ctx context.Context, tx *bolt.Tx, slot types.Slot) [][]byte {
	_, span := trace.StartSpan(ctx, "BeaconDB.blockRootsBySlot")
	defer span.End()

	roots := make([][]byte, 0)
	bkt := tx.Bucket(blockSlotIndicesBucket)
	key := bytesutil.SlotToBytesBigEndian(slot)
	c := bkt.Cursor()
	k, v := c.Seek(key)
	if k != nil && bytes.Equal(k, key) {
		for i := 0; i < len(v); i += 32 {
			roots = append(roots, v[i:i+32])
		}
	}
	return roots
}

// createBlockIndicesFromBlock takes in a beacon block and returns
// a map of bolt DB index buckets corresponding to each particular key for indices for
// data, such as (shard indices bucket -> shard 5).
func createBlockIndicesFromBlock(ctx context.Context, block block.BeaconBlock) map[string][]byte {
	_, span := trace.StartSpan(ctx, "BeaconDB.createBlockIndicesFromBlock")
	defer span.End()
	indicesByBucket := make(map[string][]byte)
	// Every index has a unique bucket for fast, binary-search
	// range scans for filtering across keys.
	buckets := [][]byte{
		blockSlotIndicesBucket,
	}
	indices := [][]byte{
		bytesutil.SlotToBytesBigEndian(block.Slot()),
	}
	if block.ParentRoot() != nil && len(block.ParentRoot()) > 0 {
		buckets = append(buckets, blockParentRootIndicesBucket)
		indices = append(indices, block.ParentRoot())
	}
	for i := 0; i < len(buckets); i++ {
		indicesByBucket[string(buckets[i])] = indices[i]
	}
	return indicesByBucket
}

// createBlockFiltersFromIndices takes in filter criteria and returns
// a map with a single key-value pair: "block-parent-root-indices” -> parentRoot (array of bytes).
//
// For blocks, these are list of signing roots of block
// objects. If a certain filter criterion does not apply to
// blocks, an appropriate error is returned.
func createBlockIndicesFromFilters(ctx context.Context, f *filters.QueryFilter) (map[string][]byte, error) {
	_, span := trace.StartSpan(ctx, "BeaconDB.createBlockIndicesFromFilters")
	defer span.End()
	indicesByBucket := make(map[string][]byte)
	for k, v := range f.Filters() {
		switch k {
		case filters.ParentRoot:
			parentRoot, ok := v.([]byte)
			if !ok {
				return nil, errors.New("parent root is not []byte")
			}
			indicesByBucket[string(blockParentRootIndicesBucket)] = parentRoot
		// The following cases are passthroughs for blocks, as they are not used
		// for filtering indices.
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

// unmarshal block from marshaled proto beacon block bytes to versioned beacon block struct type.
func unmarshalBlock(_ context.Context, enc []byte) (block.SignedBeaconBlock, error) {
	var err error
	enc, err = snappy.Decode(nil, enc)
	if err != nil {
		return nil, err
	}
	var rawBlock ssz.Unmarshaler
	switch {
	case hasAltairKey(enc):
		// Marshal block bytes to altair beacon block.
		rawBlock = &ethpb.SignedBeaconBlockAltair{}
		if err := rawBlock.UnmarshalSSZ(enc[len(altairKey):]); err != nil {
			return nil, err
		}
	case hasBellatrixKey(enc):
		rawBlock = &ethpb.SignedBeaconBlockBellatrix{}
		if err := rawBlock.UnmarshalSSZ(enc[len(bellatrixKey):]); err != nil {
			return nil, err
		}
	default:
		// Marshal block bytes to phase 0 beacon block.
		rawBlock = &ethpb.SignedBeaconBlock{}
		if err := rawBlock.UnmarshalSSZ(enc); err != nil {
			return nil, err
		}
	}
	return wrapper.WrappedSignedBeaconBlock(rawBlock)
}

// marshal versioned beacon block from struct type down to bytes.
func marshalBlock(_ context.Context, blk block.SignedBeaconBlock) ([]byte, error) {
	obj, err := blk.MarshalSSZ()
	if err != nil {
		return nil, err
	}
	switch blk.Version() {
	case version.Bellatrix:
		return snappy.Encode(nil, append(bellatrixKey, obj...)), nil
	case version.Altair:
		return snappy.Encode(nil, append(altairKey, obj...)), nil
	case version.Phase0:
		return snappy.Encode(nil, obj), nil
	default:
		return nil, errors.New("Unknown block version")
	}
}
