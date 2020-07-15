package kv

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"

	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
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
			indicesByBucket := createStateIndicesFromStateSlot(ctx, states[i].Slot())
			if err := updateValueForIndices(ctx, indicesByBucket, rt[:], tx); err != nil {
				return errors.Wrap(err, "could not update DB indices")
			}
			if err := bucket.Put(rt[:], multipleEncs[i]); err != nil {
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

	rootMap := make(map[[32]byte]bool, len(blockRoots))
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
			indicesByBucket := createStateIndicesFromStateSlot(ctx, slot)
			if err := deleteValueForIndices(ctx, indicesByBucket, blockRoot[:], tx); err != nil {
				return errors.Wrap(err, "could not delete root for DB indices")
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

// HighestSlotStatesBelow returns the states with the highest slot below the input slot
// from the db. Ideally there should just be one state per slot, but given validator
// can double propose, a single slot could have multiple block roots and
// results states. This returns a list of states.
func (kv *Store) HighestSlotStatesBelow(ctx context.Context, slot uint64) ([]*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HighestSlotStatesBelow")
	defer span.End()

	var best []byte
	if err := kv.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(stateSlotIndicesBucket)
		c := bkt.Cursor()
		for s, root := c.First(); s != nil; s, root = c.Next() {
			key := bytesutil.BytesToUint64BigEndian(s)
			if root == nil {
				continue
			}
			if key >= slot {
				break
			}
			best = root
		}
		return nil
	}); err != nil {
		return nil, err
	}

	var st *state.BeaconState
	var err error
	if best != nil {
		st, err = kv.State(ctx, bytesutil.ToBytes32(best))
		if err != nil {
			return nil, err
		}
	}
	if st == nil {
		st, err = kv.GenesisState(ctx)
		if err != nil {
			return nil, err
		}
	}

	return []*state.BeaconState{st}, nil
}

// createBlockIndicesFromBlock takes in a beacon block and returns
// a map of bolt DB index buckets corresponding to each particular key for indices for
// data, such as (shard indices bucket -> shard 5).
func createStateIndicesFromStateSlot(ctx context.Context, slot uint64) map[string][]byte {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.createStateIndicesFromState")
	defer span.End()
	indicesByBucket := make(map[string][]byte)
	// Every index has a unique bucket for fast, binary-search
	// range scans for filtering across keys.
	buckets := [][]byte{
		stateSlotIndicesBucket,
	}

	indices := [][]byte{
		bytesutil.Uint64ToBytesBigEndian(slot),
	}
	for i := 0; i < len(buckets); i++ {
		indicesByBucket[string(buckets[i])] = indices[i]
	}
	return indicesByBucket
}
