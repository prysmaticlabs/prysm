package kv

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/snappy"
	"github.com/pkg/errors"
	ssz "github.com/prysmaticlabs/fastssz"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/v3/config/features"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/container/slice"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// used to represent errors for inconsistent slot ranges.
var errInvalidSlotRange = errors.New("invalid end slot and start slot provided")

// Block retrieval by root.
func (s *Store) Block(ctx context.Context, blockRoot [32]byte) (interfaces.SignedBeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.Block")
	defer span.End()
	// Return block from cache if it exists.
	if v, ok := s.blockCache.Get(string(blockRoot[:])); v != nil && ok {
		return v.(interfaces.SignedBeaconBlock), nil
	}
	var blk interfaces.SignedBeaconBlock
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
	ctx, span := trace.StartSpan(ctx, "BeaconDB.OriginCheckpointBlockRoot")
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
	ctx, span := trace.StartSpan(ctx, "BeaconDB.BackfillBlockRoot")
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
func (s *Store) HeadBlock(ctx context.Context) (interfaces.SignedBeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HeadBlock")
	defer span.End()
	var headBlock interfaces.SignedBeaconBlock
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
func (s *Store) Blocks(ctx context.Context, f *filters.QueryFilter) ([]interfaces.SignedBeaconBlock, [][32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.Blocks")
	defer span.End()
	blocks := make([]interfaces.SignedBeaconBlock, 0)
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
				return errors.Wrapf(err, "could not unmarshal block with key %#x", keys[i])
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
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HasBlock")
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
func (s *Store) BlocksBySlot(ctx context.Context, slot types.Slot) ([]interfaces.SignedBeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.BlocksBySlot")
	defer span.End()

	blocks := make([]interfaces.SignedBeaconBlock, 0)
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)
		roots, err := blockRootsBySlot(ctx, tx, slot)
		if err != nil {
			return errors.Wrap(err, "could not retrieve blocks by slot")
		}
		for _, r := range roots {
			encoded := bkt.Get(r[:])
			blk, err := unmarshalBlock(ctx, encoded)
			if err != nil {
				return err
			}
			blocks = append(blocks, blk)
		}
		return nil
	})
	return blocks, err
}

// BlockRootsBySlot retrieves a list of beacon block roots by slot
func (s *Store) BlockRootsBySlot(ctx context.Context, slot types.Slot) (bool, [][32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.BlockRootsBySlot")
	defer span.End()
	blockRoots := make([][32]byte, 0)
	err := s.db.View(func(tx *bolt.Tx) error {
		var err error
		blockRoots, err = blockRootsBySlot(ctx, tx, slot)
		return err
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
		return err
	}

	if err := s.deleteStateSummary(root); err != nil {
		return err
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(finalizedBlockRootsIndexBucket)
		if b := bkt.Get(root[:]); b != nil {
			return ErrDeleteJustifiedAndFinalized
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
func (s *Store) SaveBlock(ctx context.Context, signed interfaces.SignedBeaconBlock) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveBlock")
	defer span.End()
	blockRoot, err := signed.Block().HashTreeRoot()
	if err != nil {
		return err
	}
	if v, ok := s.blockCache.Get(string(blockRoot[:])); v != nil && ok {
		return nil
	}
	return s.SaveBlocks(ctx, []interfaces.SignedBeaconBlock{signed})
}

// SaveBlocks via bulk updates to the db.
func (s *Store) SaveBlocks(ctx context.Context, blks []interfaces.SignedBeaconBlock) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveBlocks")
	defer span.End()

	// Performing marshaling, hashing, and indexing outside the bolt transaction
	// to minimize the time we hold the DB lock.
	blockRoots := make([][]byte, len(blks))
	encodedBlocks := make([][]byte, len(blks))
	indicesForBlocks := make([]map[string][]byte, len(blks))
	for i, blk := range blks {
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
		for i, blk := range blks {
			if existingBlock := bkt.Get(blockRoots[i]); existingBlock != nil {
				continue
			}
			if err := updateValueForIndices(ctx, indicesForBlocks[i], blockRoots[i], tx); err != nil {
				return errors.Wrap(err, "could not update DB indices")
			}
			if features.Get().EnableOnlyBlindedBeaconBlocks {
				blindedBlock, err := blk.ToBlinded()
				if err != nil {
					if !errors.Is(err, blocks.ErrUnsupportedVersion) {
						return err
					}
				} else {
					blk = blindedBlock
				}
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
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveHeadBlockRoot")
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
func (s *Store) GenesisBlock(ctx context.Context) (interfaces.SignedBeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.GenesisBlock")
	defer span.End()
	var blk interfaces.SignedBeaconBlock
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
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveGenesisBlockRoot")
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
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveOriginCheckpointBlockRoot")
	defer span.End()
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blocksBucket)
		return bucket.Put(originCheckpointBlockRootKey, blockRoot[:])
	})
}

// SaveBackfillBlockRoot is used to keep track of the most recently backfilled block root when
// the node was initialized via checkpoint sync.
func (s *Store) SaveBackfillBlockRoot(ctx context.Context, blockRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveBackfillBlockRoot")
	defer span.End()
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blocksBucket)
		return bucket.Put(backfillBlockRootKey, blockRoot[:])
	})
}

// HighestRootsBelowSlot returns roots from the database slot index from the highest slot below the input slot.
// The slot value at the beginning of the return list is the slot where the roots were found. This is helpful so that
// calling code can make decisions based on the slot without resolving the blocks to discover their slot (for instance
// checking which root is canonical in fork choice, which operates purely on roots,
// then if no canonical block is found, continuing to search through lower slots).
func (s *Store) HighestRootsBelowSlot(ctx context.Context, slot types.Slot) (fs types.Slot, roots [][32]byte, err error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HighestRootsBelowSlot")
	defer span.End()

	sk := bytesutil.Uint64ToBytesBigEndian(uint64(slot))
	err = s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blockSlotIndicesBucket)
		c := bkt.Cursor()
		// The documentation for Seek says:
		// "If the key does not exist then the next key is used. If no keys follow, a nil key is returned."
		seekPast := func(ic *bolt.Cursor, k []byte) ([]byte, []byte) {
			ik, iv := ic.Seek(k)
			// So if there are slots in the index higher than the requested slot, sl will be equal to the key that is
			// one higher than the value we want. If the slot argument is higher than the highest value in the index,
			// we'll get a nil value for `sl`. In that case we'll go backwards from Cursor.Last().
			if ik == nil {
				return ic.Last()
			}
			return ik, iv
		}
		// re loop condition: when .Prev() rewinds past the beginning off the collection, the loop will terminate,
		// because `sl` will be nil. If we don't find a value for `root` before iteration ends,
		// `root` will be the zero value, in which case this function will return the genesis block.
		for sl, r := seekPast(c, sk); sl != nil; sl, r = c.Prev() {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if r == nil {
				continue
			}
			fs = bytesutil.BytesToSlotBigEndian(sl)
			// Iterating through the index using .Prev will move from higher to lower, so the first key we find behind
			// the requested slot must be the highest block below that slot.
			if slot > fs {
				roots, err = splitRoots(r)
				if err != nil {
					return errors.Wrapf(err, "error parsing packed roots %#x", r)
				}
				return nil
			}
		}
		return nil
	})
	if err != nil {
		return 0, nil, err
	}
	if len(roots) == 0 || (len(roots) == 1 && roots[0] == params.BeaconConfig().ZeroHash) {
		gr, err := s.GenesisBlockRoot(ctx)
		return 0, [][32]byte{gr}, err
	}

	return fs, roots, nil
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
		// IF the fee recipient is not found in the standard fee recipient bucket, then
		// check the registration bucket. The fee recipient may be there.
		// This is to resolve imcompatility until we fully migrate to the registration bucket.
		if addr == nil {
			bkt = tx.Bucket(registrationBucket)
			enc := bkt.Get(bytesutil.Uint64ToBytesBigEndian(uint64(id)))
			if enc == nil {
				return errors.Wrapf(ErrNotFoundFeeRecipient, "validator id %d", id)
			}
			reg := &ethpb.ValidatorRegistrationV1{}
			if err := decode(ctx, enc, reg); err != nil {
				return err
			}
			addr = reg.FeeRecipient
		}
		return nil
	})
	return common.BytesToAddress(addr), err
}

