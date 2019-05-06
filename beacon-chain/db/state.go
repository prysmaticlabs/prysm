package db

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"go.opencensus.io/trace"
)

var (
	stateBytes = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacondb_state_size_bytes",
		Help: "The protobuf encoded size of the last saved state in the beaconDB",
	})
)

// InitializeState creates an initial genesis state for the beacon
// node using a set of genesis validators.
func (db *BeaconDB) InitializeState(ctx context.Context, genesisTime uint64, deposits []*pb.Deposit, eth1Data *pb.Eth1Data) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.InitializeState")
	defer span.End()

	beaconState, err := state.GenesisBeaconState(deposits, genesisTime, eth1Data)
	if err != nil {
		return err
	}

	// #nosec G104
	stateEnc, _ := proto.Marshal(beaconState)
	stateHash := hashutil.Hash(stateEnc)
	genesisBlock := b.NewGenesisBlock(stateHash[:])
	// #nosec G104
	blockRoot, _ := hashutil.HashBeaconBlock(genesisBlock)
	// #nosec G104
	blockEnc, _ := proto.Marshal(genesisBlock)
	zeroBinary := encodeSlotNumber(0)

	db.serializedState = stateEnc
	db.stateHash = stateHash

	if err := db.SaveState(ctx, beaconState); err != nil {
		return err
	}

	return db.update(func(tx *bolt.Tx) error {
		blockBkt := tx.Bucket(blockBucket)
		validatorBkt := tx.Bucket(validatorBucket)
		mainChain := tx.Bucket(mainChainBucket)
		chainInfo := tx.Bucket(chainInfoBucket)

		if err := chainInfo.Put(mainChainHeightKey, zeroBinary); err != nil {
			return fmt.Errorf("failed to record block height: %v", err)
		}

		if err := mainChain.Put(zeroBinary, blockRoot[:]); err != nil {
			return fmt.Errorf("failed to record block hash: %v", err)
		}

		if err := blockBkt.Put(blockRoot[:], blockEnc); err != nil {
			return err
		}

		for i, validator := range beaconState.ValidatorRegistry {
			h := hashutil.Hash(validator.Pubkey)
			buf := make([]byte, binary.MaxVarintLen64)
			n := binary.PutUvarint(buf, uint64(i))
			if err := validatorBkt.Put(h[:], buf[:n]); err != nil {
				return err
			}
		}

		// Putting in finalized state.
		if err := chainInfo.Put(finalizedStateLookupKey, stateEnc); err != nil {
			return err
		}

		return chainInfo.Put(stateLookupKey, stateEnc)
	})
}

// HeadState fetches the canonical beacon chain's head state from the DB.
func (db *BeaconDB) HeadState(ctx context.Context) (*pb.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HeadState")
	defer span.End()

	ctx, lockSpan := trace.StartSpan(ctx, "BeaconDB.stateLock.Lock")
	db.stateLock.RLock()
	defer db.stateLock.RUnlock()
	lockSpan.End()

	// Return in-memory cached state, if available.
	if db.serializedState != nil {
		_, span := trace.StartSpan(ctx, "proto.Marshal")
		defer span.End()
		newState := &pb.BeaconState{}
		// For each READ we unmarshal the serialized state into a new state struct and return that.
		if err := proto.Unmarshal(db.serializedState, newState); err != nil {
			return nil, err
		}
		return newState, nil
	}

	var beaconState *pb.BeaconState
	err := db.view(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)
		enc := chainInfo.Get(stateLookupKey)
		if enc == nil {
			return nil
		}

		var err error
		beaconState, err = createState(enc)

		if beaconState != nil && beaconState.Slot > db.highestBlockSlot {
			db.highestBlockSlot = beaconState.Slot
		}
		db.serializedState = enc
		db.stateHash = hashutil.Hash(enc)

		return err
	})

	return beaconState, err
}

// HeadStateRoot returns the root of the current state from the db.
func (db *BeaconDB) HeadStateRoot() [32]byte {
	return db.stateHash
}

// SaveState updates the beacon chain state.
func (db *BeaconDB) SaveState(ctx context.Context, beaconState *pb.BeaconState) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveState")
	defer span.End()

	ctx, lockSpan := trace.StartSpan(ctx, "BeaconDB.stateLock.Lock")
	db.stateLock.Lock()
	defer db.stateLock.Unlock()
	lockSpan.End()

	// For each WRITE of the state, we serialize the inputted state and save it in memory,
	// and then the state is saved to disk.
	enc, err := proto.Marshal(beaconState)
	if err != nil {
		return err
	}
	stateHash := hashutil.Hash(enc)
	tempState := &pb.BeaconState{}
	tempState.ValidatorRegistry = beaconState.ValidatorRegistry

	copy(db.validatorBalances, beaconState.ValidatorBalances)
	db.validatorRegistry = proto.Clone(tempState).(*pb.BeaconState).ValidatorRegistry
	db.serializedState = enc
	db.stateHash = stateHash

	if err := db.SaveHistoricalState(ctx, beaconState); err != nil {
		return err
	}

	return db.update(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)

		stateBytes.Set(float64(len(enc)))
		reportStateMetrics(beaconState)
		return chainInfo.Put(stateLookupKey, enc)
	})
}

