package db

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/go-ssz"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
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
func (db *BeaconDB) InitializeState(ctx context.Context, genesisTime uint64, deposits []*ethpb.Deposit, eth1Data *ethpb.Eth1Data) error {
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
	blockRoot, _ := ssz.SigningRoot(genesisBlock)
	// #nosec G104
	blockEnc, _ := proto.Marshal(genesisBlock)
	zeroBinary := encodeSlotNumberRoot(0, blockRoot)

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
			return errors.Wrap(err, "failed to record block height")
		}

		if err := mainChain.Put(zeroBinary, blockEnc); err != nil {
			return errors.Wrap(err, "failed to record block hash")
		}

		if err := chainInfo.Put(canonicalHeadKey, blockRoot[:]); err != nil {
			return errors.Wrap(err, "failed to record block as canonical")
		}

		if err := blockBkt.Put(blockRoot[:], blockEnc); err != nil {
			return err
		}

		for i, validator := range beaconState.Validators {
			h := hashutil.Hash(validator.PublicKey)
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
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	ctx, span := trace.StartSpan(ctx, "BeaconDB.HeadState")
	defer span.End()

	ctx, lockSpan := trace.StartSpan(ctx, "BeaconDB.stateLock.Lock")
	db.stateLock.RLock()
	defer db.stateLock.RUnlock()
	lockSpan.End()

	var beaconState *pb.BeaconState
	var err error
	err = db.view(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)
		root := chainInfo.Get(canonicalHeadKey)
		if root == nil {
			return nil
		}

		beaconState, err = db.ForkChoiceState(ctx, root)
		return err
	})
	if beaconState != nil {
		return beaconState, nil
	}

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

	err = db.view(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)
		enc := chainInfo.Get(stateLookupKey)
		if enc == nil {
			return nil
		}

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
	tempState.Validators = beaconState.Validators

	copy(db.validatorBalances, beaconState.Balances)
	db.validatorRegistry = proto.Clone(tempState).(*pb.BeaconState).Validators
	db.serializedState = enc
	db.stateHash = stateHash

	if beaconState.LatestBlockHeader != nil {
		blockRoot, err := ssz.SigningRoot(beaconState.LatestBlockHeader)
		if err != nil {
			return err
		}

		if err := db.SaveHistoricalState(ctx, beaconState, blockRoot); err != nil {
			return err
		}
	}

	return db.update(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)

		stateBytes.Set(float64(len(enc)))
		reportStateMetrics(beaconState)
		return chainInfo.Put(stateLookupKey, enc)
	})
}

// SaveHistoricalState saves the last finalized state in the db.
func (db *BeaconDB) SaveHistoricalState(ctx context.Context, beaconState *pb.BeaconState, blockRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.db.SaveHistoricalState")
	defer span.End()

	slotRootBinary := encodeSlotNumberRoot(beaconState.Slot, blockRoot)
	stateHash, err := hashutil.HashProto(beaconState)
	if err != nil {
		return err
	}

	return db.update(func(tx *bolt.Tx) error {
		histState := tx.Bucket(histStateBucket)
		chainInfo := tx.Bucket(chainInfoBucket)
		if err := histState.Put(slotRootBinary, stateHash[:]); err != nil {
			return err
		}
		beaconStateEnc, err := proto.Marshal(beaconState)
		if err != nil {
			return err
		}
		return chainInfo.Put(stateHash[:], beaconStateEnc)
	})
}

