package kv

import (
	"bytes"
	"context"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"go.opencensus.io/trace"
)

// State returns the saved state using block's signing root,
// this particular block was used to generate the state.
func (k *Store) State(ctx context.Context, blockRoot [32]byte) (*state.BeaconState, error) {
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
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, nil
	}
	return state.InitializeFromProtoUnsafe(s)
}

// HeadState returns the latest canonical state in beacon chain.
func (k *Store) HeadState(ctx context.Context) (*state.BeaconState, error) {
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
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, nil
	}
	span.AddAttributes(trace.BoolAttribute("exists", s != nil))
	if s != nil {
		span.AddAttributes(trace.Int64Attribute("slot", int64(s.Slot)))
	}
	return state.InitializeFromProtoUnsafe(s)
}

// GenesisState returns the genesis state in beacon chain.
func (k *Store) GenesisState(ctx context.Context) (*state.BeaconState, error) {
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
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, nil
	}
	return state.InitializeFromProtoUnsafe(s)
}

// SaveState stores a state to the db using block's signing root which was used to generate the state.
func (k *Store) SaveState(ctx context.Context, state *state.BeaconState, blockRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveState")
	defer span.End()
	if state == nil {
		return errors.New("nil state")
	}
	enc, err := encode(state.InnerStateUnsafe())
	if err != nil {
		return err
	}

	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(stateBucket)
		return bucket.Put(blockRoot[:], enc)
	})
}

// HasState checks if a state by root exists in the db.
func (k *Store) HasState(ctx context.Context, blockRoot [32]byte) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HasState")
	defer span.End()
	var exists bool
	// #nosec G104. Always returns nil.
	k.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(stateBucket)
		exists = bucket.Get(blockRoot[:]) != nil
		return nil
	})
	return exists
}

// DeleteState by block root.
func (k *Store) DeleteState(ctx context.Context, blockRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DeleteState")
	defer span.End()

	return k.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)
		genesisBlockRoot := bkt.Get(genesisBlockRootKey)

		bkt = tx.Bucket(checkpointBucket)
		enc := bkt.Get(finalizedCheckpointKey)
		checkpoint := &ethpb.Checkpoint{}
		if enc == nil {
			checkpoint = &ethpb.Checkpoint{Root: genesisBlockRoot}
		} else if err := decode(enc, checkpoint); err != nil {
			return err
		}

		bkt = tx.Bucket(blocksBucket)
		headBlkRoot := bkt.Get(headBlockRootKey)

		// Safe guard against deleting genesis, finalized, head state.
		if bytes.Equal(blockRoot[:], checkpoint.Root) || bytes.Equal(blockRoot[:], genesisBlockRoot) || bytes.Equal(blockRoot[:], headBlkRoot) {
			return errors.New("cannot delete genesis, finalized, or head state")
		}

		bkt = tx.Bucket(stateBucket)
		return bkt.Delete(blockRoot[:])
	})
}

// DeleteStates by block roots.
func (k *Store) DeleteStates(ctx context.Context, blockRoots [][32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DeleteStates")
	defer span.End()

	return k.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)
		genesisBlockRoot := bkt.Get(genesisBlockRootKey)

		bkt = tx.Bucket(checkpointBucket)
		enc := bkt.Get(finalizedCheckpointKey)
		checkpoint := &ethpb.Checkpoint{}
		if enc == nil {
			checkpoint = &ethpb.Checkpoint{Root: genesisBlockRoot}
		} else if err := decode(enc, checkpoint); err != nil {
			return err
		}

		bkt = tx.Bucket(blocksBucket)
		headBlkRoot := bkt.Get(headBlockRootKey)

		for _, blockRoot := range blockRoots {
			// Safe guard against deleting genesis, finalized, or head state.
			if bytes.Equal(blockRoot[:], checkpoint.Root) || bytes.Equal(blockRoot[:], genesisBlockRoot) || bytes.Equal(blockRoot[:], headBlkRoot) {
				return errors.New("could not delete genesis, finalized, or head state")
			}

			bkt = tx.Bucket(stateBucket)
			if err := bkt.Delete(blockRoot[:]); err != nil {
				return err
			}
		}
		return nil
	})
}

// SaveArchivePoint saves an archive point to the DB. This is used for cold state management.
// An archive point index is `slot / slots_per_archive_point`
func (k *Store) SaveArchivePoint(ctx context.Context, blockRoot [32]byte, index uint64) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveArchivePoint")
	defer span.End()

	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(coldStateSummaryBucket)
		return bucket.Put(uint64ToBytes(index), blockRoot[:])
	})
}

// ArchivePoint returns the block root of an archive point from the DB.
// This is used for cold state management and to restore a cold state.
func (k *Store) ArchivePoint(ctx context.Context, index uint64) []byte {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.ArchivePoint")
	defer span.End()

	var blockRoot []byte
	// #nosec G104. Always returns nil.
	k.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(coldStateSummaryBucket)
		blockRoot = bucket.Get(uint64ToBytes(index))
		return nil
	})

	return blockRoot
}

// creates state from marshaled proto state bytes.
func createState(enc []byte) (*pb.BeaconState, error) {
	protoState := &pb.BeaconState{}
	err := decode(enc, protoState)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoding")
	}
	return protoState, nil
}
