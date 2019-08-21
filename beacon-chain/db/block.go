package db

import (
	"bytes"
	"context"
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"go.opencensus.io/trace"
)

var (
	badBlockCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bad_blocks",
		Help: "Number of bad, blacklisted blocks received",
	})
	blockCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "beacon_block_cache_miss",
		Help: "The number of block requests that aren't present in the cache.",
	})
	blockCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "beacon_block_cache_hit",
		Help: "The number of block requests that are present in the cache.",
	})
	blockCacheSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacon_block_cache_size",
		Help: "The number of beacon blocks in the block cache",
	})
)

func createBlock(enc []byte) (*ethpb.BeaconBlock, error) {
	protoBlock := &ethpb.BeaconBlock{}
	err := proto.Unmarshal(enc, protoBlock)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoding")
	}
	return protoBlock, nil
}

// Block accepts a block root and returns the corresponding block.
// Returns nil if the block does not exist.
func (db *BeaconDB) Block(_ context.Context, root [32]byte) (*ethpb.BeaconBlock, error) {
	return db.BlockDeprecated(root)
}

// BlockDeprecated accepts a block root and returns the corresponding block.
// Returns nil if the block does not exist.
// DEPRECATED: Use Block.
func (db *BeaconDB) BlockDeprecated(root [32]byte) (*ethpb.BeaconBlock, error) {
	db.blocksLock.RLock()

	// Return block from cache if it exists
	if blk, exists := db.blocks[root]; exists && blk != nil {
		defer db.blocksLock.RUnlock()
		blockCacheHit.Inc()
		return db.blocks[root], nil
	}

	var block *ethpb.BeaconBlock
	err := db.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blockBucket)

		enc := bucket.Get(root[:])
		if enc == nil {
			return nil
		}

		var err error
		block, err = createBlock(enc)
		return err
	})

	db.blocksLock.RUnlock()
	db.blocksLock.Lock()
	defer db.blocksLock.Unlock()
	// Save block to the cache since it wasn't there before.
	if block != nil {
		db.blocks[root] = block
		blockCacheMiss.Inc()
		blockCacheSize.Set(float64(len(db.blocks)))
	}

	return block, err
}

// HasBlock accepts a block root and returns true if the block does not exist.
func (db *BeaconDB) HasBlock(_ context.Context, root [32]byte) bool {
	db.blocksLock.RLock()
	defer db.blocksLock.RUnlock()

	// Check the cache first to see if block exists.
	if _, exists := db.blocks[root]; exists {
		return true
	}

	hasBlock := false
	// #nosec G104
	_ = db.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blockBucket)

		hasBlock = bucket.Get(root[:]) != nil

		return nil
	})

	return hasBlock
}

// HasBlockDeprecated accepts a block root and returns true if the block does not exist.
func (db *BeaconDB) HasBlockDeprecated(root [32]byte) bool {
	return db.HasBlock(context.Background(), root)
}

// HeadBlock is not implemented.
// DEPRECATED: Do not use. Not implemented.
func (db *BeaconDB) HeadBlock(_ context.Context) (*ethpb.BeaconBlock, error) {
	return nil, errors.New("not implemented")
}

// IsEvilBlockHash determines if a certain block root has been blacklisted
// due to failing to process core state transitions.
func (db *BeaconDB) IsEvilBlockHash(root [32]byte) bool {
	db.badBlocksLock.Lock()
	defer db.badBlocksLock.Unlock()
	if db.badBlockHashes != nil {
		return db.badBlockHashes[root]
	}
	db.badBlockHashes = make(map[[32]byte]bool)
	return false
}

// MarkEvilBlockHash makes a block hash as tainted because it corresponds
// to a block which fails core state transition processing.
func (db *BeaconDB) MarkEvilBlockHash(root [32]byte) {
	db.badBlocksLock.Lock()
	defer db.badBlocksLock.Unlock()
	if db.badBlockHashes == nil {
		db.badBlockHashes = make(map[[32]byte]bool)
	}
	db.badBlockHashes[root] = true
	badBlockCount.Inc()
}

// SaveHeadBlockRoot is not implemented.
func (db *BeaconDB) SaveHeadBlockRoot(_ context.Context, root [32]byte) error {
	return errors.New("not implemented")
}

// SaveBlocks in db.
func (db *BeaconDB) SaveBlocks(ctx context.Context, blocks []*ethpb.BeaconBlock) error {
	for _, blk := range blocks {
		if err := db.SaveBlock(ctx, blk); err != nil {
			return err
		}
	}
	return nil
}

// SaveBlock in db.
func (db *BeaconDB) SaveBlock(_ context.Context, block *ethpb.BeaconBlock) error {
	return db.SaveBlockDeprecated(block)
}

