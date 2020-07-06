package stategen

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/prysmaticlabs/go-ssz"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// MigrateToCold advances the finalized info in between the cold and hot state sections.
// It moves the recent finalized states from the hot section to the cold section and
// only preserve the ones that's on archived point.
func (s *State) MigrateToCold(ctx context.Context, fSlot uint64, fRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "stateGen.MigrateToCold")
	defer span.End()

	s.finalizedInfo.lock.RLock()
	oldFSlot := s.finalizedInfo.slot
	s.finalizedInfo.lock.RUnlock()

	if oldFSlot > fSlot {
		return nil
	}

	// Start at previous finalized slot, stop at current finalized slot.
	// If the slot is on archived point, save the state of that slot to the DB.
	for i := oldFSlot; i < fSlot; i++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if i%s.slotsPerArchivedPoint == 0 && i != 0 {
			cached, exists, err := s.epochBoundaryStateCache.getBySlot(i)
			if err != nil {
				return fmt.Errorf("could not get epoch boundary state for slot %d", i)
			}

			aIndex := i / s.slotsPerArchivedPoint
			var aRoot [32]byte
			var aState *stateTrie.BeaconState

			// When the epoch boundary state is not in cache due to skip slot scenario,
			// we have to regenerate the state which will represent epoch boundary.
			// By finding the highest available block below epoch boundary slot, we
			// generate the state for that block root.
			if exists {
				aRoot = cached.root
				aState = cached.state
			} else {
				blks, err := s.beaconDB.HighestSlotBlocksBelow(ctx, i)
				if err != nil {
					return err
				}
				// Given the block has been finalized, the db should not have more than one block in a given slot.
				// We should error out when this happens.
				if len(blks) != 1 {
					return errUnknownBlock
				}
				missingRoot, err := ssz.HashTreeRoot(blks[0].Block)
				if err != nil {
					return err
				}
				missingState, err := s.StateByRoot(ctx, missingRoot)
				if err != nil {
					return err
				}
				aRoot = missingRoot
				aState = missingState
			}
			if s.beaconDB.HasState(ctx, aRoot) {
				continue
			}

			if err := s.beaconDB.SaveState(ctx, aState, aRoot); err != nil {
				return err
			}
			if err := s.beaconDB.SaveArchivedPointRoot(ctx, aRoot, aIndex); err != nil {
				return err
			}
			if err := s.beaconDB.SaveLastArchivedIndex(ctx, aIndex); err != nil {
				return err
			}
			log.WithFields(
				logrus.Fields{
					"slot":          aState.Slot(),
					"archivedIndex": aIndex,
					"root":          hex.EncodeToString(bytesutil.Trunc(aRoot[:])),
				}).Info("Saved state in DB")
		}
	}

	// Migrate all state summary objects from state summary cache to DB.
	if err := s.beaconDB.SaveStateSummaries(ctx, s.stateSummaryCache.GetAll()); err != nil {
		return err
	}
	s.stateSummaryCache.Clear()

	// Update finalized info in memory.
	fInfo, ok, err := s.epochBoundaryStateCache.getByRoot(fRoot)
	if err != nil {
		return err
	}
	if ok {
		s.SaveFinalizedState(fSlot, fRoot, fInfo.state)
	}

	return nil
}
