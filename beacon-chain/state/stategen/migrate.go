package stategen

import (
	"context"

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

	log.WithFields(logrus.Fields{
		"lastFinalizedSlot": s.splitInfo.slot,
		"newFinalizedSlot": finalizedSlot,
	}).Info("Hot to cold migration")

	for i := s.splitInfo.slot + 1; i <= finalizedSlot; i++ {
		if i % s.slotsPerArchivedPoint == 0 {
			if i == s.finalized.state.Slot() {
				log.Infof("Save state for slot %d", i)
			} else {
				log.Infof("Save regenerated state for slot %d %d", s.finalized.state.Slot(), i)
			}
		}
	}

	// Update the split slot and root.
	s.splitInfo = &splitSlotAndRoot{slot: finalizedSlot, root: finalizedRoot}

	// Update finalized state in memory.
	fState, err := s.beaconDB.State(ctx, finalizedRoot)
	if err != nil {
		return err
	}
	s.finalized.lock.Lock()
	s.finalized.state = fState
	s.finalized.lock.Unlock()

	return nil
}

