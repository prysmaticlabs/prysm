package stategen

import (
	"context"
	"encoding/hex"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

// MigrateToCold advances the split slot point between the cold and hot sections.
// It moves the new finalized states from the hot section to the cold section.
func (s *State) MigrateToCold(ctx context.Context, finalizedState *state.BeaconState, finalizedRoot [32]byte) error {
	// Verify migration is sensible. The new finalized point must increase the current split slot, and
	// on an epoch boundary for hot state summary scheme to work.
	currentSplitSlot := s.splitInfo.slot
	if currentSplitSlot > finalizedState.Slot() {
		return nil
	}
	if !helpers.IsEpochStart(finalizedState.Slot()) {
		return nil
	}

	// Move the states between split slot to finalized slot from hot section to the cold section.
	filter := filters.NewFilter().SetStartSlot(currentSplitSlot).SetEndSlot(finalizedState.Slot() - 1)
	blockRoots, err := s.beaconDB.BlockRoots(ctx, filter)
	if err != nil {
		return err
	}

	for _, r := range blockRoots {
		hotStateSummary, err := s.beaconDB.HotStateSummary(ctx, r)
		if err != nil {
			return err
		}
		if hotStateSummary == nil || hotStateSummary.Slot == 0 {
			continue
		}

		if hotStateSummary.Slot%s.slotsPerArchivePoint == 0 {
			archivePointIndex := hotStateSummary.Slot / s.slotsPerArchivePoint
			if s.beaconDB.HasState(ctx, r) {
				hotState, err := s.beaconDB.State(ctx, r)
				if err != nil {
					return err
				}
				if err := s.beaconDB.SaveArchivedPointState(ctx, hotState.Copy(), archivePointIndex); err != nil {
					return err
				}
			} else {
				hotState, err := s.ComputeStateUpToSlot(ctx, hotStateSummary.Slot)
				if err != nil {
					return err
				}
				if err := s.beaconDB.SaveArchivedPointState(ctx, hotState.Copy(), archivePointIndex); err != nil {
					return err
				}
			}
			if err := s.beaconDB.SaveArchivedPointRoot(ctx, r, archivePointIndex); err != nil {
				return err
			}

			archivePointSaved.Inc()
			log.WithFields(logrus.Fields{
				"slot":         hotStateSummary.Slot,
				"archiveIndex": archivePointIndex,
				"root":         hex.EncodeToString(bytesutil.Trunc(r[:])),
			}).Info("Saved archive point during state migration")
		}

		if s.beaconDB.HasState(ctx, r) {
			if err := s.beaconDB.DeleteState(ctx, r); err != nil {
				return err
			}
			hotStateSaved.Dec()
			log.WithFields(logrus.Fields{
				"slot": hotStateSummary.Slot,
				"root": hex.EncodeToString(bytesutil.Trunc(r[:])),
			}).Info("Deleted state during migration")
		}

		// Migrate state summary from hot to cold.
		if err := s.beaconDB.SaveColdStateSummary(ctx, r, &pb.ColdStateSummary{Slot: hotStateSummary.Slot}); err != nil {
			return err
		}
		if err := s.beaconDB.DeleteHotStateSummary(ctx, r); err != nil {
			return err
		}
		s.deleteEpochBoundaryRoot(hotStateSummary.Slot)

		coldSummarySaved.Inc()
		hotSummarySaved.Dec()
	}

	// Update the split slot and root.
	s.splitInfo = &splitSlotAndRoot{slot: finalizedState.Slot(), root: finalizedRoot}

	return nil
}

// This verifies the archive point frequency is valid. It checks the interval
// is a divisor of the number of slots per historical root and divisible by
// the number of slots per epoch. This ensures we have at least one
// archive point within range of our state root history when iterating
// backwards. It also ensures the archive points align with hot state summaries
// which makes it quicker to migrate hot to cold.
func verifySlotsPerArchivePoint(slotsPerArchivePoint uint64) bool {
	return slotsPerArchivePoint > 0 &&
		slotsPerArchivePoint%params.BeaconConfig().SlotsPerHistoricalRoot == 0 &&
		slotsPerArchivePoint%params.BeaconConfig().SlotsPerEpoch == 0
}
