package db

import (
	"errors"
	"fmt"

	"github.com/prysmaticlabs/prysm/shared/ssz"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
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

// SaveBlock accepts a block and writes it to disk.
func (db *BeaconDB) SaveBlock(block *pb.BeaconBlock) error {
	root, err := ssz.TreeHash(block)
	if err != nil {
		return fmt.Errorf("failed to tree hash block: %v", err)
	}
	enc, err := proto.Marshal(block)
	if err != nil {
		return fmt.Errorf("failed to encode block: %v", err)
	}

	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blockBucket)

		return bucket.Put(root[:], enc)
	})
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
// Including a new crystallized state is optional.
func (db *BeaconDB) UpdateChainHead(block *pb.BeaconBlock, beaconState *pb.BeaconState) error {
	blockRoot, err := ssz.TreeHash(block)
	if err != nil {
		return fmt.Errorf("unable to tree hash block: %v", err)
	}

	beaconStateEnc, err := proto.Marshal(beaconState)
	if err != nil {
		return fmt.Errorf("unable to encode beacon state: %v", err)
	}

	slotBinary := encodeSlotNumber(block.GetSlot())

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

		if err := chainInfo.Put(stateLookupKey, beaconStateEnc); err != nil {
			return fmt.Errorf("failed to save beacon state as canonical: %v", err)
		}
		return nil
	})
}

// BlockBySlot accepts a slot number and returns the corresponding block in the main chain.
// Returns nil if a block was not recorded for the given slot.
func (db *BeaconDB) BlockBySlot(slot uint64) (*pb.BeaconBlock, error) {
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
			return fmt.Errorf("block not found: %#x", blockRoot)
		}

		var err error
		block, err = createBlock(enc)
		return err
	})

	return block, err
}
