package db

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
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
func (db *BeaconDB) InitializeState(genesisTime uint64, deposits []*pb.Deposit, eth1Data *pb.Eth1Data) error {
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

	db.currentState = beaconState

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

// State fetches the canonical beacon chain's state from the DB.
func (db *BeaconDB) State(ctx context.Context) (*pb.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.State")
	defer span.End()

	db.stateLock.RLock()
	defer db.stateLock.RUnlock()

	if db.currentState != nil {
		if cachedState, ok := proto.Clone(db.currentState).(*pb.BeaconState); ok {
			return cachedState, nil
		}
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
		return err
	})

	return beaconState, err
}

// SaveState updates the beacon chain state.
func (db *BeaconDB) SaveState(beaconState *pb.BeaconState) error {
	db.stateLock.Lock()
	defer db.stateLock.Unlock()

	// Clone to prevent mutations of the cached copy
	currentState, ok := proto.Clone(beaconState).(*pb.BeaconState)
	if !ok {
		return errors.New("could not clone beacon state")
	}
	db.currentState = currentState
	return db.update(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)

		prevState := chainInfo.Get(stateLookupKey)
		if prevState != nil {
			prevStatePb := &pb.BeaconState{}
			if err := proto.Unmarshal(prevState, prevStatePb); err != nil {
				return err
			}
			if prevStatePb.Slot >= beaconState.Slot {
				log.WithField(
					"prevStateSlot",
					prevStatePb.Slot,
				).WithField(
					"newStateSlot",
					beaconState.Slot,
				).Warn("Current saved state has a slot number greater or equal to the state attempted to be saved")
			}
		}

		beaconStateEnc, err := proto.Marshal(beaconState)
		if err != nil {
			return err
		}
		stateBytes.Set(float64(len(beaconStateEnc)))
		return chainInfo.Put(stateLookupKey, beaconStateEnc)
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
	finalizedSlot := beaconState.FinalizedEpoch * params.BeaconConfig().SlotsPerEpoch
	// Delete historical states if we are saving a new finalized state.
	if err := db.deleteHistoricalStates(finalizedSlot); err != nil {
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
func (db *BeaconDB) SaveHistoricalState(beaconState *pb.BeaconState) error {
	slotSinceGenesis := beaconState.Slot - params.BeaconConfig().GenesisSlot

	// Do not save state, if slot diff is not
	// a power of 2.
	if slotSinceGenesis%params.BeaconConfig().SlotsPerEpoch != 0 {
		return nil
	}

	slotBinary := encodeSlotNumber(slotSinceGenesis)
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

// SaveCurrentAndFinalizedState saves the state as both the current and last finalized state.
func (db *BeaconDB) SaveCurrentAndFinalizedState(beaconState *pb.BeaconState) error {
	// Clone to prevent mutations of the cached copy
	currentState, ok := proto.Clone(beaconState).(*pb.BeaconState)
	if !ok {
		return errors.New("could not clone beacon state")
	}

	if err := db.SaveState(beaconState); err != nil {
		return err
	}

	db.currentState = currentState
	finalizedSlot := beaconState.FinalizedEpoch * params.BeaconConfig().SlotsPerEpoch
	// Delete historical states if we are saving a new finalized state.
	if err := db.deleteHistoricalStates(finalizedSlot); err != nil {
		return err
	}
	return db.update(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)
		beaconStateEnc, err := proto.Marshal(beaconState)
		if err != nil {
			return err
		}
		// Putting in historical state.
		if err := chainInfo.Put(stateLookupKey, beaconStateEnc); err != nil {
			return err
		}

		return chainInfo.Put(finalizedStateLookupKey, beaconStateEnc)
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

// HistoricalStateFromSlot retrieves the closest historical state to a slot.
func (db *BeaconDB) HistoricalStateFromSlot(slot uint64) (*pb.BeaconState, error) {
	state, err := db.FinalizedState()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve finalized state %v", err)
	}
	slotSinceGenesis := slot - params.BeaconConfig().GenesisSlot
	if slotSinceGenesis%params.BeaconConfig().SlotsPerEpoch != 0 {
		return state, nil
	}
	var beaconState *pb.BeaconState

	err = db.view(func(tx *bolt.Tx) error {
		var err error
		var highestStateSlot uint64
		histStateKey := make([]byte, 32)

		chainInfo := tx.Bucket(chainInfoBucket)
		histState := tx.Bucket(histStateBucket)
		hsCursor := histState.Cursor()

		for k, v := hsCursor.First(); k != nil; k, v = hsCursor.Next() {
			slotNumber := decodeToSlotNumber(k)
			if slotNumber == slotSinceGenesis {
				highestStateSlot = slotNumber
				histStateKey = v
				break
			}
		}
		// If no state exists send the finalized state to be unencoded.
		if highestStateSlot == 0 {
			encState := chainInfo.Get(finalizedStateLookupKey)
			if encState == nil {
				return errors.New("no finalized state saved")
			}
			beaconState, err = createState(encState)
			return err
		}

		// retrieve the stored historical state.
		encState := chainInfo.Get(histStateKey)
		if encState == nil {
			return errors.New("no historical state saved")
		}
		beaconState, err = createState(encState)
		return err
	})
	return beaconState, err
}

func createState(enc []byte) (*pb.BeaconState, error) {
	protoState := &pb.BeaconState{}
	err := proto.Unmarshal(enc, protoState)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal encoding: %v", err)
	}
	return protoState, nil
}

// GenesisTime returns the genesis timestamp for the state.
func (db *BeaconDB) GenesisTime(ctx context.Context) (time.Time, error) {
	state, err := db.State(ctx)
	if err != nil {
		return time.Time{}, fmt.Errorf("could not retrieve state: %v", err)
	}
	if state == nil {
		return time.Time{}, fmt.Errorf("state not found: %v", err)
	}
	genesisTime := time.Unix(int64(state.GenesisTime), int64(0))
	return genesisTime, nil
}

func (db *BeaconDB) deleteHistoricalStates(slot uint64) error {
	return db.update(func(tx *bolt.Tx) error {
		histState := tx.Bucket(histStateBucket)
		chainInfo := tx.Bucket(chainInfoBucket)
		hsCursor := histState.Cursor()
		slotSinceGenesis := slot - params.BeaconConfig().GenesisSlot

		for k, v := hsCursor.First(); k != nil; k, v = hsCursor.Next() {
			keySlotNumber := decodeToSlotNumber(k)
			if keySlotNumber <= slotSinceGenesis {
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
