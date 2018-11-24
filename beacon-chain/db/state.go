package db

import (
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// InitializeState ...
func (db *BeaconDB) InitializeState(genesisValidators []*pb.ValidatorRecord) error {
	beaconState, err := types.NewGenesisBeaconState(genesisValidators)
	if err != nil {
		return err
	}

	// #nosec G104
	stateHash, _ := beaconState.Hash()
	genesisBlock := types.NewGenesisBlock(stateHash)
	// #nosec G104
	blockHash, _ := genesisBlock.Hash()
	// #nosec G104
	blockEnc, _ := genesisBlock.Marshal()
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

// GetState --
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
// func (db *BeaconDB) GetUnfinalizedBlockState(
// 	aStateRoot [32]byte,
// 	cStateRoot [32]byte,
// ) (*types.ActiveState, *types.CrystallizedState, error) {
// 	var aState *types.ActiveState
// 	var cState *types.CrystallizedState
// 	err := db.view(func(tx *bolt.Tx) error {
// 		chainInfo := tx.Bucket(chainInfoBucket)

// 		encActive := chainInfo.Get(aStateRoot[:])
// 		if encActive == nil {
// 			return nil
// 		}
// 		encCrystallized := chainInfo.Get(cStateRoot[:])
// 		if encCrystallized == nil {
// 			return nil
// 		}

// 		var err error
// 		aState, err = createActiveState(encActive)
// 		if err != nil {
// 			return err
// 		}
// 		cState, err = createCrystallizedState(encCrystallized)
// 		return err
// 	})

// 	return aState, cState, err
// }

// SaveUnfinalizedBlockState persists the associated crystallized and
// active state pair for a given unfinalized block.
// func (db *BeaconDB) SaveUnfinalizedBlockState(aState *types.ActiveState, cState *types.CrystallizedState) error {
// 	aStateHash, err := aState.Hash()
// 	if err != nil {
// 		return fmt.Errorf("unable to hash the active state: %v", err)
// 	}
// 	aStateEnc, err := aState.Marshal()
// 	if err != nil {
// 		return fmt.Errorf("unable to encode the active state: %v", err)
// 	}

// 	var cStateEnc []byte
// 	var cStateHash [32]byte
// 	if cState != nil {
// 		cStateHash, err = cState.Hash()
// 		if err != nil {
// 			return fmt.Errorf("unable to hash the crystallized state: %v", err)
// 		}
// 		cStateEnc, err = cState.Marshal()
// 		if err != nil {
// 			return fmt.Errorf("unable to encode the crystallized state: %v", err)
// 		}
// 	}

// 	return db.update(func(tx *bolt.Tx) error {
// 		chainInfo := tx.Bucket(chainInfoBucket)
// 		if err := chainInfo.Put(aStateHash[:], aStateEnc); err != nil {
// 			return fmt.Errorf("failed to save active state as canonical: %v", err)
// 		}

// 		if cStateEnc != nil {
// 			if err := chainInfo.Put(cStateHash[:], cStateEnc); err != nil {
// 				return fmt.Errorf("failed to save crystallized state as canonical: %v", err)
// 			}
// 		}
// 		return nil
// 	})
// }

func createState(enc []byte) (*types.BeaconState, error) {
	protoState := &pb.BeaconState{}
	err := proto.Unmarshal(enc, protoState)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal encoding: %v", err)
	}

	state := types.NewBeaconState(protoState)
	return state, nil
}
