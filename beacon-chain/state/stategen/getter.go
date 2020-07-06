package stategen

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// StateByRoot retrieves the state from DB using input block root.
// It retrieves state from the hot section if the state summary slot
// is below the split point cut off.
func (s *State) StateByRoot(ctx context.Context, blockRoot [32]byte) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.StateByRoot")
	defer span.End()

	// Genesis case. If block root is zero hash, short circuit to use genesis cachedState stored in DB.
	if blockRoot == params.BeaconConfig().ZeroHash {
		return s.beaconDB.State(ctx, blockRoot)
	}

	// Short cut if the cachedState is already in the DB.
	if s.beaconDB.HasState(ctx, blockRoot) {
		return s.beaconDB.State(ctx, blockRoot)
	}

	summary, err := s.stateSummary(ctx, blockRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get cachedState summary")
	}

	if summary.Slot < s.finalizedInfo.slot {
		return s.loadColdStateByRoot(ctx, blockRoot)
	}

	return s.loadHotStateByRoot(ctx, blockRoot)
}

// StateByRootInitialSync retrieves the state from the DB for the initial syncing phase.
// It assumes initial syncing using a block list rather than a block tree hence the returned
// state is not copied.
// Do not use this method for anything other than initial syncing purpose or block tree is applied.
func (s *State) StateByRootInitialSync(ctx context.Context, blockRoot [32]byte) (*state.BeaconState, error) {
	// Genesis case. If block root is zero hash, short circuit to use genesis state stored in DB.
	if blockRoot == params.BeaconConfig().ZeroHash {
		return s.beaconDB.State(ctx, blockRoot)
	}

	if s.hotStateCache.Has(blockRoot) {
		return s.hotStateCache.GetWithoutCopy(blockRoot), nil
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
		return nil, errUnknownState
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
		return nil, errors.Wrap(err, "could not load blocks for hot state using root")
	}
	startState, err = s.ReplayBlocks(ctx, startState, blks, summary.Slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not replay blocks for hot state using root")
	}

	return startState, nil
}

// StateBySlot retrieves the state from DB using input slot.
// It retrieves state from the cold section if the input slot
// is below the split point cut off.
// Note: `StateByRoot` is preferred over this. Retrieving state
// by root `StateByRoot` is more performant than retrieving by slot.
func (s *State) StateBySlot(ctx context.Context, slot uint64) (*state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.StateBySlot")
	defer span.End()

	if slot < s.finalizedInfo.slot {
		return s.loadColdStateBySlot(ctx, slot)
	}

	return s.loadHotStateBySlot(ctx, slot)
}

// StateSummaryExists returns true if the corresponding state summary of the input block root either
// exists in the DB or in the cache.
func (s *State) StateSummaryExists(ctx context.Context, blockRoot [32]byte) bool {
	return s.beaconDB.HasStateSummary(ctx, blockRoot) || s.stateSummaryCache.Has(blockRoot)
}

// This returns the state summary object of a given block root, it first checks the cache
// then checks the DB. An error is returned if state summary object is nil.
func (s *State) stateSummary(ctx context.Context, blockRoot [32]byte) (*pb.StateSummary, error) {
	var summary *pb.StateSummary
	var err error
	if s.stateSummaryCache.Has(blockRoot) {
		summary = s.stateSummaryCache.Get(blockRoot)
	} else {
		summary, err = s.beaconDB.StateSummary(ctx, blockRoot)
		if err != nil {
			return nil, err
		}
	}
	if summary == nil {
		return s.recoverStateSummary(ctx, blockRoot)
	}
	return summary, nil
}

// This recovers state summary object of a given block root by using the saved block in DB.
func (s *State) recoverStateSummary(ctx context.Context, blockRoot [32]byte) (*pb.StateSummary, error) {
	if s.beaconDB.HasBlock(ctx, blockRoot) {
		b, err := s.beaconDB.Block(ctx, blockRoot)
		if err != nil {
			return nil, err
		}
		summary := &pb.StateSummary{Slot: b.Block.Slot, Root: blockRoot[:]}
		if err := s.beaconDB.SaveStateSummary(ctx, summary); err != nil {
			return nil, err
		}
		return summary, nil
	}
	return nil, errUnknownStateSummary
}