// SaveBlockDeprecated accepts a block and writes it to disk.
// DEPRECATED: Use SaveBlock.
func (db *BeaconDB) SaveBlockDeprecated(block *ethpb.BeaconBlock) error {
	db.blocksLock.Lock()
	defer db.blocksLock.Unlock()

	signingRoot, err := ssz.SigningRoot(block)
	if err != nil {
		return errors.Wrap(err, "failed to tree hash header")
	}

	// Skip saving block to DB if it exists in the cache.
	if blk, exists := db.blocks[signingRoot]; exists && blk != nil {
		return nil
	}
	// Save it to the cache if it's not in the cache.
	db.blocks[signingRoot] = block
	blockCacheSize.Set(float64(len(db.blocks)))

	enc, err := proto.Marshal(block)
	if err != nil {
		return errors.Wrap(err, "failed to encode block")
	}
	slotRootBinary := encodeSlotNumberRoot(block.Slot, signingRoot)

	if block.Slot > db.highestBlockSlot {
		db.highestBlockSlot = block.Slot
	}

	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blockBucket)
		if err := bucket.Put(slotRootBinary, enc); err != nil {
			return errors.Wrap(err, "failed to include the block in the main chain bucket")
		}
		return bucket.Put(signingRoot[:], enc)
	})
}

// DeleteBlock deletes a block using the slot and its root as keys in their respective buckets.
func (db *BeaconDB) DeleteBlock(_ context.Context, hash [32]byte) error {
	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blockBucket)
		return bucket.Delete(hash[:])
	})
}

// DeleteBlockDeprecated deletes a block using the slot and its root as keys in their respective buckets.
func (db *BeaconDB) DeleteBlockDeprecated(block *ethpb.BeaconBlock) error {
	db.blocksLock.Lock()
	defer db.blocksLock.Unlock()

	signingRoot, err := ssz.SigningRoot(block)
	if err != nil {
		return errors.Wrap(err, "failed to tree hash block")
	}

	// Delete the block from the cache.
	delete(db.blocks, signingRoot)
	blockCacheSize.Set(float64(len(db.blocks)))

	slotRootBinary := encodeSlotNumberRoot(block.Slot, signingRoot)

	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blockBucket)
		if err := bucket.Delete(slotRootBinary); err != nil {
			return errors.Wrap(err, "failed to include the block in the main chain bucket")
		}
		return bucket.Delete(signingRoot[:])
	})
}

// SaveJustifiedBlock saves the last justified block from canonical chain to DB.
func (db *BeaconDB) SaveJustifiedBlock(block *ethpb.BeaconBlock) error {
	return db.update(func(tx *bolt.Tx) error {
		enc, err := proto.Marshal(block)
		if err != nil {
			return errors.Wrap(err, "failed to encode block")
		}
		chainInfo := tx.Bucket(chainInfoBucket)
		return chainInfo.Put(justifiedBlockLookupKey, enc)
	})
}

// SaveFinalizedBlock saves the last finalized block from canonical chain to DB.
func (db *BeaconDB) SaveFinalizedBlock(block *ethpb.BeaconBlock) error {
	return db.update(func(tx *bolt.Tx) error {
		enc, err := proto.Marshal(block)
		if err != nil {
			return errors.Wrap(err, "failed to encode block")
		}
		chainInfo := tx.Bucket(chainInfoBucket)
		return chainInfo.Put(finalizedBlockLookupKey, enc)
	})
}

// JustifiedBlock retrieves the justified block from the db.
func (db *BeaconDB) JustifiedBlock() (*ethpb.BeaconBlock, error) {
	var block *ethpb.BeaconBlock
	err := db.view(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)
		encBlock := chainInfo.Get(justifiedBlockLookupKey)
		if encBlock == nil {
			return errors.New("no justified block saved")
		}

		var err error
		block, err = createBlock(encBlock)
		return err
	})
	return block, err
}

// FinalizedBlock retrieves the finalized block from the db.
func (db *BeaconDB) FinalizedBlock() (*ethpb.BeaconBlock, error) {
	var block *ethpb.BeaconBlock
	err := db.view(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)
		encBlock := chainInfo.Get(finalizedBlockLookupKey)
		if encBlock == nil {
			return errors.New("no finalized block saved")
		}

		var err error
		block, err = createBlock(encBlock)
		return err
	})
	return block, err
}

// ChainHead returns the head of the main chain.
func (db *BeaconDB) ChainHead() (*ethpb.BeaconBlock, error) {
	var block *ethpb.BeaconBlock
	err := db.view(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)
		blockBkt := tx.Bucket(blockBucket)

		height := chainInfo.Get(mainChainHeightKey)
		if height == nil {
			return errors.New("unable to determine chain height")
		}

		blockRoot := chainInfo.Get(canonicalHeadKey)
		if blockRoot == nil {
			return fmt.Errorf("root at the current height not found: %d", height)
		}

		enc := blockBkt.Get(blockRoot)
		if enc == nil {
			return fmt.Errorf("block not found: %x", blockRoot)
		}

		var err error
		block, err = createBlock(enc)
		return err
	})

	return block, err
}

