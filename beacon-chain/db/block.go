package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

var (
	badBlockCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bad_blocks",
		Help: "Number of bad, blacklisted blocks received",
	})
)

func createBlock(enc []byte) (*pb.BeaconBlock, error) {
	protoBlock := &pb.BeaconBlock{}
	err := proto.Unmarshal(enc, protoBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal encoding: %v", err)
	}
	return protoBlock, nil
}

// Block accepts a block root and returns the corresponding block.
// Returns nil if the block does not exist.
func (db *BeaconDB) Block(root [32]byte) (*pb.BeaconBlock, error) {
	var block *pb.BeaconBlock
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

	return block, err
}

// HasBlock accepts a block root and returns true if the block does not exist.
func (db *BeaconDB) HasBlock(root [32]byte) bool {
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
func (db *BeaconDB) SaveBlock(block *pb.BeaconBlock) error {
	root, err := hashutil.HashBeaconBlock(block)
	if err != nil {
		return fmt.Errorf("failed to tree hash block: %v", err)
	}
	enc, err := proto.Marshal(block)
	if err != nil {
		return fmt.Errorf("failed to encode block: %v", err)
	}
	slotBinary := encodeSlotNumber(block.Slot)

	if block.Slot > db.highestBlockSlot {
		db.highestBlockSlot = block.Slot
	}

	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blockBucket)
		mainChain := tx.Bucket(mainChainBucket)
		if err := mainChain.Put(slotBinary, root[:]); err != nil {
			return fmt.Errorf("failed to include the block in the main chain bucket: %v", err)
		}
		return bucket.Put(root[:], enc)
	})
}

// DeleteBlock deletes a block using the slot and its root as keys in their respective buckets.
func (db *BeaconDB) DeleteBlock(block *pb.BeaconBlock) error {
	root, err := hashutil.HashBeaconBlock(block)
	if err != nil {
		return fmt.Errorf("failed to tree hash block: %v", err)
	}
	slotBinary := encodeSlotNumber(block.Slot)

	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blockBucket)
		mainChain := tx.Bucket(mainChainBucket)
		if err := mainChain.Delete(slotBinary); err != nil {
			return fmt.Errorf("failed to include the block in the main chain bucket: %v", err)
		}
		return bucket.Delete(root[:])
	})
}

// SaveJustifiedBlock saves the last justified block from canonical chain to DB.
func (db *BeaconDB) SaveJustifiedBlock(block *pb.BeaconBlock) error {
	return db.update(func(tx *bolt.Tx) error {
		enc, err := proto.Marshal(block)
		if err != nil {
			return fmt.Errorf("failed to encode block: %v", err)
		}
		chainInfo := tx.Bucket(chainInfoBucket)
		return chainInfo.Put(justifiedBlockLookupKey, enc)
	})
}

// SaveFinalizedBlock saves the last finalized block from canonical chain to DB.
func (db *BeaconDB) SaveFinalizedBlock(block *pb.BeaconBlock) error {
	return db.update(func(tx *bolt.Tx) error {
		enc, err := proto.Marshal(block)
		if err != nil {
			return fmt.Errorf("failed to encode block: %v", err)
		}
		chainInfo := tx.Bucket(chainInfoBucket)
		return chainInfo.Put(finalizedBlockLookupKey, enc)
	})
}

// JustifiedBlock retrieves the justified block from the db.
func (db *BeaconDB) JustifiedBlock() (*pb.BeaconBlock, error) {
	var block *pb.BeaconBlock
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
func (db *BeaconDB) FinalizedBlock() (*pb.BeaconBlock, error) {
	var block *pb.BeaconBlock
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
func (db *BeaconDB) ChainHead() (*pb.BeaconBlock, error) {
	var block *pb.BeaconBlock
	err := db.view(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)
		mainChain := tx.Bucket(mainChainBucket)
		blockBkt := tx.Bucket(blockBucket)

		height := chainInfo.Get(mainChainHeightKey)
		if height == nil {
			return errors.New("unable to determine chain height")
		}

		blockRoot := mainChain.Get(height)
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
func (db *BeaconDB) UpdateChainHead(ctx context.Context, block *pb.BeaconBlock, beaconState *pb.BeaconState) error {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.db.UpdateChainHead")
	defer span.End()

	blockRoot, err := hashutil.HashBeaconBlock(block)
	if err != nil {
		return fmt.Errorf("unable to tree hash block: %v", err)
	}

	slotBinary := encodeSlotNumber(block.Slot)
	if block.Slot > db.highestBlockSlot {
		db.highestBlockSlot = block.Slot
	}

	if err := db.SaveState(ctx, beaconState); err != nil {
		return fmt.Errorf("failed to save beacon state as canonical: %v", err)
	}

	return db.update(func(tx *bolt.Tx) error {
		blockBucket := tx.Bucket(blockBucket)
		chainInfo := tx.Bucket(chainInfoBucket)
		mainChain := tx.Bucket(mainChainBucket)

		if blockBucket.Get(blockRoot[:]) == nil {
			return fmt.Errorf("expected block %#x to have already been saved before updating head: %v", blockRoot, err)
		}

		if err := mainChain.Put(slotBinary, blockRoot[:]); err != nil {
			return fmt.Errorf("failed to include the block in the main chain bucket: %v", err)
		}

		if err := chainInfo.Put(mainChainHeightKey, slotBinary); err != nil {
			return fmt.Errorf("failed to record the block as the head of the main chain: %v", err)
		}

		return nil
	})
}

// BlockBySlot accepts a slot number and returns the corresponding block in the main chain.
// Returns nil if a block was not recorded for the given slot.
func (db *BeaconDB) BlockBySlot(ctx context.Context, slot uint64) (*pb.BeaconBlock, error) {
	_, span := trace.StartSpan(ctx, "BeaconDB.BlockBySlot")
	defer span.End()
	span.AddAttributes(trace.Int64Attribute("slot", int64(slot-params.BeaconConfig().GenesisSlot)))

	var block *pb.BeaconBlock
	slotEnc := encodeSlotNumber(slot)

	err := db.view(func(tx *bolt.Tx) error {
		mainChain := tx.Bucket(mainChainBucket)
		blockBkt := tx.Bucket(blockBucket)

		blockRoot := mainChain.Get(slotEnc)
		if blockRoot == nil {
			return nil
		}

		enc := blockBkt.Get(blockRoot)
		if enc == nil {
			return nil
		}

		var err error
		block, err = createBlock(enc)
		return err
	})

	return block, err
}

// HighestBlockSlot returns the in-memory value for the highest block we've
// seen in the database.
func (db *BeaconDB) HighestBlockSlot() uint64 {
	return db.highestBlockSlot
}
