package db

import (
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// InitializeState ...
func (db *BeaconDB) InitializeState(genesisValidators []*pb.ValidatorRecord) error {
	aState := types.NewGenesisActiveState()
	cState, err := types.NewGenesisCrystallizedState(genesisValidators)
	if err != nil {
		return err
	}

	// #nosec G104
	activeStateHash, _ := aState.Hash()
	// #nosec G104
	crystallizedStateHash, _ := cState.Hash()

	genesisBlock := types.NewGenesisBlock(activeStateHash, crystallizedStateHash)
	// #nosec G104
	blockhash, _ := genesisBlock.Hash()

	// #nosec G104
	blockEnc, _ := genesisBlock.Marshal()
	// #nosec G104
	aStateEnc, _ := aState.Marshal()
	// #nosec G104
	cStateEnc, _ := cState.Marshal()

	zeroBinary := encodeSlotNumber(0)

	return db.update(func(tx *bolt.Tx) error {
		blockBkt := tx.Bucket(blockBucket)
		mainChain := tx.Bucket(mainChainBucket)
		chainInfo := tx.Bucket(chainInfoBucket)

		if err := chainInfo.Put(mainChainHeightKey, zeroBinary); err != nil {
			return fmt.Errorf("failed to record block height: %v", err)
		}

		if err := mainChain.Put(zeroBinary, blockhash[:]); err != nil {
			return fmt.Errorf("failed to record block hash: %v", err)
		}

		if err := blockBkt.Put(blockhash[:], blockEnc); err != nil {
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

// GetActiveState fetches the current active state.
func (db *BeaconDB) GetActiveState() (*types.ActiveState, error) {
	var aState *types.ActiveState
	err := db.view(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)

		enc := chainInfo.Get(aStateLookupKey)
		if enc == nil {
			return nil
		}

		var err error
		aState, err = createActiveState(enc)
		return err
	})

	return aState, err
}

// GetCrystallizedState fetches the current active state.
func (db *BeaconDB) GetCrystallizedState() (*types.CrystallizedState, error) {
	var cState *types.CrystallizedState
	err := db.view(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)

		enc := chainInfo.Get(cStateLookupKey)
		if enc == nil {
			return nil
		}

		var err error
		cState, err = createCrystallizedState(enc)
		return err
	})

	return cState, err
}

func createActiveState(enc []byte) (*types.ActiveState, error) {
	protoState := &pb.ActiveState{}
	err := proto.Unmarshal(enc, protoState)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal encoding: %v", err)
	}

	state := types.NewActiveState(protoState, map[[32]byte]*utils.VoteCache{})

	return state, nil
}

func createCrystallizedState(enc []byte) (*types.CrystallizedState, error) {
	protoState := &pb.CrystallizedState{}
	err := proto.Unmarshal(enc, protoState)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal encoding: %v", err)
	}

	state := types.NewCrystallizedState(protoState)

	return state, nil
}
