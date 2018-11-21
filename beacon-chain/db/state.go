package db

import (
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
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

// SaveCrystallizedState updates the crystallized state for initial sync.
func (db *BeaconDB) SaveCrystallizedState(cState *types.CrystallizedState) error {

	return db.update(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)
		cStateEnc, err := cState.Marshal()
		if err != nil {
			return err
		}
		return chainInfo.Put(cStateLookupKey, cStateEnc)
	})
}

// GetUnfinalizedBlockState fetches an unfinalized block's
// active and crystallized state pair.
func (db *BeaconDB) GetUnfinalizedBlockState(
	aStateRoot [32]byte,
	cStateRoot [32]byte,
) (*types.ActiveState, *types.CrystallizedState, error) {
	var aState *types.ActiveState
	var cState *types.CrystallizedState
	err := db.view(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)

		encActive := chainInfo.Get(aStateRoot[:])
		if encActive == nil {
			return nil
		}
		encCrystallized := chainInfo.Get(cStateRoot[:])
		if encCrystallized == nil {
			return nil
		}

		var err error
		aState, err = createActiveState(encActive)
		if err != nil {
			return err
		}
		cState, err = createCrystallizedState(encCrystallized)
		return err
	})

	return aState, cState, err
}

// SaveUnfinalizedBlockState persists the associated crystallized and
// active state pair for a given unfinalized block.
func (db *BeaconDB) SaveUnfinalizedBlockState(aState *types.ActiveState, cState *types.CrystallizedState) error {
	aStateHash, err := aState.Hash()
	if err != nil {
		return fmt.Errorf("unable to hash the active state: %v", err)
	}
	aStateEnc, err := aState.Marshal()
	if err != nil {
		return fmt.Errorf("unable to encode the active state: %v", err)
	}

	var cStateEnc []byte
	var cStateHash [32]byte
	if cState != nil {
		cStateHash, err = cState.Hash()
		if err != nil {
			return fmt.Errorf("unable to hash the crystallized state: %v", err)
		}
		cStateEnc, err = cState.Marshal()
		if err != nil {
			return fmt.Errorf("unable to encode the crystallized state: %v", err)
		}
	}

	return db.update(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)
		if err := chainInfo.Put(aStateHash[:], aStateEnc); err != nil {
			return fmt.Errorf("failed to save active state as canonical: %v", err)
		}

		if cStateEnc != nil {
			if err := chainInfo.Put(cStateHash[:], cStateEnc); err != nil {
				return fmt.Errorf("failed to save crystallized state as canonical: %v", err)
			}
		}
		return nil
	})
}

func createActiveState(enc []byte) (*types.ActiveState, error) {
	protoState := &pb.ActiveState{}
	err := proto.Unmarshal(enc, protoState)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal encoding: %v", err)
	}

	state := types.NewActiveState(protoState)

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
