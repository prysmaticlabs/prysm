package db

import (
	"bytes"
	"context"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/go-ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
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
func (db *BeaconDB) Block(root [32]byte) (*ethpb.BeaconBlock, error) {
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
func (db *BeaconDB) HasBlock(root [32]byte) bool {
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

// SaveBlock accepts a block and writes it to disk.
func (db *BeaconDB) SaveBlock(block *ethpb.BeaconBlock) error {
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

	parentRoot := bytesutil.ToBytes32(block.ParentRoot)
	db.blockChildrenRoots[parentRoot] = append(db.blockChildrenRoots[parentRoot], signingRoot[:])

	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blockBucket)
		if err := bucket.Put(slotRootBinary, enc); err != nil {
			return errors.Wrap(err, "failed to include the block in the main chain bucket")
		}
		return bucket.Put(signingRoot[:], enc)
	})
}

// DeleteBlock deletes a block using the slot and its root as keys in their respective buckets.
func (db *BeaconDB) DeleteBlock(block *ethpb.BeaconBlock) error {
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

// SaveHeadBlockRoot saves head block root.
func (db *BeaconDB) SaveHeadBlockRoot(root []byte) error {
	return db.update(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)
		return chainInfo.Put(canonicalHeadKey, root[:])
	})
}

// ChildrenBlocksFromParent retrieves the children block roots from a parent block root.
// It has an optional filter slot to filter out children blocks below input slot.
func (db *BeaconDB) ChildrenBlocksFromParent(parentRoot []byte, slot uint64) ([][]byte, error) {
	childrenRoots := db.blockChildrenRoots[bytesutil.ToBytes32(parentRoot)]
	i := 0
	for _, r := range childrenRoots {
		var b *ethpb.BeaconBlock
		err := db.view(func(tx *bolt.Tx) error {
			bucket := tx.Bucket(blockBucket)

			enc := bucket.Get(r)
			if enc == nil {
				return nil
			}

			var err error
			b, err = createBlock(enc)

			if b.Slot > slot {
				childrenRoots[i] = r
				i++
			}
			return err
		})
		if err != nil {
			return nil, err
		}
	}

	return childrenRoots[:i], nil
}
