package kv

import (
	"bytes"
	"context"
	"math"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// StateByRoot retrieves the state using input block root.
func (s *Store) StateByRoot(ctx context.Context, blockRoot [32]byte) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.StateByRoot")
	defer span.End()

	// Genesis case. If block root is zero hash, short circuit to use genesis cachedState stored in DB.
	if blockRoot == params.BeaconConfig().ZeroHash {
		return s.State(ctx, blockRoot)
	}
	return s.loadStateByRoot(ctx, blockRoot)
}

// StateByRootInitialSync retrieves the state from the DB for the initial syncing phase.
// It assumes initial syncing using a block list rather than a block tree hence the returned
// state is not copied.
// It invalidates cache for parent root because pre state will get mutated.
// Do not use this method for anything other than initial syncing purpose or block tree is applied.
func (s *Store) StateByRootInitialSync(ctx context.Context, blockRoot [32]byte) (*state.BeaconState, error) {
	// Genesis case. If block root is zero hash, short circuit to use genesis state stored in DB.
	if blockRoot == params.BeaconConfig().ZeroHash {
		return s.State(ctx, blockRoot)
	}

	// To invalidate cache for parent root because pre state will get mutated.
	defer s.hotStateCache.delete(blockRoot)

	if s.hotStateCache.has(blockRoot) {
		return s.hotStateCache.getWithoutCopy(blockRoot), nil
	}

	cachedInfo, ok, err := s.epochBoundaryStateCache.getByRoot(blockRoot)
	if err != nil {
		return nil, err
	}
	if ok {
		return cachedInfo.state, nil
	}

	startState, err := s.lastAncestorState(ctx, blockRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get ancestor state")
	}
	if startState == nil {
		return nil, errors.New("nil state")
	}
	summary, err := s.stateSummary(ctx, blockRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get state summary")
	}
	if startState.Slot() == summary.Slot {
		return startState, nil
	}

	blks, err := s.LoadBlocks(ctx, startState.Slot()+1, summary.Slot, bytesutil.ToBytes32(summary.Root))
	if err != nil {
		return nil, errors.Wrap(err, "could not load blocks")
	}
	startState, err = s.ReplayBlocks(ctx, startState, blks, summary.Slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not replay blocks")
	}

	return startState, nil
}

// StateBySlot retrieves the state using input slot.
func (s *Store) StateBySlot(ctx context.Context, slot uint64) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.StateBySlot")
	defer span.End()

	return s.loadStateBySlot(ctx, slot)
}

// This loads a beacon state from either the cache or DB then replay blocks up the requested block root.
func (s *Store) loadStateByRoot(ctx context.Context, blockRoot [32]byte) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.loadStateByRoot")
	defer span.End()

	// First, it checks if the state exists in hot state cache.
	cachedState := s.hotStateCache.get(blockRoot)
	if cachedState != nil {
		return cachedState, nil
	}

	// Second, it checks if the state exits in epoch boundary state cache.
	cachedInfo, ok, err := s.epochBoundaryStateCache.getByRoot(blockRoot)
	if err != nil {
		return nil, err
	}
	if ok {
		return cachedInfo.state, nil
	}

	// Short cut if the cachedState is already in the DB.
	has, err := s.HasState(ctx, blockRoot)
	if err != nil {
		return nil, err
	}
	if has {
		return s.State(ctx, blockRoot)
	}

	summary, err := s.stateSummary(ctx, blockRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get state summary")
	}
	targetSlot := summary.Slot

	// Since the requested state is not in caches, start replaying using the last available ancestor state which is
	// retrieved using input block's parent root.
	startState, err := s.lastAncestorState(ctx, blockRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get ancestor state")
	}
	if startState == nil {
		return nil, errUnknownBoundaryState
	}

	blks, err := s.LoadBlocks(ctx, startState.Slot()+1, targetSlot, bytesutil.ToBytes32(summary.Root))
	if err != nil {
		return nil, errors.Wrap(err, "could not load blocks for hot state using root")
	}

	replayBlockCount.Observe(float64(len(blks)))

	return s.ReplayBlocks(ctx, startState, blks, targetSlot)
}

