package stategen

import (
	"context"
	"encoding/hex"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/sirupsen/logrus"
)

// This saves a post finalized beacon state in the hot DB. On the epoch boundary,
// it saves a full state. On an intermediate slot, it saves a back pointer to the
// nearest epoch boundary state.
func (s *Service) saveHotState(ctx context.Context, blockRoot [32]byte, state *state.BeaconState) error {
	// On an epoch boundary, saves the whole state.
	if helpers.IsEpochStart(state.Slot()) {
		if err := s.beaconDB.SaveState(ctx, state, blockRoot); err != nil {
			return err
		}
		log.WithFields(logrus.Fields{
			"slot": state.Slot(),
			"blockRoot": hex.EncodeToString(bytesutil.Trunc(blockRoot[:]))}).Debug("Saved full state on epoch boundary")
	}

	// On an intermediate slot, save the state summary.
	if err := s.beaconDB.SaveHotStateSummary(ctx, &pb.HotStateSummary{
		Slot:         state.Slot(),
		LatestRoot:   blockRoot[:],
		BoundaryRoot: nil,
	}); err != nil {
		return err
	}

	// Store the state in the cache.

	return nil
}

// This loads a post finalized beacon state from the hot DB. If necessary it will
// replay blocks from the nearest epoch boundary.
func (s *Service) loadHotState(ctx context.Context, blockRoot [32]byte) (*state.BeaconState, error) {
	// Load the cache

	summary, err := s.beaconDB.HotStateSummary(ctx, blockRoot[:])
	if err != nil {
		return nil, err
	}
	targetSlot := summary.Slot

	boundaryState, err := s.beaconDB.State(ctx, bytesutil.ToBytes32(summary.BoundaryRoot))
	if err != nil {
		return nil, err
	}

	// Don't need to replay the blocks if we're already on an epoch boundary.
	var hotState *state.BeaconState
	if helpers.IsEpochStart(targetSlot) {
		hotState = boundaryState
	} else {
		blks, err := s.loadBlocks(ctx, boundaryState.Slot(), targetSlot, bytesutil.ToBytes32(summary.LatestRoot))
		if err != nil {
			return nil, err
		}
		hotState, err = s.replayBlocks(ctx, boundaryState, blks, targetSlot)
		if err != nil {
			return nil, err
		}
	}

	// Save the cache

	return hotState, nil
}
