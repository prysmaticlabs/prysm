package kv

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
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

// DeleteState by block root.
func (k *Store) DeleteState(ctx context.Context, blockRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DeleteState")
	defer span.End()

	return k.db.Batch(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)
		genesisBlockRoot := bkt.Get(genesisBlockRootKey)

		bkt = tx.Bucket(checkpointBucket)
		enc := bkt.Get(finalizedCheckpointKey)
		checkpoint := &ethpb.Checkpoint{}
		if enc == nil {
			checkpoint = &ethpb.Checkpoint{Root: genesisBlockRoot}
		} else {
			proto.Unmarshal(enc, checkpoint)
		}

		// Safe guard against deleting genesis or finalized state.
		if bytes.Equal(blockRoot[:], checkpoint.Root) || bytes.Equal(blockRoot[:], genesisBlockRoot) {
			return errors.New("could not delete genesis or finalized state")
		}

		bkt = tx.Bucket(stateBucket)
		return bkt.Delete(blockRoot[:])
	})
}

// DeleteStates by block roots.
func (k *Store) DeleteStates(ctx context.Context, blockRoots [][32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DeleteStates")
	defer span.End()
	var wg sync.WaitGroup
	errs := make([]string, 0)
	wg.Add(len(blockRoots))
	for _, r := range blockRoots {
		go func(w *sync.WaitGroup, root [32]byte) {
			defer w.Done()
			if err := k.DeleteState(ctx, root); err != nil {
				errs = append(errs, err.Error())
				return
			}
		}(&wg, r)
	}
	wg.Wait()
	if len(errs) > 0 {
		return fmt.Errorf("deleting states failed with %d errors: %s", len(errs), strings.Join(errs, ", "))
	}
	return nil
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