// This loads a state by slot.
func (s *Store) loadStateBySlot(ctx context.Context, slot uint64) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.loadStateBySlot")
	defer span.End()

	// Return genesis state if slot is 0.
	if slot == 0 {
		return s.GenesisState(ctx)
	}

	// Gather last saved state, that is where node starts to replay the blocks.
	startState, err := s.lastSavedState(ctx, slot)
	if err != nil {
		return nil, err
	}

	// Gather the last saved block root and the slot number.
	lastValidRoot, lastValidSlot, err := s.lastSavedBlock(ctx, slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get last valid block for hot state using slot")
	}

	// Load and replay blocks to get the intermediate state.
	replayBlks, err := s.LoadBlocks(ctx, startState.Slot()+1, lastValidSlot, lastValidRoot)
	if err != nil {
		return nil, err
	}

	// If there's no blocks to replay, a node doesn't need to recalculate the start state.
	// A node can simply advance the slots on the last saved state.
	if len(replayBlks) == 0 {
		return s.ReplayBlocks(ctx, startState, replayBlks, slot)
	}

	pRoot := bytesutil.ToBytes32(replayBlks[0].Block.ParentRoot)
	replayStartState, err := s.loadStateByRoot(ctx, pRoot)
	if err != nil {
		return nil, err
	}
	return s.ReplayBlocks(ctx, replayStartState, replayBlks, slot)
}

// State returns the saved state using block's signing root,
// this particular block was used to generate the state.
func (s *Store) State(ctx context.Context, blockRoot [32]byte) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.State")
	defer span.End()
	var st *pb.BeaconState
	enc, err := s.stateBytes(ctx, blockRoot)
	if err != nil {
		return nil, err
	}

	if len(enc) == 0 {
		return nil, nil
	}

	st, err = createState(ctx, enc)
	if err != nil {
		return nil, err
	}
	return state.InitializeFromProtoUnsafe(st)
}

// GenesisState returns the genesis state in beacon chain.
func (s *Store) GenesisState(ctx context.Context) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.GenesisState")
	defer span.End()
	var st *pb.BeaconState
	err := s.db.View(func(tx *bolt.Tx) error {
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
		st, err = createState(ctx, enc)
		return err
	})
	if err != nil {
		return nil, err
	}
	if st == nil {
		return nil, nil
	}
	return state.InitializeFromProtoUnsafe(st)
}

// SaveState stores a state to the db using block's signing root which was used to generate the state.
func (s *Store) SaveState(ctx context.Context, st *state.BeaconState, blockRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveState")
	defer span.End()

	return s.SaveStates(ctx, []*state.BeaconState{st}, [][32]byte{blockRoot})
}

