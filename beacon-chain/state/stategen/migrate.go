package stategen

import (
	"context"
	"encoding/hex"

	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// MigrateToCold advances the split point in between the cold and hot state sections.
// It moves the recent finalized states from the hot section to the cold section and
// only preserve the ones that's on archived point.
func (s *State) MigrateToCold(ctx context.Context, finalizedSlot uint64, finalizedRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "stateGen.MigrateToCold")
	defer span.End()

	// Verify migration is sensible. The new finalized point must increase the current split slot, and
	// on an epoch boundary for hot state summary scheme to work.
	currentSplitSlot := s.splitInfo.slot
	if currentSplitSlot > finalizedSlot {
		return nil
	}

	// Migrate all state summary objects from cache to DB.
	if err := s.beaconDB.SaveStateSummaries(ctx, s.stateSummaryCache.GetAll()); err != nil {
		return err
	}
	s.stateSummaryCache.Clear()

	lastArchivedIndex, err := s.beaconDB.LastArchivedIndex(ctx)
	if err != nil {
		return err
	}

	// Move the states between split slot to finalized slot from hot section to the cold section.
	filter := filters.NewFilter().SetStartSlot(currentSplitSlot).SetEndSlot(finalizedSlot - 1)
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

		archivedPointIndex := stateSummary.Slot / s.slotsPerArchivedPoint
		nextArchivedPointSlot := (lastArchivedIndex + 1) * s.slotsPerArchivedPoint
		// Only migrate if current slot is equal to or greater than next archived point slot.
		if stateSummary.Slot >= nextArchivedPointSlot {
			if !s.beaconDB.HasState(ctx, r) {
				continue
			}
			if err := s.beaconDB.SaveArchivedPointRoot(ctx, r, archivedPointIndex); err != nil {
				return err
			}
			if err := s.beaconDB.SaveLastArchivedIndex(ctx, archivedPointIndex); err != nil {
				return err
			}
			lastArchivedIndex++
			log.WithFields(logrus.Fields{
				"slot":         stateSummary.Slot,
				"archiveIndex": archivedPointIndex,
				"root":         hex.EncodeToString(bytesutil.Trunc(r[:])),
			}).Info("Saved archived point during state migration")
		} else {
			// Do not delete the current finalized state in case user wants to
			// switch back to old state service, deleting the recent finalized state
			// could cause issue switching back.
			lastArchivedIndexRoot := s.beaconDB.LastArchivedIndexRoot(ctx)
			if s.beaconDB.HasState(ctx, r) && r != lastArchivedIndexRoot && r != finalizedRoot {
				if err := s.beaconDB.DeleteState(ctx, r); err != nil {
					// For whatever reason if node is unable to delete a state due to
					// state is finalized, it is more reasonable to continue than to exit.
					log.Warnf("Unable to delete state during migration: %v", err)
					continue
				}
				log.WithFields(logrus.Fields{
					"slot": stateSummary.Slot,
					"root": hex.EncodeToString(bytesutil.Trunc(r[:])),
				}).Info("Deleted state during migration")
			}
		}
	}

	// Update the split slot and root.
	s.splitInfo = &splitSlotAndRoot{slot: finalizedSlot, root: finalizedRoot}
	log.WithFields(logrus.Fields{
		"slot": s.splitInfo.slot,
		"root": hex.EncodeToString(bytesutil.Trunc(s.splitInfo.root[:])),
	}).Info("Set hot and cold state split point")

	return nil
}