// UpdateChainHead atomically updates the head of the chain as well as the corresponding state changes
// Including a new state is optional.
func (db *BeaconDB) UpdateChainHead(ctx context.Context, block *ethpb.BeaconBlock, beaconState *pb.BeaconState) error {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.db.UpdateChainHead")
	defer span.End()

	blockRoot, err := ssz.SigningRoot(block)
	if err != nil {
		return errors.Wrap(err, "unable to determine block signing root")
	}

	slotBinary := encodeSlotNumber(block.Slot)
	if block.Slot > db.highestBlockSlot {
		db.highestBlockSlot = block.Slot
	}

	if err := db.SaveStateDeprecated(ctx, beaconState); err != nil {
		return errors.Wrap(err, "failed to save beacon state as canonical")
	}

	blockEnc, err := proto.Marshal(block)
	if err != nil {
		return err
	}

	return db.update(func(tx *bolt.Tx) error {
		blockBucket := tx.Bucket(blockBucket)
		chainInfo := tx.Bucket(chainInfoBucket)
		mainChainBucket := tx.Bucket(mainChainBucket)

		if blockBucket.Get(blockRoot[:]) == nil {
			return fmt.Errorf("expected block %#x to have already been saved before updating head", blockRoot)
		}

		if err := chainInfo.Put(mainChainHeightKey, slotBinary); err != nil {
			return err
		}

		if err := mainChainBucket.Put(slotBinary, blockEnc); err != nil {
			return err
		}

		if err := chainInfo.Put(canonicalHeadKey, blockRoot[:]); err != nil {
			return errors.Wrap(err, "failed to record the block as the head of the main chain")
		}

		return nil
	})
}

// CanonicalBlockBySlot accepts a slot number and returns the corresponding canonical block.
func (db *BeaconDB) CanonicalBlockBySlot(ctx context.Context, slot uint64) (*ethpb.BeaconBlock, error) {
	_, span := trace.StartSpan(ctx, "BeaconDB.CanonicalBlockBySlot")
	defer span.End()
	span.AddAttributes(trace.Int64Attribute("slot", int64(slot)))

	var block *ethpb.BeaconBlock
	slotEnc := encodeSlotNumber(slot)

	err := db.view(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(mainChainBucket)
		blockEnc := bkt.Get(slotEnc)
		var err error
		if blockEnc != nil {
			block, err = createBlock(blockEnc)
		}
		return err
	})

	return block, err
}

// BlocksBySlot accepts a slot number and returns the corresponding blocks in the db.
// Returns empty list if no blocks were recorded for the given slot.
func (db *BeaconDB) BlocksBySlot(ctx context.Context, slot uint64) ([]*ethpb.BeaconBlock, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	_, span := trace.StartSpan(ctx, "BeaconDB.BlocksBySlot")
	defer span.End()
	span.AddAttributes(trace.Int64Attribute("slot", int64(slot)))

	blocks := []*ethpb.BeaconBlock{}
	slotEnc := encodeSlotNumber(slot)

	err := db.view(func(tx *bolt.Tx) error {
		c := tx.Bucket(blockBucket).Cursor()

		var err error
		prefix := slotEnc
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			block, err := createBlock(v)
			if err != nil {
				return err
			}
			blocks = append(blocks, block)
		}

		return err
	})

	return blocks, err
}

// HighestBlockSlot returns the in-memory value for the highest block we've
// seen in the database.
func (db *BeaconDB) HighestBlockSlot() uint64 {
	return db.highestBlockSlot
}

// ClearBlockCache prunes the block cache. This is used on every new finalized epoch.
func (db *BeaconDB) ClearBlockCache() {
	db.blocksLock.Lock()
	defer db.blocksLock.Unlock()
	db.blocks = make(map[[32]byte]*ethpb.BeaconBlock)
	blockCacheSize.Set(float64(len(db.blocks)))
}

// BlockRoots retrieves a list of beacon block roots by filter criteria.
// DEPRECATED: Not implemented at all. Use github.com/prysmaticlabs/prysm/db/kv
func (db *BeaconDB) BlockRoots(ctx context.Context, f *filters.QueryFilter) ([][]byte, error) {
	return nil, errors.New("not implemented")
}

// Blocks retrieves a list of beacon blocks by filter criteria.
// DEPRECATED: Not implemented at all. Use github.com/prysmaticlabs/prysm/db/kv
func (db *BeaconDB) Blocks(ctx context.Context, f *filters.QueryFilter) ([]*ethpb.BeaconBlock, error) {
	return nil, errors.New("not implemented")
}