// SaveStates stores multiple states to the db using the provided corresponding roots.
func (s *Store) SaveStates(ctx context.Context, states []*state.BeaconState, blockRoots [][32]byte) error {
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

	return s.db.Update(func(tx *bolt.Tx) error {
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
func (s *Store) HasState(ctx context.Context, blockRoot [32]byte) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HasState")
	defer span.End()

	has, err := s.HasStateInCache(ctx, blockRoot)
	if err != nil {
		return false, err
	}
	if has {
		return true, nil
	}

	enc, err := s.stateBytes(ctx, blockRoot)
	if err != nil {
		panic(err)
	}
	return len(enc) > 0, nil
}

// HasStateInCache returns true if the state exists in cache.
func (s *Store) HasStateInCache(ctx context.Context, blockRoot [32]byte) (bool, error) {
	if s.hotStateCache.has(blockRoot) {
		return true, nil
	}
	_, has, err := s.epochBoundaryStateCache.getByRoot(blockRoot)
	if err != nil {
		return false, err
	}
	return has, nil
}

// DeleteState by block root.
func (s *Store) DeleteState(ctx context.Context, blockRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DeleteState")
	defer span.End()

	return s.db.Update(func(tx *bolt.Tx) error {
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
		// Safe guard against deleting genesis, finalized, head state.
		if bytes.Equal(blockRoot[:], checkpoint.Root) || bytes.Equal(blockRoot[:], genesisBlockRoot) || bytes.Equal(blockRoot[:], headBlkRoot) {
			return errors.New("cannot delete genesis, finalized, or head state")
		}

		slot, err := slotByBlockRoot(ctx, tx, blockRoot[:])
		if err != nil {
			return err
		}
		indicesByBucket := createStateIndicesFromStateSlot(ctx, slot)
		if err := deleteValueForIndices(ctx, indicesByBucket, blockRoot[:], tx); err != nil {
			return errors.Wrap(err, "could not delete root for DB indices")
		}

		return bkt.Delete(blockRoot[:])
	})
}

// DeleteStates by block roots.
func (s *Store) DeleteStates(ctx context.Context, blockRoots [][32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DeleteStates")
	defer span.End()

	for _, r := range blockRoots {
		if err := s.DeleteState(ctx, r); err != nil {
			return err
		}
	}

	return nil
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
func (s *Store) stateBytes(ctx context.Context, blockRoot [32]byte) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.stateBytes")
	defer span.End()
	var dst []byte
	err := s.db.View(func(tx *bolt.Tx) error {
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
func (s *Store) HighestSlotStatesBelow(ctx context.Context, slot uint64) ([]*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HighestSlotStatesBelow")
	defer span.End()

	var best []byte
	if err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(stateSlotIndicesBucket)
		c := bkt.Cursor()
		for s, root := c.First(); s != nil; s, root = c.Next() {
			if ctx.Err() != nil {
				return ctx.Err()
			}
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
		st, err = s.State(ctx, bytesutil.ToBytes32(best))
		if err != nil {
			return nil, err
		}
	}
	if st == nil {
		st, err = s.GenesisState(ctx)
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

// CleanUpDirtyStates removes states in DB that falls to under archived point interval rules.
// Only following states would be kept:
// 1.) state_slot % archived_interval == 0. (e.g. archived_interval=2048, states with slot 2048, 4096... etc)
// 2.) archived_interval - archived_interval/3 < state_slot % archived_interval
//   (e.g. archived_interval=2048, states with slots after 1365).
//   This is to tolerate skip slots. Not every state lays on the boundary.
// 3.) state with current finalized root
// 4.) unfinalized States
func (s *Store) CleanUpDirtyStates(ctx context.Context, slotsPerArchivedPoint uint64) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB. CleanUpDirtyStates")
	defer span.End()

	f, err := s.FinalizedCheckpoint(ctx)
	if err != nil {
		return err
	}
	finalizedSlot, err := helpers.StartSlot(f.Epoch)
	if err != nil {
		return err
	}
	deletedRoots := make([][32]byte, 0)

	err = s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(stateSlotIndicesBucket)
		return bkt.ForEach(func(k, v []byte) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			finalizedChkpt := bytesutil.ToBytes32(f.Root) == bytesutil.ToBytes32(v)
			slot := bytesutil.BytesToUint64BigEndian(k)
			mod := slot % slotsPerArchivedPoint
			nonFinalized := slot > finalizedSlot

			// The following conditions cover 1, 2, 3 and 4 above.
			if mod != 0 && mod <= slotsPerArchivedPoint-slotsPerArchivedPoint/3 && !finalizedChkpt && !nonFinalized {
				deletedRoots = append(deletedRoots, bytesutil.ToBytes32(v))
			}
			return nil
		})
	})
	if err != nil {
		return err
	}

	// Length of to be deleted roots is 0. Nothing to do.
	if len(deletedRoots) == 0 {
		return nil
	}

	log.WithField("count", len(deletedRoots)).Info("Cleaning up dirty states")
	if err := s.DeleteStates(ctx, deletedRoots); err != nil {
		return err
	}

	return err
}

// This returns the highest available ancestor state of the input block root.
// It recursively look up block's parent until a corresponding state of the block root
// is found in the caches or DB.
//
// There's three ways to derive block parent state:
// 1.) block parent state is the last finalized state
// 2.) block parent state is the epoch boundary state and exists in epoch boundary cache.
// 3.) block parent state is in DB.
func (s *Store) lastAncestorState(ctx context.Context, root [32]byte) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.lastAncestorState")
	defer span.End()

	if s.isFinalizedRoot(root) && s.finalizedState() != nil {
		return s.finalizedState(), nil
	}

	b, err := s.Block(ctx, root)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, errors.New("nil block")
	}

	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		// Is the state a genesis state.
		parentRoot := bytesutil.ToBytes32(b.Block.ParentRoot)
		if parentRoot == params.BeaconConfig().ZeroHash {
			return s.GenesisState(ctx)
		}

		// Does the state exist in the hot state cache.
		if s.hotStateCache.has(parentRoot) {
			return s.hotStateCache.get(parentRoot), nil
		}

		// Does the state exist in finalized info cache.
		if s.isFinalizedRoot(parentRoot) {
			return s.finalizedState(), nil
		}

		// Does the state exist in epoch boundary cache.
		cachedInfo, ok, err := s.epochBoundaryStateCache.getByRoot(parentRoot)
		if err != nil {
			return nil, err
		}
		if ok {
			return cachedInfo.state, nil
		}

		// Does the state exists in DB.
		has, err := s.HasState(ctx, parentRoot)
		if err != nil {
			return nil, err
		}
		if has {
			return s.StateByRoot(ctx, parentRoot)
		}
		b, err = s.Block(ctx, parentRoot)
		if err != nil {
			return nil, err
		}
		if b == nil {
			return nil, errUnknownBlock
		}
	}
}

// This returns the state summary object of a given block root, it first checks the cache
// then checks the DB. An error is returned if state summary object is nil.
func (s *Store) stateSummary(ctx context.Context, blockRoot [32]byte) (*pb.StateSummary, error) {
	var summary *pb.StateSummary
	var err error
	if s.stateSummaryCache == nil {
		return nil, errors.New("nil stateSummaryCache")
	}
	if s.stateSummaryCache.Has(blockRoot) {
		summary = s.stateSummaryCache.Get(blockRoot)
	} else {
		summary, err = s.StateSummary(ctx, blockRoot)
		if err != nil {
			return nil, err
		}
	}
	if summary == nil {
		return s.RecoverStateSummary(ctx, blockRoot)
	}
	return summary, nil
}

// ForceCheckpoint initiates a cold state save of the given state. This method does not update the
// "last archived state" but simply saves the specified state from the root argument into the DB.
func (s *Store) ForceCheckpoint(ctx context.Context, root []byte) error {
	ctx, span := trace.StartSpan(ctx, "stateGen.ForceCheckpoint")
	defer span.End()

	root32 := bytesutil.ToBytes32(root)
	// Before the first finalized check point, the finalized root is zero hash.
	// Return early if there hasn't been a finalized check point.
	if root32 == params.BeaconConfig().ZeroHash {
		return nil
	}

	fs, err := s.loadStateByRoot(ctx, root32)
	if err != nil {
		return err
	}

	return s.SaveState(ctx, fs, root32)
}

// This saves a post beacon state. On the epoch boundary,
// it saves a full state. On an intermediate slot, it saves a back pointer to the
// nearest epoch boundary state.
func (s *Store) SaveStateByRoot(ctx context.Context, blockRoot [32]byte, state *state.BeaconState) error {
	ctx, span := trace.StartSpan(ctx, "stateGen.saveStateByRoot")
	defer span.End()

	// Duration can't be 0 to prevent panic for division.
	duration := uint64(math.Max(float64(s.saveHotStateDB.duration), 1))

	s.saveHotStateDB.lock.Lock()
	if s.saveHotStateDB.enabled && state.Slot()%duration == 0 {
		if err := s.SaveState(ctx, state, blockRoot); err != nil {
			s.saveHotStateDB.lock.Unlock()
			return err
		}
		s.saveHotStateDB.savedStateRoots = append(s.saveHotStateDB.savedStateRoots, blockRoot)

		log.WithFields(logrus.Fields{
			"slot":                   state.Slot(),
			"totalHotStateSavedInDB": len(s.saveHotStateDB.savedStateRoots),
		}).Info("Saving hot state to DB")
	}
	s.saveHotStateDB.lock.Unlock()

	// If the hot state is already in cache, one can be sure the state was processed and in the DB.
	if s.hotStateCache.has(blockRoot) {
		return nil
	}

	// Only on an epoch boundary slot, saves epoch boundary state in epoch boundary root state cache.
	if helpers.IsEpochStart(state.Slot()) {
		if err := s.epochBoundaryStateCache.put(blockRoot, state); err != nil {
			return err
		}
	}

	// On an intermediate slots, save the hot state summary.
	s.stateSummaryCache.Put(blockRoot, &pb.StateSummary{
		Slot: state.Slot(),
		Root: blockRoot[:],
	})

	// Store the copied state in the hot state cache.
	s.hotStateCache.put(blockRoot, state)

	return nil
}

// EnableSaveHotStateToDB enters the mode that saves hot beacon state to the DB.
// This usually gets triggered when there's long duration since finality.
func (s *Store) EnableSaveHotStateToDB(_ context.Context) {
	s.saveHotStateDB.lock.Lock()
	defer s.saveHotStateDB.lock.Unlock()
	if s.saveHotStateDB.enabled {
		return
	}

	s.saveHotStateDB.enabled = true

	log.WithFields(logrus.Fields{
		"enabled":       s.saveHotStateDB.enabled,
		"slotsInterval": s.saveHotStateDB.duration,
	}).Warn("Entering mode to save hot states in DB")
}

// DisableSaveHotStateToDB exits the mode that saves beacon state to DB for the hot states.
// This usually gets triggered once there's finality after long duration since finality.
func (s *Store) DisableSaveHotStateToDB(ctx context.Context) error {
	s.saveHotStateDB.lock.Lock()
	defer s.saveHotStateDB.lock.Unlock()
	if !s.saveHotStateDB.enabled {
		return nil
	}

	log.WithFields(logrus.Fields{
		"enabled":          s.saveHotStateDB.enabled,
		"deletedHotStates": len(s.saveHotStateDB.savedStateRoots),
	}).Warn("Exiting mode to save hot states in DB")

	// Delete previous saved states in DB as we are turning this mode off.
	s.saveHotStateDB.enabled = false
	if err := s.DeleteStates(ctx, s.saveHotStateDB.savedStateRoots); err != nil {
		return err
	}
	s.saveHotStateDB.savedStateRoots = nil

	return nil
}
