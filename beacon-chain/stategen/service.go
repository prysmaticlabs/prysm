package stategen

import (
	"context"
	"sync"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// State represents a management object that handles the internal
// logic of maintaining both hot and cold states in DB.
type State struct {
	beaconDB                db.NoHeadAccessDatabase
	splitSlot               uint64
	slotsPerArchivePoint    uint64
	epochBoundarySlotToRoot map[uint64][32]byte
	epochBoundaryLock       sync.RWMutex
}

// New returns a new state management object.
func New(db db.NoHeadAccessDatabase) *State {
	return &State{
		beaconDB: db,
		//slotsPerArchivePoint: uint64(flags.Get().SlotsPerArchivePoint),
		slotsPerArchivePoint:    128,
		epochBoundarySlotToRoot: make(map[uint64][32]byte),
	}
}

// Resume sets up the new state management object from previously saved finalized check point in DB.
func (s *State) Resume(ctx context.Context, finalizedRoot [32]byte) (*state.BeaconState, error) {
	finalizedState, err := s.beaconDB.State(ctx, finalizedRoot)
	if err != nil {
		return nil, err
	}
	s.splitSlot = finalizedState.Slot()
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
