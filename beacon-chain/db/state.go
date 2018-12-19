package db

import (
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/types"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// InitializeState creates an initial genesis state for the beacon
// node using a set of genesis validators.
func (db *BeaconDB) InitializeState(genesisValidatorRegistry []*pb.ValidatorRecord) error {
	beaconState, err := types.NewGenesisBeaconState(genesisValidatorRegistry)
	if err != nil {
		return err
	}

	// #nosec G104
	stateHash, _ := beaconState.Hash()
	genesisBlock := b.NewGenesisBlock(stateHash[:])
	// #nosec G104
	blockHash, _ := b.Hash(genesisBlock)
	// #nosec G104
	blockEnc, _ := proto.Marshal(genesisBlock)
	// #nosec G104
	stateEnc, _ := beaconState.Marshal()
	zeroBinary := encodeSlotNumber(0)

	return db.update(func(tx *bolt.Tx) error {
		blockBkt := tx.Bucket(blockBucket)
		mainChain := tx.Bucket(mainChainBucket)
		chainInfo := tx.Bucket(chainInfoBucket)

		if err := chainInfo.Put(mainChainHeightKey, zeroBinary); err != nil {
			return fmt.Errorf("failed to record block height: %v", err)
		}

		if err := mainChain.Put(zeroBinary, blockHash[:]); err != nil {
			return fmt.Errorf("failed to record block hash: %v", err)
		}

		if err := blockBkt.Put(blockHash[:], blockEnc); err != nil {
			return err
		}

		if err := chainInfo.Put(stateLookupKey, stateEnc); err != nil {
			return err
		}

		return nil
	})
}

// GetState fetches the canonical beacon chain's state from the DB.
func (db *BeaconDB) GetState() (*types.BeaconState, error) {
	var beaconState *types.BeaconState
	err := db.view(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)
		enc := chainInfo.Get(stateLookupKey)
		if enc == nil {
			return nil
		}

		var err error
		beaconState, err = createState(enc)
		return err
	})

	return beaconState, err
}

// SaveState updates the beacon chain state.
func (db *BeaconDB) SaveState(beaconState *types.BeaconState) error {
	return db.update(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)
		beaconStateEnc, err := beaconState.Marshal()
		if err != nil {
			return err
		}
		return chainInfo.Put(stateLookupKey, beaconStateEnc)
	})
}

// GetUnfinalizedBlockState fetches an unfinalized block's
// active and crystallized state pair.
func (db *BeaconDB) GetUnfinalizedBlockState(stateRoot [32]byte) (*types.BeaconState, error) {
	var beaconState *types.BeaconState
	err := db.view(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)
		encState := chainInfo.Get(stateRoot[:])
		if encState == nil {
			return nil
		}

		var err error
		beaconState, err = createState(encState)
		return err
	})
	return beaconState, err
}

// SaveUnfinalizedBlockState persists the associated state
// for a given unfinalized block.
func (db *BeaconDB) SaveUnfinalizedBlockState(beaconState *types.BeaconState) error {
	stateHash, err := beaconState.Hash()
	if err != nil {
		return fmt.Errorf("unable to hash the beacon state: %v", err)
	}
	beaconStateEnc, err := beaconState.Marshal()
	if err != nil {
		return fmt.Errorf("unable to encode the beacon state: %v", err)
	}

	return db.update(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)
		if err := chainInfo.Put(stateHash[:], beaconStateEnc); err != nil {
			return fmt.Errorf("failed to save beacon state: %v", err)
		}
		return nil
	})
}

func createState(enc []byte) (*types.BeaconState, error) {
	protoState := &pb.BeaconState{}
	err := proto.Unmarshal(enc, protoState)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal encoding: %v", err)
	}

	state := types.NewBeaconState(protoState)
	return state, nil
}
