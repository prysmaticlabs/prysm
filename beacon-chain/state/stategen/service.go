// Package stategen defines functions to regenerate beacon chain states
// by replaying blocks from a stored state checkpoint, useful for
// optimization and reducing a beacon node's resource consumption.
package stategen

import (
	"context"
	"errors"
	"sync"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

var defaultHotStateDBInterval uint64 = 128 // slots
var errInvalidRange = errors.New("start slot and end slot are not a valid range")

// State represents a management object that handles the internal
// logic of maintaining both hot and cold states in DB.
type State struct {
	beaconDB                db.NoHeadAccessDatabase
	slotsPerArchivedPoint   uint64
	hotStateCache           *cache.HotStateCache
	finalizedInfo           *finalizedInfo
	stateSummaryCache       *cache.StateSummaryCache
	epochBoundaryStateCache *epochBoundaryState
	saveHotStateDB          *saveHotStateDbConfig
	LoadBlocks              func(ctx context.Context, startSlot, endSlot uint64, endBlockRoot [32]byte) ([]*eth.SignedBeaconBlock, error)
}

// This tracks the config in the event of long non-finality,
// how often does the node save hot states to db? what are
// the saved hot states in db?... etc
type saveHotStateDbConfig struct {
	enabled         bool
	lock            sync.Mutex
	duration        uint64
	savedStateRoots [][32]byte
}

// This tracks the finalized point. It's also the point where slot and the block root of
// cold and hot sections of the DB splits.
type finalizedInfo struct {
	slot  uint64
	root  [32]byte
	state *state.BeaconState
	lock  sync.RWMutex
}

// New returns a new state management object.
func New(db db.NoHeadAccessDatabase, stateSummaryCache *cache.StateSummaryCache) *State {
	s := &State{
		beaconDB:                db,
		hotStateCache:           cache.NewHotStateCache(),
		finalizedInfo:           &finalizedInfo{slot: 0, root: params.BeaconConfig().ZeroHash},
		slotsPerArchivedPoint:   params.BeaconConfig().SlotsPerArchivedPoint,
		stateSummaryCache:       stateSummaryCache,
		epochBoundaryStateCache: newBoundaryStateCache(),
		saveHotStateDB: &saveHotStateDbConfig{
			duration: defaultHotStateDBInterval,
		},
	}
	// s.LoadBlocks = loadBlocks(s)
	s.LoadBlocks = func(ctx context.Context, startSlot, endSlot uint64, endBlockRoot [32]byte) ([]*eth.SignedBeaconBlock, error) {
		// Nothing to load for invalid range.
		if endSlot < startSlot {
			return nil, errInvalidRange
		}
		filter := filters.NewFilter().SetStartSlot(startSlot).SetEndSlot(endSlot)
		blocks, blockRoots, err := s.beaconDB.Blocks(ctx, filter)
		if err != nil {
			return nil, err
		}
		// The retrieved blocks and block roots have to be in the same length given same filter.
		if len(blocks) != len(blockRoots) {
			return nil, errors.New("length of blocks and roots don't match")
		}
		// Return early if there's no block given the input.
		length := len(blocks)
		if length == 0 {
			return nil, nil
		}

		// The last retrieved block root has to match input end block root.
		// Covers the edge case if there's multiple blocks on the same end slot,
		// the end root may not be the last index in `blockRoots`.
		for length >= 3 && blocks[length-1].Block.Slot == blocks[length-2].Block.Slot && blockRoots[length-1] != endBlockRoot {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			length--
			if blockRoots[length-2] == endBlockRoot {
				length--
				break
			}
		}

		if blockRoots[length-1] != endBlockRoot {
			return nil, errors.New("end block roots don't match")
		}

		filteredBlocks := []*eth.SignedBeaconBlock{blocks[length-1]}
		// Starting from second to last index because the last block is already in the filtered block list.
		for i := length - 2; i >= 0; i-- {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			b := filteredBlocks[len(filteredBlocks)-1]
			if bytesutil.ToBytes32(b.Block.ParentRoot) != blockRoots[i] {
				continue
			}
			filteredBlocks = append(filteredBlocks, blocks[i])
		}

		return filteredBlocks, nil
	}

	return s
}

// Resume resumes a new state management object from previously saved finalized check point in DB.
func (s *State) Resume(ctx context.Context) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.Resume")
	defer span.End()

	c, err := s.beaconDB.FinalizedCheckpoint(ctx)
	if err != nil {
		return nil, err
	}
	fRoot := bytesutil.ToBytes32(c.Root)
	// Resume as genesis state if last finalized root is zero hashes.
	if fRoot == params.BeaconConfig().ZeroHash {
		return s.beaconDB.GenesisState(ctx)
	}
	fState, err := s.StateByRoot(ctx, fRoot)
	if err != nil {
		return nil, err
	}
	if fState == nil {
		return nil, errors.New("finalized state not found in disk")
	}

	go func() {
		if err := s.beaconDB.CleanUpDirtyStates(ctx, s.slotsPerArchivedPoint); err != nil {
			log.WithError(err).Error("Could not clean up dirty states")
		}
	}()

	s.finalizedInfo = &finalizedInfo{slot: fState.Slot(), root: fRoot, state: fState.Copy()}

	return fState, nil
}

// SaveFinalizedState saves the finalized slot, root and state into memory to be used by state gen service.
// This used for migration at the correct start slot and used for hot state play back to ensure
// lower bound to start is always at the last finalized state.
func (s *State) SaveFinalizedState(fSlot uint64, fRoot [32]byte, fState *state.BeaconState) {
	s.finalizedInfo.lock.Lock()
	defer s.finalizedInfo.lock.Unlock()
	s.finalizedInfo.root = fRoot
	s.finalizedInfo.state = fState.Copy()
	s.finalizedInfo.slot = fSlot
}

// Returns true if input root equals to cached finalized root.
func (s *State) isFinalizedRoot(r [32]byte) bool {
	s.finalizedInfo.lock.RLock()
	defer s.finalizedInfo.lock.RUnlock()
	return r == s.finalizedInfo.root
}

// Returns the cached and copied finalized state.
func (s *State) finalizedState() *state.BeaconState {
	s.finalizedInfo.lock.RLock()
	defer s.finalizedInfo.lock.RUnlock()
	return s.finalizedInfo.state.Copy()
}
