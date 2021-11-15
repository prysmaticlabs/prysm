// Package stategen defines functions to regenerate beacon chain states
// by replaying blocks from a stored state checkpoint, useful for
// optimization and reducing a beacon node's resource consumption.
package stategen

import (
	"context"
	"errors"
	"sync"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"go.opencensus.io/trace"
)

var defaultHotStateDBInterval types.Slot = 128

// StateManager represents a management object that handles the internal
// logic of maintaining both hot and cold states in DB.
type StateManager interface {
	Resume(ctx context.Context, fState state.BeaconState) (state.BeaconState, error)
	SaveFinalizedState(fSlot types.Slot, fRoot [32]byte, fState state.BeaconState)
	MigrateToCold(ctx context.Context, fRoot [32]byte) error
	ReplayBlocks(ctx context.Context, state state.BeaconState, signed []block.SignedBeaconBlock, targetSlot types.Slot) (state.BeaconState, error)
	LoadBlocks(ctx context.Context, startSlot, endSlot types.Slot, endBlockRoot [32]byte) ([]block.SignedBeaconBlock, error)
	HasState(ctx context.Context, blockRoot [32]byte) (bool, error)
	HasStateInCache(ctx context.Context, blockRoot [32]byte) (bool, error)
	StateByRoot(ctx context.Context, blockRoot [32]byte) (state.BeaconState, error)
	StateByRootInitialSync(ctx context.Context, blockRoot [32]byte) (state.BeaconState, error)
	StateBySlot(ctx context.Context, slot types.Slot) (state.BeaconState, error)
	RecoverStateSummary(ctx context.Context, blockRoot [32]byte) (*ethpb.StateSummary, error)
	SaveState(ctx context.Context, root [32]byte, st state.BeaconState) error
	ForceCheckpoint(ctx context.Context, root []byte) error
	EnableSaveHotStateToDB(_ context.Context)
	DisableSaveHotStateToDB(ctx context.Context) error
}

// State is a concrete implementation of StateManager.
type State struct {
	beaconDB                db.NoHeadAccessDatabase
	slotsPerArchivedPoint   types.Slot
	hotStateCache           *hotStateCache
	finalizedInfo           *finalizedInfo
	epochBoundaryStateCache *epochBoundaryState
	saveHotStateDB          *saveHotStateDbConfig
}

// This tracks the config in the event of long non-finality,
// how often does the node save hot states to db? what are
// the saved hot states in db?... etc
type saveHotStateDbConfig struct {
	enabled         bool
	lock            sync.Mutex
	duration        types.Slot
	savedStateRoots [][32]byte
}

// This tracks the finalized point. It's also the point where slot and the block root of
// cold and hot sections of the DB splits.
type finalizedInfo struct {
	slot  types.Slot
	root  [32]byte
	state state.BeaconState
	lock  sync.RWMutex
}

// New returns a new state management object.
func New(beaconDB db.NoHeadAccessDatabase) *State {
	return &State{
		beaconDB:                beaconDB,
		hotStateCache:           newHotStateCache(),
		finalizedInfo:           &finalizedInfo{slot: 0, root: params.BeaconConfig().ZeroHash},
		slotsPerArchivedPoint:   params.BeaconConfig().SlotsPerArchivedPoint,
		epochBoundaryStateCache: newBoundaryStateCache(),
		saveHotStateDB: &saveHotStateDbConfig{
			duration: defaultHotStateDBInterval,
		},
	}
}

// Resume resumes a new state management object from previously saved finalized check point in DB.
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
