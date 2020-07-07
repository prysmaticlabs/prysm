package kv

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// State returns the saved state using block's signing root,
// this particular block was used to generate the state.
func (kv *Store) State(ctx context.Context, blockRoot [32]byte) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.State")
	defer span.End()
	var s *pb.BeaconState
	enc, err := kv.stateBytes(ctx, blockRoot)
	if err != nil {
		return nil, err
	}

	if len(enc) == 0 {
		return nil, nil
	}

	s, err = createState(ctx, enc)
	if err != nil {
		return nil, err
	}
	return state.InitializeFromProtoUnsafe(s)
}

// HeadState returns the latest canonical state in beacon chain.
func (kv *Store) HeadState(ctx context.Context) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HeadState")
	defer span.End()
	var s *pb.BeaconState
	err := kv.db.View(func(tx *bolt.Tx) error {
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
		s, err = createState(ctx, enc)
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
func (kv *Store) GenesisState(ctx context.Context) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.GenesisState")
	defer span.End()
	var s *pb.BeaconState
	err := kv.db.View(func(tx *bolt.Tx) error {
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
		s, err = createState(ctx, enc)
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
func (kv *Store) SaveState(ctx context.Context, st *state.BeaconState, blockRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveState")
	defer span.End()

	return kv.SaveStates(ctx, []*state.BeaconState{st}, [][32]byte{blockRoot})
}

// SaveStates stores multiple states to the db using the provided corresponding roots.
func (kv *Store) SaveStates(ctx context.Context, states []*state.BeaconState, blockRoots [][32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveStates")
	defer span.End()
	if states == nil {
		return errors.New("nil state")
	}
	var err error
	multipleEncs := make([][]byte, len(states))
	for i, st := range states {
		multipleEncs[i], err = encode(ctx, st.InnerStateUnsafe())
		if err != nil {
			return err
		}
	}

	return kv.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(stateBucket)
		for i, rt := range blockRoots {
			if err := kv.setStateSlotBitField(ctx, tx, states[i].Slot()); err != nil {
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
func (kv *Store) HasState(ctx context.Context, blockRoot [32]byte) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HasState")
	defer span.End()
	enc, err := kv.stateBytes(ctx, blockRoot)
	if err != nil {
		panic(err)
	}
	return len(enc) > 0
}

// DeleteState by block root.
func (kv *Store) DeleteState(ctx context.Context, blockRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DeleteState")
	defer span.End()
	return kv.DeleteStates(ctx, [][32]byte{blockRoot})
}

// DeleteStates by block roots.
//
// Note: bkt.Delete(key) uses a binary search to find the item in the database. Iterating with a
// cursor is faster when there are a large set of keys to delete. This method is O(n) deletion where
// n is the number of keys in the database. The alternative of calling  bkt.Delete on each key to
// delete would be O(m*log(n)) which would be much slower given a large set of keys to delete.
func (kv *Store) DeleteStates(ctx context.Context, blockRoots [][32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DeleteStates")
	defer span.End()

	rootMap := make(map[[32]byte]bool)
	for _, blockRoot := range blockRoots {
		rootMap[blockRoot] = true
	}

	return kv.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blocksBucket)
		genesisBlockRoot := bkt.Get(genesisBlockRootKey)

		bkt = tx.Bucket(checkpointBucket)
		enc := bkt.Get(finalizedCheckpointKey)
		checkpoint := &ethpb.Checkpoint{}
		if enc == nil {
			checkpoint = &ethpb.Checkpoint{Root: genesisBlockRoot}
		} else if err := decode(ctx, enc, checkpoint); err != nil {
			return err
		}

		blockBkt := tx.Bucket(blocksBucket)
		headBlkRoot := blockBkt.Get(headBlockRootKey)
		bkt = tx.Bucket(stateBucket)
		c := bkt.Cursor()

		for blockRoot, _ := c.First(); blockRoot != nil; blockRoot, _ = c.Next() {
			if !rootMap[bytesutil.ToBytes32(blockRoot)] {
				continue
			}
			// Safe guard against deleting genesis, finalized, head state.
			if bytes.Equal(blockRoot[:], checkpoint.Root) || bytes.Equal(blockRoot[:], genesisBlockRoot) || bytes.Equal(blockRoot[:], headBlkRoot) {
				return errors.New("cannot delete genesis, finalized, or head state")
			}

			slot, err := slotByBlockRoot(ctx, tx, blockRoot)
			if err != nil {
				return err
			}
			if err := kv.clearStateSlotBitField(ctx, tx, slot); err != nil {
				return err
			}

			if err := c.Delete(); err != nil {
				return err
			}
		}
		return nil
	})
}

// creates state from marshaled proto state bytes.
func createState(ctx context.Context, enc []byte) (*pb.BeaconState, error) {
	protoState := &pb.BeaconState{}
	if err := decode(ctx, enc, protoState); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoding")
	}
	return protoState, nil
}

// HasState checks if a state by root exists in the db.
func (kv *Store) stateBytes(ctx context.Context, blockRoot [32]byte) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.stateBytes")
	defer span.End()
	var dst []byte
	err := kv.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(stateBucket)
		dst = bkt.Get(blockRoot[:])
		return nil
	})
	return dst, err
}

// slotByBlockRoot retrieves the corresponding slot of the input block root.
func slotByBlockRoot(ctx context.Context, tx *bolt.Tx, blockRoot []byte) (uint64, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.slotByBlockRoot")
	defer span.End()

	bkt := tx.Bucket(stateSummaryBucket)
	enc := bkt.Get(blockRoot)

	if enc == nil {
		// Fall back to check the block.
		bkt := tx.Bucket(blocksBucket)
		enc := bkt.Get(blockRoot)

		if enc == nil {
			// Fallback and check the state.
			bkt = tx.Bucket(stateBucket)
			enc = bkt.Get(blockRoot)
			if enc == nil {
				return 0, errors.New("state enc can't be nil")
			}
			s, err := createState(ctx, enc)
			if err != nil {
				return 0, err
			}
			if s == nil {
				return 0, errors.New("state can't be nil")
			}
			return s.Slot, nil
		}
		b := &ethpb.SignedBeaconBlock{}
		err := decode(ctx, enc, b)
		if err != nil {
			return 0, err
		}
		if b.Block == nil {
			return 0, errors.New("block can't be nil")
		}
		return b.Block.Slot, nil
	}
	stateSummary := &pb.StateSummary{}
	if err := decode(ctx, enc, stateSummary); err != nil {
		return 0, err
	}
	return stateSummary.Slot, nil
}

// HighestSlotStates returns the states with the highest slot from the db.
// Ideally there should just be one state per slot, but given validator
// can double propose, a single slot could have multiple block roots and
// reuslts states. This returns a list of states.
func (kv *Store) HighestSlotStates(ctx context.Context) ([]*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HighestSlotState")
	defer span.End()
	var states []*state.BeaconState
	err := kv.db.View(func(tx *bolt.Tx) error {
		slotBkt := tx.Bucket(slotsHasObjectBucket)
		savedSlots := slotBkt.Get(savedStateSlotsKey)
		highestIndex, err := bytesutil.HighestBitIndex(savedSlots)
		if err != nil {
			return err
		}
		states, err = kv.statesAtSlotBitfieldIndex(ctx, tx, highestIndex)

		return err
	})
	if err != nil {
		return nil, err
	}

	if len(states) == 0 {
		return nil, errors.New("could not get one state")
	}

	return states, nil
}

// HighestSlotStatesBelow returns the states with the highest slot below the input slot
// from the db. Ideally there should just be one state per slot, but given validator
// can double propose, a single slot could have multiple block roots and
// reuslts states. This returns a list of states.
func (kv *Store) HighestSlotStatesBelow(ctx context.Context, slot uint64) ([]*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HighestSlotStatesBelow")
	defer span.End()
	var states []*state.BeaconState
	err := kv.db.View(func(tx *bolt.Tx) error {
		slotBkt := tx.Bucket(slotsHasObjectBucket)
		savedSlots := slotBkt.Get(savedStateSlotsKey)
		if len(savedSlots) == 0 {
			savedSlots = bytesutil.MakeEmptyBitlists(int(slot))
		}
		highestIndex, err := bytesutil.HighestBitIndexAt(savedSlots, int(slot))
		if err != nil {
			return err
		}

		states, err = kv.statesAtSlotBitfieldIndex(ctx, tx, highestIndex)

		return err
	})
	if err != nil {
		return nil, err
	}

	if len(states) == 0 {
		return nil, errors.New("could not get one state")
	}

	return states, nil
}

// statesAtSlotBitfieldIndex retrieves the states in DB given the input index. The index represents
// the position of the slot bitfield the saved state maps to.
func (kv *Store) statesAtSlotBitfieldIndex(ctx context.Context, tx *bolt.Tx, index int) ([]*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.statesAtSlotBitfieldIndex")
	defer span.End()

	highestSlot := uint64(0)
	if uint64(index) > highestSlot+1 {
		highestSlot = uint64(index - 1)
	}

	if highestSlot == 0 {
		gState, err := kv.GenesisState(ctx)
		if err != nil {
			return nil, err
		}
		return []*state.BeaconState{gState}, nil
	}

	f := filters.NewFilter().SetStartSlot(highestSlot).SetEndSlot(highestSlot)

	keys, err := getBlockRootsByFilter(ctx, tx, f)
	if err != nil {
		return nil, err
	}

	if len(keys) == 0 {
		return nil, errors.New("could not get one block root to get state")
	}

	stateBkt := tx.Bucket(stateBucket)
	states := make([]*state.BeaconState, 0, len(keys))
	for i := range keys {
		enc := stateBkt.Get(keys[i][:])
		if enc == nil {
			continue
		}
		pbState, err := createState(ctx, enc)
		if err != nil {
			return nil, err
		}
		s, err := state.InitializeFromProtoUnsafe(pbState)
		if err != nil {
			return nil, err
		}
		states = append(states, s)
	}

	return states, err
}

// setStateSlotBitField sets the state slot bit in DB.
// This helps to track which slot has a saved state in db.
func (kv *Store) setStateSlotBitField(ctx context.Context, tx *bolt.Tx, slot uint64) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.setStateSlotBitField")
	defer span.End()

	kv.stateSlotBitLock.Lock()
	defer kv.stateSlotBitLock.Unlock()

	bucket := tx.Bucket(slotsHasObjectBucket)
	slotBitfields := bucket.Get(savedStateSlotsKey)

	// Copy is needed to avoid unsafe pointer conversions.
	// See: https://github.com/etcd-io/bbolt/pull/201
	tmp := make([]byte, len(slotBitfields))
	copy(tmp, slotBitfields)
	slotBitfields = bytesutil.SetBit(tmp, int(slot))
	return bucket.Put(savedStateSlotsKey, slotBitfields)
}

// clearStateSlotBitField clears the state slot bit in DB.
// This helps to track which slot has a saved state in db.
func (kv *Store) clearStateSlotBitField(ctx context.Context, tx *bolt.Tx, slot uint64) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.clearStateSlotBitField")
	defer span.End()

	kv.stateSlotBitLock.Lock()
	defer kv.stateSlotBitLock.Unlock()

	bucket := tx.Bucket(slotsHasObjectBucket)
	slotBitfields := bucket.Get(savedStateSlotsKey)

	// Copy is needed to avoid unsafe pointer conversions.
	// See: https://github.com/etcd-io/bbolt/pull/201
	tmp := make([]byte, len(slotBitfields))
	copy(tmp, slotBitfields)
	slotBitfields = bytesutil.ClearBit(tmp, int(slot))
	return bucket.Put(savedStateSlotsKey, slotBitfields)
}
