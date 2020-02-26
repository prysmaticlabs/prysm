package stategen

import (
	"context"
	"sync"

	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// State represents a management object that handles the internal
// logic of maintaining both hot and cold states in DB.
type State struct {
	beaconDB                db.NoHeadAccessDatabase
	splitInfo               *splitSlotAndRoot
	slotsPerArchivePoint    uint64
	epochBoundarySlotToRoot map[uint64][32]byte
	epochBoundaryLock       sync.RWMutex
	hotStateCache           *cache.HotStateCache
}

type splitSlotAndRoot struct {
	slot uint64
	root [32]byte
}

// New returns a new state management object.
func New(db db.NoHeadAccessDatabase) *State {
	return &State{
		beaconDB: db,
		//slotsPerArchivePoint: uint64(flags.Get().SlotsPerArchivePoint),
		slotsPerArchivePoint:    128,
		epochBoundarySlotToRoot: make(map[uint64][32]byte),
		splitInfo:               &splitSlotAndRoot{slot: 0, root: params.BeaconConfig().ZeroHash},
		hotStateCache:           cache.NewHotStateCache(),
	}
}

// Resume sets up the new state management object from previously saved finalized check point in DB.
func (s *State) Resume(ctx context.Context, finalizedRoot [32]byte) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.Resume")
	defer span.End()

	finalizedState, err := s.beaconDB.State(ctx, finalizedRoot)
	if err != nil {
		return nil, err
	}
	s.splitInfo = &splitSlotAndRoot{slot: finalizedState.Slot(), root: finalizedRoot}
	if err := s.beaconDB.SaveColdStateSummary(ctx, finalizedRoot, &pb.ColdStateSummary{Slot: finalizedState.Slot()}); err != nil {
		return nil, err
	}

	s.setEpochBoundaryRoot(finalizedState.Slot(), finalizedRoot)

	return finalizedState, nil
}

// This sets epoch boundary slot to root mapping.
func (s *State) setEpochBoundaryRoot(slot uint64, root [32]byte) {
	s.epochBoundaryLock.Lock()
	defer s.epochBoundaryLock.Unlock()
	s.epochBoundarySlotToRoot[slot] = root
}

// This reads epoch boundary slot to root mapping.
func (s *State) epochBoundaryRoot(slot uint64) ([32]byte, bool) {
	s.epochBoundaryLock.RLock()
	defer s.epochBoundaryLock.RUnlock()
	r, ok := s.epochBoundarySlotToRoot[slot]
	return r, ok
}

// This deletes an entry of epoch boundary slot to root mapping.
func (s *State) deleteEpochBoundaryRoot(slot uint64) {
	s.epochBoundaryLock.Lock()
	defer s.epochBoundaryLock.Unlock()
	delete(s.epochBoundarySlotToRoot, slot)
}
