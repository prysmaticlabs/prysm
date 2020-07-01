package stategen

import (
	"context"
	"encoding/hex"

	"github.com/prysmaticlabs/go-ssz"
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

	log.WithFields(logrus.Fields{
		"lastFinalizedSlot": s.splitInfo.slot,
		"newFinalizedSlot": finalizedSlot,
	}).Info("Hot to cold migration")

	for i := s.splitInfo.slot; i <= finalizedSlot; i++ {
		if i % s.slotsPerArchivedPoint == 0 {
			log.Infof("Save state for slot %d", i)
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

// This recovers the last archived point. By passing in the current archived point, this recomputes
// the state of last skipped archived point and save the missing state, archived point root, archived index to the DB.
func (s *State) recoverArchivedPoint(ctx context.Context, currentArchivedPoint uint64) (uint64, error) {
	missingIndex := currentArchivedPoint - 1
	missingIndexSlot := missingIndex * s.slotsPerArchivedPoint
	blks, err := s.beaconDB.HighestSlotBlocksBelow(ctx, missingIndexSlot)
	if err != nil {
		return 0, err
	}
	if len(blks) != 1 {
		return 0, errUnknownBlock
	}
	missingRoot, err := ssz.HashTreeRoot(blks[0].Block)
	if err != nil {
		return 0, err
	}
	missingState, err := s.StateByRoot(ctx, missingRoot)
	if err != nil {
		return 0, err
	}

	if err := s.beaconDB.SaveState(ctx, missingState, missingRoot); err != nil {
		return 0, err
	}
	if err := s.beaconDB.SaveArchivedPointRoot(ctx, missingRoot, missingIndex); err != nil {
		return 0, err
	}
	if err := s.beaconDB.SaveLastArchivedIndex(ctx, missingIndex); err != nil {
		return 0, err
	}

	log.WithFields(logrus.Fields{
		"slot":         blks[0].Block.Slot,
		"archiveIndex": missingIndex,
		"root":         hex.EncodeToString(bytesutil.Trunc(missingRoot[:])),
	}).Info("Saved recovered archived point during state migration")

	return missingIndex, nil
}

// This returns true if the last archived point was skipped.
func skippedArchivedPoint(currentArchivedPoint uint64, lastArchivedPoint uint64) bool {
	return currentArchivedPoint-lastArchivedPoint > 1
}
