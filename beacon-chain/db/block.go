package db

import (
	"errors"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// GetCanonicalBlock fetches the latest head stored in persistent storage.
func (db *BeaconDB) GetCanonicalBlock() (*types.Block, error) {
	bytes, err := db.get(canonicalHeadLookupKey)
	if err != nil {
		return nil, err
	}
	block := &pb.BeaconBlock{}
	if err := proto.Unmarshal(bytes, block); err != nil {
		return nil, fmt.Errorf("cannot unmarshal proto: %v", err)
	}
	return types.NewBlock(block), nil
}

// HasBlock returns true if the block for the given hash exists.
func (db *BeaconDB) HasBlock(blockhash [32]byte) (bool, error) {
	return db.has(blockKey(blockhash))
}

// SaveBlock puts the passed block into the beacon chain db.
func (db *BeaconDB) SaveBlock(block *types.Block) error {
	hash, err := block.Hash()
	if err != nil {
		return err
	}

	key := blockKey(hash)
	encodedState, err := block.Marshal()
	if err != nil {
		return err
	}
	return db.put(key, encodedState)
}

// SaveCanonicalSlotNumber saves the slotnumber and blockhash of a canonical block
// saved in the db. This will alow for canonical blocks to be retrieved from the db
// by using their slotnumber as a key, and then using the retrieved blockhash to get
// the block from the db.
// prefix + slotNumber -> blockhash
// prefix + blockHash -> block
func (db *BeaconDB) SaveCanonicalSlotNumber(slotNumber uint64, hash [32]byte) error {
	return db.put(canonicalBlockKey(slotNumber), hash[:])
}

// SaveCanonicalBlock puts the passed block into the beacon chain db
// and also saves a "latest-head" key mapping to the block in the db.
func (db *BeaconDB) SaveCanonicalBlock(block *types.Block) error {
	enc, err := block.Marshal()
	if err != nil {
		return err
	}

	return db.put(canonicalHeadLookupKey, enc)
}

// GetBlock retrieves a block from the db using its hash.
func (db *BeaconDB) GetBlock(hash [32]byte) (*types.Block, error) {
	key := blockKey(hash)
	has, err := db.has(key)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, errors.New("block not found")
	}
	enc, err := db.get(key)
	if err != nil {
		return nil, err
	}

	block := &pb.BeaconBlock{}

	err = proto.Unmarshal(enc, block)

	return types.NewBlock(block), err
}

// removeBlock removes the block from the db.
func (db *BeaconDB) removeBlock(hash [32]byte) error {
	return db.delete(blockKey(hash))
}

// HasCanonicalBlockForSlot checks the db if the canonical block for
// this slot exists.
func (db *BeaconDB) HasCanonicalBlockForSlot(slotNumber uint64) (bool, error) {
	return db.has(canonicalBlockKey(slotNumber))
}

// GetCanonicalBlockForSlot retrieves the canonical block which is saved in the db
// for that required slot number.
func (db *BeaconDB) GetCanonicalBlockForSlot(slotNumber uint64) (*types.Block, error) {
	enc, err := db.get(canonicalBlockKey(slotNumber))
	if err != nil {
		return nil, err
	}

	var blockhash [32]byte
	copy(blockhash[:], enc)

	block, err := db.GetBlock(blockhash)

	return block, err
}

// GetGenesisTime returns the timestamp for the genesis block
func (db *BeaconDB) GetGenesisTime() (time.Time, error) {
	genesis, err := db.GetCanonicalBlockForSlot(0)
	if err != nil {
		return time.Time{}, fmt.Errorf("Could not get genesis block: %v", err)
	}
	genesisTime, err := genesis.Timestamp()
	if err != nil {
		return time.Time{}, fmt.Errorf("Could not get genesis timestamp: %v", err)
	}

	return genesisTime, nil
}

// GetSimulatedBlock retrieves the last block broadcast by the simulator.
func (db *BeaconDB) GetSimulatedBlock() (*types.Block, error) {
	enc, err := db.get(simulatedBlockKey)
	if err != nil {
		return nil, err
	}

	protoBlock := &pb.BeaconBlock{}
	err = proto.Unmarshal(enc, protoBlock)
	if err != nil {
		return nil, err
	}

	return types.NewBlock(protoBlock), nil
}

// SaveSimulatedBlock saves the last broadcast block to the database.
func (db *BeaconDB) SaveSimulatedBlock(block *types.Block) error {
	enc, err := block.Marshal()
	if err != nil {
		return err
	}

	return db.put(simulatedBlockKey, enc)
}

// HasSimulatedBlock checks if a block was broadcast by the simulator.
func (db *BeaconDB) HasSimulatedBlock() (bool, error) {
	return db.has(simulatedBlockKey)
}
