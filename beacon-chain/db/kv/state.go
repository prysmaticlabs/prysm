package kv

import (
	"bytes"
	"context"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
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
		if err := k.setStateSlotBitField(ctx, tx, state.Slot()); err != nil {
			return err
		}

		bucket := tx.Bucket(stateBucket)
		return bucket.Put(blockRoot[:], enc)
	})
}

// SaveStates stores multiple states to the db using the provided corresponding roots.
func (k *Store) SaveStates(ctx context.Context, states []*state.BeaconState, blockRoots [][32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveStates")
	defer span.End()
	if states == nil {
		return errors.New("nil state")
	}
	var err error
	multipleEncs := make([][]byte, len(states))
	for i, st := range states {
		multipleEncs[i], err = encode(st.InnerStateUnsafe())
		if err != nil {
			return err
		}
	}

	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(stateBucket)
		for i, rt := range blockRoots {
			if err := k.setStateSlotBitField(ctx, tx, states[i].Slot()); err != nil {
				return err
			}
			err = bucket.Put(rt[:], multipleEncs[i])
			if err != nil {
				return err
			}
		}
		return nil
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

		if featureconfig.Get().NewStateMgmt {
			if tx.Bucket(stateSummaryBucket).Get(blockRoot[:]) == nil {
				return errors.New("cannot delete state without state summary")
			}
		} else {
			// Safe guard against deleting genesis, finalized, head state.
			if bytes.Equal(blockRoot[:], checkpoint.Root) || bytes.Equal(blockRoot[:], genesisBlockRoot) || bytes.Equal(blockRoot[:], headBlkRoot) {
				return errors.New("cannot delete genesis, finalized, or head state")
			}
		}

		enc = bkt.Get(blockRoot[:])
		if enc == nil {
			return nil
		}
		block := &ethpb.SignedBeaconBlock{}
		if err := decode(enc, block); err != nil {
			return err
		}
		if err := k.clearBlockSlotBitField(ctx, tx, block.Block.Slot); err != nil {
			return err
		}

		bkt = tx.Bucket(stateBucket)
		return bkt.Delete(blockRoot[:])
	})
}

// DeleteStates by block roots.
//
// Note: bkt.Delete(key) uses a binary search to find the item in the database. Iterating with a
// cursor is faster when there are a large set of keys to delete. This method is O(n) deletion where
// n is the number of keys in the database. The alternative of calling  bkt.Delete on each key to
// delete would be O(m*log(n)) which would be much slower given a large set of keys to delete.
func (k *Store) DeleteStates(ctx context.Context, blockRoots [][32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DeleteStates")
	defer span.End()

	rootMap := make(map[[32]byte]bool)
	for _, blockRoot := range blockRoots {
		rootMap[blockRoot] = true
	}

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

		blockBkt := tx.Bucket(blocksBucket)
		headBlkRoot := blockBkt.Get(headBlockRootKey)
		bkt = tx.Bucket(stateBucket)
		c := bkt.Cursor()

		for blockRoot, _ := c.First(); blockRoot != nil; blockRoot, _ = c.Next() {
			if rootMap[bytesutil.ToBytes32(blockRoot)] {
				if featureconfig.Get().NewStateMgmt {
					if tx.Bucket(stateSummaryBucket).Get(blockRoot[:]) == nil {
						return errors.New("cannot delete state without state summary")
					}
				} else {
					// Safe guard against deleting genesis, finalized, head state.
					if bytes.Equal(blockRoot[:], checkpoint.Root) || bytes.Equal(blockRoot[:], genesisBlockRoot) || bytes.Equal(blockRoot[:], headBlkRoot) {
						return errors.New("cannot delete genesis, finalized, or head state")
					}
				}

				enc = bkt.Get(blockRoot[:])
				if enc == nil {
					return nil
				}
				block := &ethpb.SignedBeaconBlock{}
				if err := decode(enc, block); err != nil {
					return err
				}
				if err := k.clearBlockSlotBitField(ctx, tx, block.Block.Slot); err != nil {
					return err
				}

				if err := c.Delete(); err != nil {
					return err
				}
			}
		}
		return nil
	})
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

// HighestSlotState returns the block with the highest slot from the db.
func (k *Store) HighestSlotState(ctx context.Context) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HighestSlotState")
	defer span.End()
	var s *pb.BeaconState
	err := k.db.View(func(tx *bolt.Tx) error {
		slotBkt := tx.Bucket(slotsHasObjectBucket)
		savedSlots := slotBkt.Get(savedStateSlotsKey)
		highestIndex, err := bytesutil.HighestBitIndex(savedSlots)
		if err != nil {
			return err
		}
		highestSlot := highestIndex - 1
		f := filters.NewFilter().SetStartSlot(uint64(highestSlot)).SetEndSlot(uint64(highestSlot))

		keys, err := getBlockRootsByFilter(ctx, tx, f)
		if err != nil {
			return err
		}
		if len(keys) != 1 {
			return errors.New("could not get one block root to get state")
		}

		stateBkt := tx.Bucket(stateBucket)
		enc := stateBkt.Get(keys[0][:])
		if enc == nil {
			return nil
		}

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

// setStateSlotBitField sets the state slot bit in DB.
// This helps to track which slot has a saved state in db.
func (k *Store) setStateSlotBitField(ctx context.Context, tx *bolt.Tx, slot uint64) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.setStateSlotBitField")
	defer span.End()

	bucket := tx.Bucket(slotsHasObjectBucket)
	slotBitfields := bucket.Get(savedStateSlotsKey)
	tmp := make([]byte, len(slotBitfields))
	copy(tmp, slotBitfields)
	slotBitfields = bytesutil.SetBit(tmp, int(slot))
	return bucket.Put(savedStateSlotsKey, slotBitfields)
}

// clearStateSlotBitField clears the state slot bit in DB.
// This helps to track which slot has a saved state in db.
func (k *Store) clearStateSlotBitField(ctx context.Context, tx *bolt.Tx, slot uint64) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.clearStateSlotBitField")
	defer span.End()

	bucket := tx.Bucket(slotsHasObjectBucket)
	slotBitfields := bucket.Get(savedStateSlotsKey)
	tmp := make([]byte, len(slotBitfields))
	copy(tmp, slotBitfields)
	slotBitfields = bytesutil.ClearBit(tmp, int(slot))
	return bucket.Put(savedStateSlotsKey, slotBitfields)
}
