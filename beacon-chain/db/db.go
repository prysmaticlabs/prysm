package db

import (
	"errors"
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// DB is a wrapper for BoltDB with getter/setter methods specifically for the Beacon Cain.
type DB struct {
	bolt *bolt.DB
}

// NewDB instantiates a new database
func NewDB(db *bolt.DB) *DB {
	_ = db.Update(func(tx *bolt.Tx) error {
		tx.CreateBucket(blockBucket)
		tx.CreateBucket(attestationBucket)
		tx.CreateBucket(mainChainBucket)
		tx.CreateBucket(chainInfoBucket)
		return nil
	})
	return &DB{ bolt: db }
}

// Close releases all boltDB resources.
// All transactions must be closed before calling this.
func (db *DB) Close() error {
	return db.bolt.Close()
}

func createBlockFromEncoding(enc []byte) (*types.Block, error) {
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
func (db *DB) GetBlock(hash [32]byte) (*types.Block, error) {
	var block *types.Block
	err := db.bolt.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(blockBucket)

		enc := b.Get(hash[:])
		if enc == nil {
			return nil
		}

		var err error
		block, err = createBlockFromEncoding(enc)
		return err
	})

	return block, err
}

// HasBlock accepts a block hash and returns true if the block does not exist.
func (db *DB) HasBlock(hash [32]byte) bool {
	var hasBlock = false
	_ = db.bolt.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(blockBucket)

		hasBlock = b.Get(hash[:]) != nil

		return nil
	})

	return hasBlock
}

// SaveBlock accepts a block and writes it to disk.
func (db *DB) SaveBlock(block *types.Block) error {
	hash, err := block.Hash()
	if err != nil {
		return fmt.Errorf("failed to hash block: %v", err)
	}
	enc, err := block.Marshal()
	if err != nil {
		return fmt.Errorf("failed to encode block: %v", err)
	}

	return db.bolt.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(blockBucket)

		return b.Put(hash[:], enc)
	})
}

func (db *DB) SaveBlockAndAttestations(block *types.Block) error {
	hash, err := block.Hash()
	if err != nil {
		return fmt.Errorf("failed to hash block: %v", err)
	}
	enc, err := block.Marshal()
	if err != nil {
		return fmt.Errorf("failed to encode block: %v", err)
	}

	encodings := make([][]byte, len(block.Attestations()))
	hashes := make([][]byte, len(block.Attestations()))
	for i, protoA := range block.Attestations() {
		a := types.NewAttestation(protoA)
		aEnc, err := a.Marshal()
		if err != nil {
			return err
		}

		aHash, err := a.Hash()
		if err != nil {
			return err
		}

		encodings[i] = aEnc
		hashes[i] = aHash[:]
	}

	return db.bolt.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(blockBucket)
		a := tx.Bucket(attestationBucket)

		if err := b.Put(hash[:], enc); err != nil {
			return err
		}

		for i := 0; i < len(encodings); i++ {
			if err := a.Put(hashes[i], encodings[i]); err != nil {
				return err
			}
		}

		return nil
	})
}

func (db *DB) HasBlockForSlot(slot uint64) bool {
	slotEnc := encodeSlotNumber(slot)
	blockExists := false
	db.bolt.View(func(tx *bolt.Tx) error {
		mainChain := tx.Bucket(mainChainBucket)
		blockBkt := tx.Bucket(blockBucket)

		blockHash := mainChain.Get(slotEnc)
		if blockHash == nil {
			return nil
		}

		enc := blockBkt.Get(blockHash)
		blockExists = enc != nil

		return nil
	})

	return blockExists
}

// GetBlockBySlot accepts a slot number and returns the corresponding block in the main chain.
// Returns nil if a block was not recorded for the given slot.
func (db *DB) GetBlockBySlot(slot uint64) (*types.Block, error) {
	var block *types.Block
	slotEnc := encodeSlotNumber(slot)

	err := db.bolt.View(func(tx *bolt.Tx) error {
		mainChain := tx.Bucket(mainChainBucket)
		blockBkt := tx.Bucket(blockBucket)

		blockHash := mainChain.Get(slotEnc)
		if blockHash == nil {
			return nil
		}

		enc := blockBkt.Get(blockHash)
		if enc == nil {
			return fmt.Errorf("block not found: %x", blockHash)
		}

		var err error
		block, err = createBlockFromEncoding(enc)
		return err
	})

	return block, err
}

// GetChainTip returns tip (block with the highest slot) of the main chain.
func (db *DB) GetChainTip() (*types.Block, error) {
	var block *types.Block

	err := db.bolt.View(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)
		mainChain := tx.Bucket(mainChainBucket)
		blockBkt := tx.Bucket(blockBucket)

		height := chainInfo.Get(mainChainHeightKey)
		if height == nil {
			return errors.New("unable to determine chain height")
		}

		blockhash := mainChain.Get(height)
		if blockhash == nil {
			return fmt.Errorf("hash for current chain tip not found: %d", height)
		}

		enc := blockBkt.Get(blockhash)
		if enc == nil {
			return fmt.Errorf("block not found: %x", blockhash)
		}

		var err error
		block, err = createBlockFromEncoding(enc)

		return err
	})

	return block, err
}

