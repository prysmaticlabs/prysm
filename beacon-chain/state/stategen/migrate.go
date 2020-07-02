package stategen

import (
	"context"
	"encoding/hex"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// MigrateToCold advances the split point in between the cold and hot state sections.
// It moves the recent finalized states from the hot section to the cold section and
// only preserve the ones that's on archived point.
func (s *State) MigrateToCold(ctx context.Context, fSlot uint64, fRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "stateGen.MigrateToCold")
	defer span.End()

	lastFSlot := s.splitInfo.slot
	lastFRoot := s.splitInfo.root
	if lastFSlot > fSlot {
		return nil
	}

	s.finalized.lock.RLock()
	for i := lastFSlot; i < fSlot; i++ {
		if i%s.slotsPerArchivedPoint == 0 && i != 0 {
			aIndex := i / s.slotsPerArchivedPoint
			if err := s.beaconDB.SaveState(ctx, s.finalized.state, lastFRoot); err != nil {
				return err
			}
			if err := s.beaconDB.SaveArchivedPointRoot(ctx, lastFRoot, aIndex); err != nil {
				return err
			}
			if err := s.beaconDB.SaveLastArchivedIndex(ctx, aIndex); err != nil {
				return err
			}
			log.WithFields(
				logrus.Fields{
					"archivedSlot":  s.finalized.state.Slot(),
					"archivedIndex": aIndex,
					"archivedRoot":  hex.EncodeToString(bytesutil.Trunc(lastFRoot[:])),
				}).Info("Saved archived state in DB")
		}
	}
	s.finalized.lock.RUnlock()

	// Update the split slot and root.
	s.splitInfo = &splitSlotAndRoot{slot: fSlot, root: fRoot}
	// Migrate all state summary objects from cache to DB.
	if err := s.beaconDB.SaveStateSummaries(ctx, s.stateSummaryCache.GetAll()); err != nil {
		return err
	}
	s.stateSummaryCache.Clear()
	// Update finalized state in memory.
	fState, err := s.StateByRoot(ctx, fRoot)
	if err != nil {
		return err
	}
	s.finalized.lock.Lock()
	s.finalized.state = fState
	s.finalized.lock.Unlock()

	return nil
}