// HistoricalStateFromSlot retrieves the state that is closest to the input slot,
// while being smaller than or equal to the input slot.
func (db *BeaconDB) HistoricalStateFromSlot(ctx context.Context, slot uint64, blockRoot [32]byte) (*pb.BeaconState, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	_, span := trace.StartSpan(ctx, "BeaconDB.HistoricalStateFromSlot")
	defer span.End()
	span.AddAttributes(trace.Int64Attribute("slot", int64(slot)))
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
			slotBinary := k[:8]
			blockRootBinary := k[8:]
			slotNumber := decodeToSlotNumber(slotBinary)

			if slotNumber == slot && bytes.Equal(blockRootBinary, blockRoot[:]) {
				stateExists = true
				highestStateSlot = slotNumber
				histStateKey = v
				break
			}
		}

		// If no historical state exists, retrieve and decode the finalized state.
		if !stateExists {
			for k, v := hsCursor.First(); k != nil; k, v = hsCursor.Next() {
				slotBinary := k[:8]
				slotNumber := decodeToSlotNumber(slotBinary)
				// find the state with slot closest to the requested slot
				if slotNumber >= highestStateSlot && slotNumber <= slot {
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

// Validators fetches the current validator registry stored in state.
func (db *BeaconDB) Validators(ctx context.Context) ([]*ethpb.Validator, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.Validators")
	defer span.End()

	db.stateLock.RLock()
	defer db.stateLock.RUnlock()

	// Return in-memory cached state, if available.
	if db.validatorRegistry != nil {
		_, span := trace.StartSpan(ctx, "proto.Clone.Validators")
		defer span.End()
		tempState := &pb.BeaconState{
			Validators: db.validatorRegistry,
		}
		newState := proto.Clone(tempState).(*pb.BeaconState)
		return newState.Validators, nil
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

	return beaconState.Validators, err
}

// ValidatorFromState fetches the validator with the desired index from the cached registry.
func (db *BeaconDB) ValidatorFromState(ctx context.Context, index uint64) (*ethpb.Validator, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.ValidatorFromState")
	defer span.End()

	db.stateLock.RLock()
	defer db.stateLock.RUnlock()

	if db.validatorRegistry != nil {
		// return error if it's an invalid validator index.
		if index >= uint64(len(db.validatorRegistry)) {
			return nil, fmt.Errorf("invalid validator index %d", index)
		}
		validator := proto.Clone(db.validatorRegistry[index]).(*ethpb.Validator)
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

	return beaconState.Validators[index], err
}

// Balances fetches the current validator balances stored in state.
func (db *BeaconDB) Balances(ctx context.Context) ([]uint64, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.Balances")
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

	return beaconState.Balances, err
}

func createState(enc []byte) (*pb.BeaconState, error) {
	protoState := &pb.BeaconState{}
	err := proto.Unmarshal(enc, protoState)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoding")
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
			slotBinary := k[:8]
			keySlotNumber := decodeToSlotNumber(slotBinary)
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

// SaveCheckpointState saves the check pointed state by using check point as key.
func (db *BeaconDB) SaveCheckpointState(ctx context.Context, beaconState *pb.BeaconState, checkpt *ethpb.Checkpoint) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveCheckpointState")
	defer span.End()

	key, err := hashutil.HashProto(checkpt)
	if err != nil {
		return err
	}

	enc, err := proto.Marshal(beaconState)
	if err != nil {
		return err
	}

	return db.update(func(tx *bolt.Tx) error {
		b := tx.Bucket(checkpointBucket)
		return b.Put(key[:], enc)
	})
}

// CheckpointState gets the check pointed state check point as key.
func (db *BeaconDB) CheckpointState(ctx context.Context, checkpt *ethpb.Checkpoint) (*pb.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.CheckpointState")
	defer span.End()

	key, err := hashutil.HashProto(checkpt)
	if err != nil {
		return nil, err
	}

	var s *pb.BeaconState
	err = db.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(checkpointBucket)

		enc := bucket.Get(key[:])
		if enc == nil {
			return nil
		}

		s, err = createState(enc)
		return err
	})
	return s, err
}

// HasCheckpoint checks if a state of the corresponded check point exists in db.
func (db *BeaconDB) HasCheckpoint(checkpt *ethpb.Checkpoint) (bool, error) {
	b, err := hashutil.HashProto(checkpt)
	if err != nil {
		return false, err
	}

	exists := false
	if err := db.view(func(tx *bolt.Tx) error {
		l := tx.Bucket(checkpointBucket)
		exists = l.Get(b[:]) != nil
		return nil
	}); err != nil {
		return false, nil
	}
	return exists, nil
}

// SaveForkChoiceState saves the beacon state using beacon block root as key. This is used for fork choice.
func (db *BeaconDB) SaveForkChoiceState(ctx context.Context, beaconState *pb.BeaconState, blockRoot []byte) error {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.db.SaveForkChoiceState")
	defer span.End()

	enc, err := proto.Marshal(beaconState)
	if err != nil {
		return err
	}

	return db.update(func(tx *bolt.Tx) error {
		b := tx.Bucket(forkChoiceStateBucket)
		return b.Put(blockRoot, enc)
	})
}

// ForkChoiceState gets the beacon state using beacon block root as key. This is used for fork choice.
func (db *BeaconDB) ForkChoiceState(ctx context.Context, blockRoot []byte) (*pb.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.db.ForkChoiceState")
	defer span.End()

	var s *pb.BeaconState
	var err error
	err = db.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(forkChoiceStateBucket)

		enc := bucket.Get(blockRoot)
		if enc == nil {
			return nil
		}

		s, err = createState(enc)
		return err
	})
	return s, err
}

// LastForkChoiceState gets the last saved beacon state using beacon block root as key.
func (db *BeaconDB) LastForkChoiceState(ctx context.Context) (*pb.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.db.ForkChoiceState")
	defer span.End()

	var s *pb.BeaconState
	var err error
	err = db.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(forkChoiceStateBucket)

		_, enc := bucket.Cursor().Last()
		if enc == nil {
			return nil
		}

		s, err = createState(enc)
		return err
	})
	return s, err
}
