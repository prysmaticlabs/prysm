package db

import (
	"errors"
	"fmt"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func createBlock(enc []byte) (*types.Block, error) {
	protoBlock := &pb.BeaconBlock{}
	err := proto.Unmarshal(enc, protoBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal encoding: %v", err)
	}

	block := types.NewBlock(protoBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate a block from the encoding: %v", err)
	}

	return block, nil
}

// GetBlock accepts a block hash and returns the corresponding block.
// Returns nil if the block does not exist.
func (db *BeaconDB) GetBlock(hash [32]byte) (*types.Block, error) {
	var block *types.Block
	err := db.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(blockBucket)

		enc := b.Get(hash[:])
		if enc == nil {
			return nil
		}

		var err error
		block, err = createBlock(enc)
		return err
	})

	return block, err
}

// HasBlock accepts a block hash and returns true if the block does not exist.
func (db *BeaconDB) HasBlock(hash [32]byte) bool {
	hasBlock := false
	// #nosec G104
	_ = db.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(blockBucket)

		hasBlock = b.Get(hash[:]) != nil

		return nil
	})

	return hasBlock
}

// SaveBlock accepts a block and writes it to disk.
func (db *BeaconDB) SaveBlock(block *types.Block) error {
	hash, err := block.Hash()
	if err != nil {
		return fmt.Errorf("failed to hash block: %v", err)
	}
	enc, err := block.Marshal()
	if err != nil {
		return fmt.Errorf("failed to encode block: %v", err)
	}

	return db.update(func(tx *bolt.Tx) error {
		b := tx.Bucket(blockBucket)

		return b.Put(hash[:], enc)
	})
}

// GetChainHead returns the head of the main chain.
func (db *BeaconDB) GetChainHead() (*types.Block, error) {
	var block *types.Block

	err := db.view(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)
		mainChain := tx.Bucket(mainChainBucket)
		blockBkt := tx.Bucket(blockBucket)

		height := chainInfo.Get(mainChainHeightKey)
		if height == nil {
			return errors.New("unable to determinechain height")
		}

		blockhash := mainChain.Get(height)
		if blockhash == nil {
			return fmt.Errorf("hash at the current height not found: %d", height)
		}

		enc := blockBkt.Get(blockhash)
		if enc == nil {
			return fmt.Errorf("block not found: %x", blockhash)
		}

		var err error
		block, err = createBlock(enc)

		return err
	})

	return block, err
}

// UpdateChainHead atomically updates the head of the chain as well as the corresponding state changes
// Including a new crystallized state is optional.
func (db *BeaconDB) UpdateChainHead(block *types.Block, aState *types.ActiveState, cState *types.CrystallizedState) error {
	blockhash, err := block.Hash()
	if err != nil {
		return fmt.Errorf("unable to get the block hash: %v", err)
	}

	aStateEnc, err := aState.Marshal()
	if err != nil {
		return fmt.Errorf("unable to encode the active state: %v", err)
	}

	var cStateEnc []byte
	if cState != nil {
		cStateEnc, err = cState.Marshal()
		if err != nil {
			return fmt.Errorf("unable to encode the crystallized state: %v", err)
		}
	}

	slotBinary := encodeSlotNumber(block.SlotNumber())

	return db.update(func(tx *bolt.Tx) error {
		blockBucket := tx.Bucket(blockBucket)
		chainInfo := tx.Bucket(chainInfoBucket)
		mainChain := tx.Bucket(mainChainBucket)

		if blockBucket.Get(blockhash[:]) == nil {
			return fmt.Errorf("expected block %#x to have already been saved before updating head: %v", blockhash, err)
		}

		if err := mainChain.Put(slotBinary, blockhash[:]); err != nil {
			return fmt.Errorf("failed to include the block in the main chain bucket: %v", err)
		}

		if err := chainInfo.Put(mainChainHeightKey, slotBinary); err != nil {
			return fmt.Errorf("failed to record the block as the head of the main chain: %v", err)
		}

		if err := chainInfo.Put(aStateLookupKey, aStateEnc); err != nil {
			return fmt.Errorf("failed to save active state as canonical: %v", err)
		}

		if cStateEnc != nil {
			if err := chainInfo.Put(cStateLookupKey, cStateEnc); err != nil {
				return fmt.Errorf("failed to save crystallized state as canonical: %v", err)
			}
		}

		return nil
	})
}

// GetBlockBySlot accepts a slot number and returns the corresponding block in the main chain.
// Returns nil if a block was not recorded for the given slot.
func (db *BeaconDB) GetBlockBySlot(slot uint64) (*types.Block, error) {
	var block *types.Block
	slotEnc := encodeSlotNumber(slot)

	err := db.view(func(tx *bolt.Tx) error {
		mainChain := tx.Bucket(mainChainBucket)
		blockBkt := tx.Bucket(blockBucket)

		blockhash := mainChain.Get(slotEnc)
		if blockhash == nil {
			return nil
		}

		enc := blockBkt.Get(blockhash)
		if enc == nil {
			return fmt.Errorf("block not found: %x", blockhash)
		}

	var err error
		block, err = createBlock(enc)
		return err
	})

	return block, err
}

// GetGenesisTime returns the timestamp for the genesis block
func (db *BeaconDB) GetGenesisTime() (time.Time, error) {
	genesis, err := db.GetBlockBySlot(0)
	if err != nil {
		return time.Time{}, fmt.Errorf("could not get genesis block: %v", err)
	}
	if genesis == nil {
		return time.Time{}, fmt.Errorf("genesis block not found: %v", err)
	}

	genesisTime, err := genesis.Timestamp()
	if err != nil {
		return time.Time{}, fmt.Errorf("could not get genesis timestamp: %v", err)
	}

	return genesisTime, nil
}
