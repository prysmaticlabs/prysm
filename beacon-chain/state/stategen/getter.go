package stategen

import (
	"context"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// HasState returns true if the state exists in cache or in DB.
func (s *State) HasState(ctx context.Context, blockRoot [32]byte) (bool, error) {
	has, err := s.HasStateInCache(ctx, blockRoot)
	if err != nil {
		return false, err
	}
	if has {
		return true, nil
	}
	return s.beaconDB.HasState(ctx, blockRoot), nil
}

// HasStateInCache returns true if the state exists in cache.
func (s *State) HasStateInCache(ctx context.Context, blockRoot [32]byte) (bool, error) {
	if s.hotStateCache.has(blockRoot) {
		return true, nil
	}
	_, has, err := s.epochBoundaryStateCache.getByRoot(blockRoot)
	if err != nil {
		return false, err
	}
	return has, nil
}

// StateByRoot retrieves the state using input block root.
func (s *State) StateByRoot(ctx context.Context, blockRoot [32]byte) (iface.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.StateByRoot")
	defer span.End()

	// Genesis case. If block root is zero hash, short circuit to use genesis cachedState stored in DB.
	if blockRoot == params.BeaconConfig().ZeroHash {
		return s.beaconDB.State(ctx, blockRoot)
	}
	return s.loadStateByRoot(ctx, blockRoot)
}

// StateByRootInitialSync retrieves the state from the DB for the initial syncing phase.
// It assumes initial syncing using a block list rather than a block tree hence the returned
// state is not copied.
// It invalidates cache for parent root because pre state will get mutated.
// Do not use this method for anything other than initial syncing purpose or block tree is applied.
func (s *State) StateByRootInitialSync(ctx context.Context, blockRoot [32]byte) (iface.BeaconState, error) {
	// Genesis case. If block root is zero hash, short circuit to use genesis state stored in DB.
	if blockRoot == params.BeaconConfig().ZeroHash {
		return s.beaconDB.State(ctx, blockRoot)
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
		return nil, errors.Wrap(err, "could not load blocks")
	}
	startState, err = s.ReplayBlocks(ctx, startState, blks, summary.Slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not replay blocks")
	}

	return startState, nil
}

// StateBySlot retrieves the state using input slot.
func (s *State) StateBySlot(ctx context.Context, slot types.Slot) (iface.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.StateBySlot")
	defer span.End()

	return s.loadStateBySlot(ctx, slot)
}

// This returns the state summary object of a given block root, it first checks the cache
// then checks the DB. An error is returned if state summary object is nil.
func (s *State) stateSummary(ctx context.Context, blockRoot [32]byte) (*pb.StateSummary, error) {
	var summary *pb.StateSummary
	var err error

	summary, err = s.beaconDB.StateSummary(ctx, blockRoot)
	if err != nil {
		return nil, err
	}

	if summary == nil {
		return s.RecoverStateSummary(ctx, blockRoot)
	}
	return summary, nil
}

// RecoverStateSummary recovers state summary object of a given block root by using the saved block in DB.
func (s *State) RecoverStateSummary(ctx context.Context, blockRoot [32]byte) (*pb.StateSummary, error) {
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
	return nil, errors.New("could not find block in DB")
}

// This loads a beacon state from either the cache or DB then replay blocks up the requested block root.
func (s *State) loadStateByRoot(ctx context.Context, blockRoot [32]byte) (iface.BeaconState, error) {
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
	if s.beaconDB.HasState(ctx, blockRoot) {
		return s.beaconDB.State(ctx, blockRoot)
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

	// Return state early if we are retrieving it from our finalized state cache.
	if startState.Slot() == targetSlot {
		return startState, nil
	}

	blks, err := s.LoadBlocks(ctx, startState.Slot()+1, targetSlot, bytesutil.ToBytes32(summary.Root))
	if err != nil {
		return nil, errors.Wrap(err, "could not load blocks for hot state using root")
	}

	replayBlockCount.Observe(float64(len(blks)))

	return s.ReplayBlocks(ctx, startState, blks, targetSlot)
}

// This loads a state by slot.
func (s *State) loadStateBySlot(ctx context.Context, slot types.Slot) (iface.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.loadStateBySlot")
	defer span.End()

	// Return genesis state if slot is 0.
	if slot == 0 {
		return s.beaconDB.GenesisState(ctx)
	}

	// Gather the last saved block root and the slot number.
	lastValidRoot, lastValidSlot, err := s.lastSavedBlock(ctx, slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get last valid block for hot state using slot")
	}

	replayStartState, err := s.loadStateByRoot(ctx, lastValidRoot)
	if err != nil {
		return nil, err
	}

	if lastValidSlot < slot {
		replayStartState, err = processSlotsStateGen(ctx, replayStartState, slot)
		if err != nil {
			return nil, err
		}
	}
	return replayStartState, nil
}

// This returns the highest available ancestor state of the input block root.
// It recursively look up block's parent until a corresponding state of the block root
// is found in the caches or DB.
//
// There's three ways to derive block parent state:
// 1.) block parent state is the last finalized state
// 2.) block parent state is the epoch boundary state and exists in epoch boundary cache.
// 3.) block parent state is in DB.
func (s *State) lastAncestorState(ctx context.Context, root [32]byte) (iface.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.lastAncestorState")
	defer span.End()

	if s.isFinalizedRoot(root) && s.finalizedState() != nil {
		return s.finalizedState(), nil
	}

	b, err := s.beaconDB.Block(ctx, root)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, errUnknownBlock
	}

	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		// Is the state a genesis state.
		parentRoot := bytesutil.ToBytes32(b.Block.ParentRoot)
		if parentRoot == params.BeaconConfig().ZeroHash {
			return s.beaconDB.GenesisState(ctx)
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
		if s.beaconDB.HasState(ctx, parentRoot) {
			return s.beaconDB.State(ctx, parentRoot)
		}
		b, err = s.beaconDB.Block(ctx, parentRoot)
		if err != nil {
			return nil, err
		}
		if b == nil {
			return nil, errUnknownBlock
		}
	}
}
