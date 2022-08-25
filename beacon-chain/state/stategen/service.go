// Package stategen defines functions to regenerate beacon chain states
// by replaying blocks from a stored state checkpoint, useful for
// optimization and reducing a beacon node's resource consumption.
package stategen

import (
	"context"
	"errors"
	"sync"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/sync/backfill"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	"go.opencensus.io/trace"
)

var defaultHotStateDBInterval types.Slot = 128

// StateManager represents a management object that handles the internal
// logic of maintaining both hot and cold states in DB.
type StateManager interface {
	Resume(ctx context.Context, fState state.BeaconState) (state.BeaconState, error)
	DisableSaveHotStateToDB(ctx context.Context) error
	EnableSaveHotStateToDB(_ context.Context)
	HasState(ctx context.Context, blockRoot [32]byte) (bool, error)
	DeleteStateFromCaches(ctx context.Context, blockRoot [32]byte) error
	ForceCheckpoint(ctx context.Context, root []byte) error
	SaveState(ctx context.Context, blockRoot [32]byte, st state.BeaconState) error
	SaveFinalizedState(fSlot types.Slot, fRoot [32]byte, fState state.BeaconState)
	MigrateToCold(ctx context.Context, fRoot [32]byte) error
	StateByRoot(ctx context.Context, blockRoot [32]byte) (state.BeaconState, error)
	StateByRootIfCachedNoCopy(blockRoot [32]byte) state.BeaconState
	StateByRootInitialSync(ctx context.Context, blockRoot [32]byte) (state.BeaconState, error)
}

// State is a concrete implementation of StateManager.
type State struct {
	beaconDB                db.NoHeadAccessDatabase
	slotsPerArchivedPoint   types.Slot
	hotStateCache           *hotStateCache
	finalizedInfo           *finalizedInfo
	epochBoundaryStateCache *epochBoundaryState
	saveHotStateDB          *saveHotStateDbConfig
	backfillStatus          *backfill.Status
}

// This tracks the config in the event of long non-finality,
// how often does the node save hot states to db? what are
// the saved hot states in db?... etc
type saveHotStateDbConfig struct {
	enabled                 bool
	lock                    sync.Mutex
	duration                types.Slot
	blockRootsOfSavedStates [][32]byte
}

// This tracks the finalized point. It's also the point where slot and the block root of
// cold and hot sections of the DB splits.
type finalizedInfo struct {
	slot  types.Slot
	root  [32]byte
	state state.BeaconState
	lock  sync.RWMutex
}

// StateGenOption is a functional option for controlling the initialization of a *State value
type StateGenOption func(*State)

func WithBackfillStatus(bfs *backfill.Status) StateGenOption {
	return func(sg *State) {
		sg.backfillStatus = bfs
	}
}

// New returns a new state management object.
func New(beaconDB db.NoHeadAccessDatabase, opts ...StateGenOption) *State {
	s := &State{
		beaconDB:                beaconDB,
		hotStateCache:           newHotStateCache(),
		finalizedInfo:           &finalizedInfo{slot: 0, root: params.BeaconConfig().ZeroHash},
		slotsPerArchivedPoint:   params.BeaconConfig().SlotsPerArchivedPoint,
		epochBoundaryStateCache: newBoundaryStateCache(),
		saveHotStateDB: &saveHotStateDbConfig{
			duration: defaultHotStateDBInterval,
		},
	}
	for _, o := range opts {
		o(s)
	}

	return s
}

// Resume resumes a new state management object from previously saved finalized checkpoint in DB.
func (s *State) Resume(ctx context.Context, fState state.BeaconState) (state.BeaconState, error) {
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

	if fState == nil || fState.IsNil() {
		return nil, errors.New("finalized state is nil")
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
func (s *State) SaveFinalizedState(fSlot types.Slot, fRoot [32]byte, fState state.BeaconState) {
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
func (s *State) finalizedState() state.BeaconState {
	s.finalizedInfo.lock.RLock()
	defer s.finalizedInfo.lock.RUnlock()
	return s.finalizedInfo.state.Copy()
}