// RecordChainTip accepts a block and records the block hash as the new tip of the main chain.
// The block's slot must be greater than the current tip's slot.
// Assumes that the block itself was already saved.
func (db *DB) RecordChainTip(block *types.Block, aState *types.ActiveState, cState *types.CrystallizedState) error {
	hash, err := block.Hash()
	if err != nil {
		return fmt.Errorf("failed to hash block: %v", err)
	}

	aStateEnc, err := aState.Marshal()
	if err != nil {
		return fmt.Errorf("failed to encode active state: %v", err)
	}

	var cStateEnc []byte
	if cState != nil {
		cStateEnc, err = cState.Marshal()
		if err != nil {
			return fmt.Errorf("failed to encode crystallized state: %gv", err)
		}
	}

	slotBinary := encodeSlotNumber(block.SlotNumber())

	return db.bolt.Update(func(tx *bolt.Tx) error {
		mainChain := tx.Bucket(mainChainBucket)
		chainInfo := tx.Bucket(chainInfoBucket)

		chainHeight := chainInfo.Get(mainChainHeightKey)
		if chainHeight != nil && block.SlotNumber() <= decodeSlotNumber(chainHeight) {
			return fmt.Errorf("block's slot %d must be greater than the current tip's slot %d", block.SlotNumber(), chainHeight)
		}

		if err := mainChain.Put(slotBinary, hash[:]); err != nil {
			return fmt.Errorf("failed to include the block in the main chain bucket: %v", err)
		}

		if err := chainInfo.Put(mainChainHeightKey, slotBinary); err != nil {
			return fmt.Errorf("failed to record the block as the tip of the main chain: %v", err)
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

// HasInitialState returns true if the genesis state has already been initialized.
func (db *DB) HasInitialState() bool {
	var hasInitialState = false
	_ = db.bolt.View(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)

		hasInitialState = chainInfo.Get(cStateLookupKey) != nil

		return nil
	})

	return hasInitialState
}

// SaveInitialState accepts the genesis block and the initial active/crystallized states and writes them to disk.
func (db *DB) SaveInitialState(block *types.Block, aState *types.ActiveState, cState *types.CrystallizedState) error {
	zeroEnc := encodeSlotNumber(0)
	aStateEnc, err := aState.Marshal()
	if err != nil {
		return err
	}

	cStateEnc, err := cState.Marshal()
	if err != nil {
		return err
	}

	return db.bolt.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(blockBucket)
		mainChain := tx.Bucket(mainChainBucket)
		chainInfo := tx.Bucket(chainInfoBucket)

		h, err := block.Hash()
		if err != nil {
			return err
		}

		enc, err := block.Marshal()
		if err != nil {
			return err
		}

		if err := b.Put(h[:], enc); err != nil {
			return err
		}

		if err := mainChain.Put(zeroEnc, h[:]); err != nil {
			return err
		}

		if err := chainInfo.Put(mainChainHeightKey, zeroEnc); err != nil {
			return err
		}

		if err := chainInfo.Put(aStateLookupKey, aStateEnc); err != nil {
			return err
		}

		if err := chainInfo.Put(cStateLookupKey, cStateEnc); err != nil {
			return err
		}

		return nil
	})
}

// GetActiveState returns the current canonical active state.
func (db *DB) GetActiveState() (*types.ActiveState, error) {
	var state *types.ActiveState
	var err error

	err = db.bolt.View(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)

		enc := chainInfo.Get(aStateLookupKey)
		if enc != nil {
			return fmt.Errorf("active state does not exist on disk")
		}

		stateProto := &pb.ActiveState{}
		err = proto.Unmarshal(enc, stateProto)
		if err != nil {
			return fmt.Errorf("failed to unmarsal active state encoding: %v", err)
		}

		state = types.NewActiveState(stateProto, nil)
		return nil
	})

	return state, err
}

// GetCrystallizedState returns the current canonical crystallized state.
func (db *DB) GetCrystallizedState() (*types.CrystallizedState, error) {
	var state *types.CrystallizedState
	var err error

	err = db.bolt.View(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)

		enc := chainInfo.Get(cStateLookupKey)
		if enc != nil {
			return fmt.Errorf("active state does not exist on disk")
		}

		stateProto := &pb.CrystallizedState{}
		err = proto.Unmarshal(enc, stateProto)
		if err != nil {
			return fmt.Errorf("failed to unmarsal active state encoding: %v", err)
		}

		state = types.NewCrystallizedState(stateProto)
		return nil
	})

	return state, err
}

// TrackSimulatedBlock writes the blockhash of the last simulated block
func (db *DB) TrackSimulatedBlock(block *types.Block) error {
	hash, err := block.Hash()
	if err != nil {
		return err
	}

	return db.bolt.Update(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)

		return chainInfo.Put(lastSimulatedBlockKey, hash[:])
	})
}

// GetLastSimulatedBlock returns the last recorded block sent by the simulator
func (db *DB) GetLastSimulatedBlock() (*types.Block, error) {
	var block *types.Block
	err := db.bolt.View(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)
		blockBucket := tx.Bucket(blockBucket)

		lastBlockHash := chainInfo.Get(lastSimulatedBlockKey)
		if lastBlockHash == nil {
			return fmt.Errorf("failed to find a recently simulated block")
		}

		blockEnc := blockBucket.Get(lastBlockHash)
		if blockEnc == nil {
			return fmt.Errorf("failed to find the block for the recently simulated hash %x", lastBlockHash)
		}

		var err error
		block, err = createBlockFromEncoding(blockEnc)
		if err != nil {
			return fmt.Errorf("failed to unmarshal block from encoding: %v", err)
		}

		return nil
	})

	return block, err
}