// SaveJustifiedState saves the last justified state in the db.
func (db *BeaconDB) SaveJustifiedState(beaconState *pb.BeaconState) error {
	return db.update(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)
		beaconStateEnc, err := proto.Marshal(beaconState)
		if err != nil {
			return err
		}
		return chainInfo.Put(justifiedStateLookupKey, beaconStateEnc)
	})
}

// SaveFinalizedState saves the last finalized state in the db.
func (db *BeaconDB) SaveFinalizedState(beaconState *pb.BeaconState) error {

	// Delete historical states if we are saving a new finalized state.
	if err := db.deleteHistoricalStates(beaconState.Slot); err != nil {
		return err
	}
	return db.update(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)
		beaconStateEnc, err := proto.Marshal(beaconState)
		if err != nil {
			return err
		}
		return chainInfo.Put(finalizedStateLookupKey, beaconStateEnc)
	})
}

// SaveHistoricalState saves the last finalized state in the db.
func (db *BeaconDB) SaveHistoricalState(ctx context.Context, beaconState *pb.BeaconState) error {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.db.SaveHistoricalState")
	defer span.End()

	slotBinary := encodeSlotNumber(beaconState.Slot)
	stateHash, err := hashutil.HashProto(beaconState)
	if err != nil {
		return err
	}

	return db.update(func(tx *bolt.Tx) error {
		histState := tx.Bucket(histStateBucket)
		chainInfo := tx.Bucket(chainInfoBucket)
		if err := histState.Put(slotBinary, stateHash[:]); err != nil {
			return err
		}
		beaconStateEnc, err := proto.Marshal(beaconState)
		if err != nil {
			return err
		}
		return chainInfo.Put(stateHash[:], beaconStateEnc)
	})
}

// JustifiedState retrieves the justified state from the db.
func (db *BeaconDB) JustifiedState() (*pb.BeaconState, error) {
	var beaconState *pb.BeaconState
	err := db.view(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)
		encState := chainInfo.Get(justifiedStateLookupKey)
		if encState == nil {
			return errors.New("no justified state saved")
		}

		var err error
		beaconState, err = createState(encState)
		return err
	})
	return beaconState, err
}

// FinalizedState retrieves the finalized state from the db.
func (db *BeaconDB) FinalizedState() (*pb.BeaconState, error) {
	var beaconState *pb.BeaconState
	err := db.view(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)
		encState := chainInfo.Get(finalizedStateLookupKey)
		if encState == nil {
			return errors.New("no finalized state saved")
		}

		var err error
		beaconState, err = createState(encState)
		return err
	})
	return beaconState, err
}

// HistoricalStateFromSlot retrieves the state that is closest to the input slot,
// while being smaller than or equal to the input slot.
func (db *BeaconDB) HistoricalStateFromSlot(ctx context.Context, slot uint64) (*pb.BeaconState, error) {
	_, span := trace.StartSpan(ctx, "BeaconDB.HistoricalStateFromSlot")
	defer span.End()
	span.AddAttributes(trace.Int64Attribute("slotSinceGenesis", int64(slot)))
	var beaconState *pb.BeaconState
	err := db.view(func(tx *bolt.Tx) error {
		var err error
		var highestStateSlot uint64
		var stateExists bool
		histStateKey := make([]byte, 32)

		chainInfo := tx.Bucket(chainInfoBucket)
		histState := tx.Bucket(histStateBucket)
		hsCursor := histState.Cursor()

		for k, v := hsCursor.First(); k != nil; k, v = hsCursor.Next() {
			slotNumber := decodeToSlotNumber(k)
			if slotNumber == slot {
				stateExists = true
				highestStateSlot = slotNumber
				histStateKey = v
				break
			}
		}

		// If no historical state exists, retrieve and decode the finalized state.
		if !stateExists {
			for k, v := hsCursor.First(); k != nil; k, v = hsCursor.Next() {
				slotNumber := decodeToSlotNumber(k)
				// find the state with slot closest to the requested slot
				if slotNumber > highestStateSlot && slotNumber <= slot {
					stateExists = true
					highestStateSlot = slotNumber
					histStateKey = v
				}
			}

			if !stateExists {
				return errors.New("no historical states saved in db")
			}
		}

		// If historical state exists, retrieve and decode it.
		encState := chainInfo.Get(histStateKey)
		if encState == nil {
			return errors.New("no historical state saved")
		}
		beaconState, err = createState(encState)
		return err
	})
	return beaconState, err
}

