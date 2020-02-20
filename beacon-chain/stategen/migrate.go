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
)

// MigrateToCold advances the split slot point between the cold and hot sections.
// It moves the new finalized states from the hot section to the cold section.
func (s *State) MigrateToCold(ctx context.Context, finalizedState *state.BeaconState) error {
	// Verify migration is sensible. The new finalized point must increase the current split slot, and
	// on an epoch boundary for hot state summary scheme to work.
	currentSplitSlot := s.splitSlot
	if currentSplitSlot > finalizedState.Slot() {
		return nil
	}
	if !helpers.IsEpochStart(finalizedState.Slot()) {
		return nil
	}

	log.WithField("slot", finalizedState.Slot()).Info("Hot to cold state migration started")
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
		if hotStateSummary == nil {
			continue
		}

		if hotStateSummary.Slot%s.slotsPerArchivePoint == 0 {
			// Since the state was prev saved, from migration's standpoint,
			// all we have to save now is just the archive point.
			archivePointIndex := hotStateSummary.Slot / s.slotsPerArchivePoint
			if err := s.beaconDB.SaveArchivePoint(ctx, r, archivePointIndex); err != nil {
				return err
			}
			archivePointSaved.Inc()
			coldStateSaved.Inc()
			log.Info("Saving archive point ", hotStateSummary.Slot, archivePointIndex, hex.EncodeToString(bytesutil.Trunc(r[:])))
		} else {
			// Delete the states that's not on the archive point.
			if s.beaconDB.HasState(ctx, r) {
				if err := s.beaconDB.DeleteState(ctx, r); err != nil {
					return err
				}
				log.Info("Deleted state ", hotStateSummary.Slot, hex.EncodeToString(bytesutil.Trunc(r[:])))
			}
		}
		// Migrate state summary from hot to cold.
		if err := s.beaconDB.SaveColdStateSummary(ctx, r, &pb.ColdStateSummary{Slot: hotStateSummary.Slot}); err != nil {
			return err
		}
		if err := s.beaconDB.DeleteHotStateSummary(ctx, r); err != nil {
			return err
		}

		coldSummarySaved.Inc()
		hotStateSaved.Dec()
		hotSummarySaved.Dec()
	}

	// Update the split slot.
	s.splitSlot = finalizedState.Slot()

	log.WithField("slot", finalizedState.Slot()).Info("Hot to cold state migration completed")

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
