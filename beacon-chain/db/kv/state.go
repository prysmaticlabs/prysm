package kv

import (
	"context"
	"fmt"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"go.opencensus.io/trace"
)

// State returns the saved state using block's signing root,
// this particular block was used to generate the state.
func (k *Store) State(ctx context.Context, blockRoot [32]byte) (*pb.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.State")
	defer span.End()
	var s *pb.BeaconState
	err := k.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(stateBucket)
		enc := bucket.Get(blockRoot[:])
		if enc == nil {
			return nil
		}

		var err error
		s, err = createState(enc)
		return err
	})
	return s, err
}

// GenerateStateAtSlot generates state from the last finalized epoch till the specified slot.
func (k *Store) GenerateStateAtSlot(ctx context.Context, slot uint64) (*pb.BeaconState, error) {
	fCheckpoint, err := k.FinalizedCheckpoint(ctx)
	if err != nil {
		return nil, err
	}
	jCheckpoint, err := k.JustifiedCheckpoint(ctx)
	if err != nil {
		return nil, err
	}

	var root [32]byte
	if slot < helpers.StartSlot(jCheckpoint.Epoch) {
		copy(root[0:32], fCheckpoint.Root)
	} else {
		copy(root[0:32], jCheckpoint.Root)
	}

	savedState, err := k.State(ctx, root)
	if err != nil {
		return nil, err
	}
	if savedState.Slot > slot {
		return nil, fmt.Errorf(
			"savedState slot %d < current finalized beacon state slot %d",
			savedState.Slot,
			slot,
		)
	}
	if savedState.Slot == slot {
		return savedState, nil
	}

	filter := filters.NewFilter()
	filter.SetStartSlot(savedState.Slot + 1)
	filter.SetEndSlot(slot)
	pBlocks, err := k.Blocks(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve block: %v", err)
	}

	// run N state transitions to generate state
	for i := 0; i < len(pBlocks); i++ {
		savedState, err = state.ExecuteStateTransitionNoVerify(
			ctx,
			savedState,
			pBlocks[i],
		)
		if err != nil {
			return nil, fmt.Errorf("could not execute state transition %v", err)
		}
	}

	return savedState, nil
}

// HeadState returns the latest canonical state in beacon chain.
func (k *Store) HeadState(ctx context.Context) (*pb.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HeadState")
	defer span.End()
	var s *pb.BeaconState
	err := k.db.View(func(tx *bolt.Tx) error {
		// Retrieve head block's signing root from blocks bucket,
		// to look up what the head state is.
		bucket := tx.Bucket(blocksBucket)
		headBlkRoot := bucket.Get(headBlockRootKey)

		bucket = tx.Bucket(stateBucket)
		enc := bucket.Get(headBlkRoot)
		if enc == nil {
			return nil
		}

		var err error
		s, err = createState(enc)
		return err
	})
	span.AddAttributes(trace.BoolAttribute("exists", s != nil))
	if s != nil {
		span.AddAttributes(trace.Int64Attribute("slot", int64(s.Slot)))
	}
	return s, err
}

// GenesisState returns the genesis state in beacon chain.
func (k *Store) GenesisState(ctx context.Context) (*pb.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.GenesisState")
	defer span.End()
	var s *pb.BeaconState
	err := k.db.View(func(tx *bolt.Tx) error {
		// Retrieve genesis block's signing root from blocks bucket,
		// to look up what the genesis state is.
		bucket := tx.Bucket(blocksBucket)
		genesisBlockRoot := bucket.Get(genesisBlockRootKey)

		bucket = tx.Bucket(stateBucket)
		enc := bucket.Get(genesisBlockRoot)
		if enc == nil {
			return nil
		}

		var err error
		s, err = createState(enc)
		return err
	})
	return s, err
}

// SaveState stores a state to the db using block's signing root which was used to generate the state.
func (k *Store) SaveState(ctx context.Context, state *pb.BeaconState, blockRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveState")
	defer span.End()
	enc, err := proto.Marshal(state)
	if err != nil {
		return err
	}

	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(stateBucket)
		return bucket.Put(blockRoot[:], enc)
	})
}

// creates state from marshaled proto state bytes.
func createState(enc []byte) (*pb.BeaconState, error) {
	protoState := &pb.BeaconState{}
	err := proto.Unmarshal(enc, protoState)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoding")
	}
	return protoState, nil
}