// ValidatorRegistry fetches the current validator registry stored in state.
func (db *BeaconDB) ValidatorRegistry(ctx context.Context) ([]*pb.Validator, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.ValidatorRegistry")
	defer span.End()

	db.stateLock.RLock()
	defer db.stateLock.RUnlock()

	// Return in-memory cached state, if available.
	if db.validatorRegistry != nil {
		_, span := trace.StartSpan(ctx, "proto.Clone.ValidatorRegistry")
		defer span.End()
		tempState := &pb.BeaconState{
			ValidatorRegistry: db.validatorRegistry,
		}
		newState := proto.Clone(tempState).(*pb.BeaconState)
		return newState.ValidatorRegistry, nil
	}

	var beaconState *pb.BeaconState
	err := db.view(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)
		enc := chainInfo.Get(stateLookupKey)
		if enc == nil {
			return nil
		}

		var err error
		beaconState, err = createState(enc)
		if beaconState != nil && beaconState.Slot > db.highestBlockSlot {
			db.highestBlockSlot = beaconState.Slot
		}
		return err
	})

	return beaconState.ValidatorRegistry, err
}

// ValidatorFromState fetches the validator with the desired index from the cached registry.
func (db *BeaconDB) ValidatorFromState(ctx context.Context, index uint64) (*pb.Validator, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.ValidatorFromState")
	defer span.End()

	db.stateLock.RLock()
	defer db.stateLock.RUnlock()

	if db.validatorRegistry != nil {
		// return error if it's an invalid validator index.
		if index >= uint64(len(db.validatorRegistry)) {
			return nil, fmt.Errorf("invalid validator index %d", index)
		}
		validator := proto.Clone(db.validatorRegistry[index]).(*pb.Validator)
		return validator, nil
	}

	var beaconState *pb.BeaconState
	err := db.view(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)
		enc := chainInfo.Get(stateLookupKey)
		if enc == nil {
			return nil
		}

		var err error
		beaconState, err = createState(enc)
		if beaconState != nil && beaconState.Slot > db.highestBlockSlot {
			db.highestBlockSlot = beaconState.Slot
		}
		return err
	})

	// return error if it's an invalid validator index.
	if index >= uint64(len(db.validatorRegistry)) {
		return nil, fmt.Errorf("invalid validator index %d", index)
	}

	return beaconState.ValidatorRegistry[index], err
}

// ValidatorBalances fetches the current validator balances stored in state.
func (db *BeaconDB) ValidatorBalances(ctx context.Context) ([]uint64, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.ValidatorBalances")
	defer span.End()

	db.stateLock.RLock()
	defer db.stateLock.RUnlock()

	// Return in-memory cached state, if available.
	if db.validatorBalances != nil {
		_, span := trace.StartSpan(ctx, "BeaconDB.Copy.Balances")
		defer span.End()
		newBalances := make([]uint64, len(db.validatorBalances))
		copy(newBalances, db.validatorBalances)
		return newBalances, nil
	}

	var beaconState *pb.BeaconState
	err := db.view(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)
		enc := chainInfo.Get(stateLookupKey)
		if enc == nil {
			return nil
		}

		var err error
		beaconState, err = createState(enc)
		if beaconState != nil && beaconState.Slot > db.highestBlockSlot {
			db.highestBlockSlot = beaconState.Slot
		}
		return err
	})

	return beaconState.ValidatorBalances, err
}

func createState(enc []byte) (*pb.BeaconState, error) {
	protoState := &pb.BeaconState{}
	err := proto.Unmarshal(enc, protoState)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal encoding: %v", err)
	}
	return protoState, nil
}

func (db *BeaconDB) deleteHistoricalStates(slot uint64) error {
	if featureconfig.FeatureConfig().DisableHistoricalStatePruning {
		return nil
	}
	return db.update(func(tx *bolt.Tx) error {
		histState := tx.Bucket(histStateBucket)
		chainInfo := tx.Bucket(chainInfoBucket)
		hsCursor := histState.Cursor()

		for k, v := hsCursor.First(); k != nil; k, v = hsCursor.Next() {
			keySlotNumber := decodeToSlotNumber(k)
			if keySlotNumber < slot {
				if err := histState.Delete(k); err != nil {
					return err
				}
				if err := chainInfo.Delete(v); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