// SaveFeeRecipientsByValidatorIDs saves the fee recipients for validator ids.
// Error is returned if `ids` and `recipients` are not the same length.
func (s *Store) SaveFeeRecipientsByValidatorIDs(ctx context.Context, ids []types.ValidatorIndex, feeRecipients []common.Address) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveFeeRecipientByValidatorID")
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

// RegistrationByValidatorID returns the validator registration object for a validator id.
// `ErrNotFoundFeeRecipient` is returned if the validator id is not found.
func (s *Store) RegistrationByValidatorID(ctx context.Context, id types.ValidatorIndex) (*ethpb.ValidatorRegistrationV1, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.RegistrationByValidatorID")
	defer span.End()
	reg := &ethpb.ValidatorRegistrationV1{}
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(registrationBucket)
		enc := bkt.Get(bytesutil.Uint64ToBytesBigEndian(uint64(id)))
		if enc == nil {
			return errors.Wrapf(ErrNotFoundFeeRecipient, "validator id %d", id)
		}
		return decode(ctx, enc, reg)
	})
	return reg, err
}

// SaveRegistrationsByValidatorIDs saves the validator registrations for validator ids.
// Error is returned if `ids` and `registrations` are not the same length.
func (s *Store) SaveRegistrationsByValidatorIDs(ctx context.Context, ids []types.ValidatorIndex, regs []*ethpb.ValidatorRegistrationV1) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveRegistrationsByValidatorIDs")
	defer span.End()

	if len(ids) != len(regs) {
		return errors.New("ids and registrations must be the same length")
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(registrationBucket)
		for i, id := range ids {
			enc, err := encode(ctx, regs[i])
			if err != nil {
				return err
			}
			if err := bkt.Put(bytesutil.Uint64ToBytesBigEndian(uint64(id)), enc); err != nil {
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
	ctx, span := trace.StartSpan(ctx, "BeaconDB.blockRootsBySlotRange")
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
func blockRootsBySlot(ctx context.Context, tx *bolt.Tx, slot types.Slot) ([][32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.blockRootsBySlot")
	defer span.End()

	bkt := tx.Bucket(blockSlotIndicesBucket)
	key := bytesutil.SlotToBytesBigEndian(slot)
	c := bkt.Cursor()
	k, v := c.Seek(key)
	if k != nil && bytes.Equal(k, key) {
		r, err := splitRoots(v)
		if err != nil {
			return nil, errors.Wrapf(err, "corrupt value in block slot index for slot=%d", slot)
		}
		return r, nil
	}
	return [][32]byte{}, nil
}

// createBlockIndicesFromBlock takes in a beacon block and returns
// a map of bolt DB index buckets corresponding to each particular key for indices for
// data, such as (shard indices bucket -> shard 5).
func createBlockIndicesFromBlock(ctx context.Context, block interfaces.BeaconBlock) map[string][]byte {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.createBlockIndicesFromBlock")
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
	buckets = append(buckets, blockParentRootIndicesBucket)
	parentRoot := block.ParentRoot()
	indices = append(indices, parentRoot[:])

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
	ctx, span := trace.StartSpan(ctx, "BeaconDB.createBlockIndicesFromFilters")
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
func unmarshalBlock(_ context.Context, enc []byte) (interfaces.SignedBeaconBlock, error) {
	var err error
	enc, err = snappy.Decode(nil, enc)
	if err != nil {
		return nil, errors.Wrap(err, "could not snappy decode block")
	}
	var rawBlock ssz.Unmarshaler
	switch {
	case hasAltairKey(enc):
		// Marshal block bytes to altair beacon block.
		rawBlock = &ethpb.SignedBeaconBlockAltair{}
		if err := rawBlock.UnmarshalSSZ(enc[len(altairKey):]); err != nil {
			return nil, errors.Wrap(err, "could not unmarshal Altair block")
		}
	case hasBellatrixKey(enc):
		rawBlock = &ethpb.SignedBeaconBlockBellatrix{}
		if err := rawBlock.UnmarshalSSZ(enc[len(bellatrixKey):]); err != nil {
			return nil, errors.Wrap(err, "could not unmarshal Bellatrix block")
		}
	case hasBellatrixBlindKey(enc):
		rawBlock = &ethpb.SignedBlindedBeaconBlockBellatrix{}
		if err := rawBlock.UnmarshalSSZ(enc[len(bellatrixBlindKey):]); err != nil {
			return nil, errors.Wrap(err, "could not unmarshal blinded Bellatrix block")
		}
	default:
		// Marshal block bytes to phase 0 beacon block.
		rawBlock = &ethpb.SignedBeaconBlock{}
		if err := rawBlock.UnmarshalSSZ(enc); err != nil {
			return nil, errors.Wrap(err, "could not unmarshal Phase0 block")
		}
	}
	return blocks.NewSignedBeaconBlock(rawBlock)
}

// marshal versioned beacon block from struct type down to bytes.
func marshalBlock(_ context.Context, blk interfaces.SignedBeaconBlock) ([]byte, error) {
	var encodedBlock []byte
	var err error
	blockToSave := blk
	if features.Get().EnableOnlyBlindedBeaconBlocks {
		blindedBlock, err := blk.ToBlinded()
		switch {
		case errors.Is(err, blocks.ErrUnsupportedVersion):
			encodedBlock, err = blk.MarshalSSZ()
			if err != nil {
				return nil, errors.Wrap(err, "could not marshal non-blinded block")
			}
		case err != nil:
			return nil, errors.Wrap(err, "could not convert block to blinded format")
		default:
			encodedBlock, err = blindedBlock.MarshalSSZ()
			if err != nil {
				return nil, errors.Wrap(err, "could not marshal blinded block")
			}
			blockToSave = blindedBlock
		}
	} else {
		encodedBlock, err = blk.MarshalSSZ()
		if err != nil {
			return nil, err
		}
	}
	switch blockToSave.Version() {
	case version.Bellatrix:
		if blockToSave.IsBlinded() {
			return snappy.Encode(nil, append(bellatrixBlindKey, encodedBlock...)), nil
		}
		return snappy.Encode(nil, append(bellatrixKey, encodedBlock...)), nil
	case version.Altair:
		return snappy.Encode(nil, append(altairKey, encodedBlock...)), nil
	case version.Phase0:
		return snappy.Encode(nil, encodedBlock), nil
	default:
		return nil, errors.New("Unknown block version")
	}
}
