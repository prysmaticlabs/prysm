package stategen

import (
	"context"
	"encoding/hex"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// MigrateToCold advances the split point in between the cold and hot state sections.
// It moves the recent finalized states from the hot section to the cold section and
// only preserve the ones that's on archived point.
func (s *State) MigrateToCold(ctx context.Context, finalizedState *state.BeaconState, finalizedRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "stateGen.MigrateToCold")
	defer span.End()

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
		stateSummary, err := s.beaconDB.StateSummary(ctx, r)
		if err != nil {
			return err
		}
		if stateSummary == nil || stateSummary.Slot == 0 {
			continue
		}

		if stateSummary.Slot%s.slotsPerArchivedPoint == 0 {
			archivePointIndex := stateSummary.Slot / s.slotsPerArchivedPoint
			if s.beaconDB.HasState(ctx, r) {
				hotState, err := s.beaconDB.State(ctx, r)
				if err != nil {
					return err
				}
				if err := s.beaconDB.SaveArchivedPointState(ctx, hotState.Copy(), archivePointIndex); err != nil {
					return err
				}
			} else {
				hotState, err := s.ComputeStateUpToSlot(ctx, stateSummary.Slot)
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

			log.WithFields(logrus.Fields{
				"slot":         stateSummary.Slot,
				"archiveIndex": archivePointIndex,
				"root":         hex.EncodeToString(bytesutil.Trunc(r[:])),
			}).Info("Saved archived point during state migration")
		}

		if s.beaconDB.HasState(ctx, r) {
			if err := s.beaconDB.DeleteState(ctx, r); err != nil {
				return err
			}
			log.WithFields(logrus.Fields{
				"slot": stateSummary.Slot,
				"root": hex.EncodeToString(bytesutil.Trunc(r[:])),
			}).Info("Deleted state during migration")
		}

		s.deleteEpochBoundaryRoot(stateSummary.Slot)
	}

	// Update the split slot and root.
	s.splitInfo = &splitSlotAndRoot{slot: finalizedState.Slot(), root: finalizedRoot}
	log.WithFields(logrus.Fields{
		"slot": s.splitInfo.slot,
		"root": hex.EncodeToString(bytesutil.Trunc(s.splitInfo.root[:])),
	}).Info("Set hot and cold state split point")

	return nil
}
