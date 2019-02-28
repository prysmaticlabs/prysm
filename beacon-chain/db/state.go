package db

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"go.opencensus.io/trace"
)

// InitializeState creates an initial genesis state for the beacon
// node using a set of genesis validators.
func (db *BeaconDB) InitializeState(genesisTime uint64, deposits []*pb.Deposit) error {
	beaconState, err := state.GenesisBeaconState(deposits, genesisTime, nil)
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

		return chainInfo.Put(stateLookupKey, stateEnc)
	})
}

// State fetches the canonical beacon chain's state from the DB.
func (db *BeaconDB) State(ctx context.Context) (*pb.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.State")
	defer span.End()
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
	return db.update(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)
		beaconStateEnc, err := proto.Marshal(beaconState)
		if err != nil {
			return err
		}
		return chainInfo.Put(stateLookupKey, beaconStateEnc)
	})
}

// UnfinalizedBlockState fetches an unfinalized block's
// active and crystallized state pair.
func (db *BeaconDB) UnfinalizedBlockState(stateRoot [32]byte) (*pb.BeaconState, error) {
	var beaconState *pb.BeaconState
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
func (db *BeaconDB) SaveUnfinalizedBlockState(beaconState *pb.BeaconState) error {
	enc, err := proto.Marshal(beaconState)
	if err != nil {
		return fmt.Errorf("unable to marshal the beacon state: %v", err)
	}
	stateHash := hashutil.Hash(enc)
	return db.update(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)
		if err := chainInfo.Put(stateHash[:], enc); err != nil {
			return fmt.Errorf("failed to save beacon state: %v", err)
		}
		return nil
	})
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
