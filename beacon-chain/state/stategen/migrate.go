package stategen

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/pkg/errors"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// MigrateToCold advances the finalized info in between the cold and hot state sections.
// It moves the recent finalized states from the hot section to the cold section and
// only preserves the ones that are on archived point.
func (s *State) MigrateToCold(ctx context.Context, fRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "stateGen.MigrateToCold")
	defer span.End()

	s.finalizedInfo.lock.RLock()
	oldFSlot := s.finalizedInfo.slot
	s.finalizedInfo.lock.RUnlock()

	fBlock, err := s.beaconDB.Block(ctx, fRoot)
	if err != nil {
		return err
	}
	fSlot := fBlock.Block().Slot()
	if oldFSlot > fSlot {
		return nil
	}

	// Start at previous finalized slot, stop at current finalized slot (it will be handled in the next migration).
	// If the slot is on archived point, save the state of that slot to the DB.
	for slot := oldFSlot; slot < fSlot; slot++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if slot%s.slotsPerArchivedPoint == 0 && slot != 0 {
			var aRoot [32]byte
			var aState state.BeaconState

			// cases we need to handle:
			// 1. state exists in the epoch boundary state cache
			// 2. state is in the database due to hot state saver in snapshot mode
			// in this case we're looking up by slot, there's no root to look up, so we still have to replay blocks
			// so state 2&3 are the same case
			// 3. state snapshot is not in the database, rebuild it from stategen
			// in all 3 cases we want to make sure the snapshot mode saver does not delete it.
			cached, exists, err := s.epochBoundaryStateCache.getBySlot(slot)
			if err != nil {
				return fmt.Errorf("could not get epoch boundary state for slot %d", slot)
			}
			if exists {
				// case 1 - state in epoch boundary state cache
				aRoot = cached.root
				aState = cached.state
			} else {
				// case 3 - state is not in db, we need to rebuild it from the most recent available state
				aState, err = s.rb.ReplayerForSlot(slot).ReplayToSlot(ctx, slot)
				if err != nil {
					return err
				}
				// compute the block hash from the state
				sr, err := aState.HashTreeRoot(ctx)
				if err != nil {
					return errors.Wrap(err, "error while computing hash_tree_root of state in MigrateToCold")
				}
				header := aState.LatestBlockHeader()
				header.StateRoot = sr[:]
				aRoot, err = header.HashTreeRoot()
				if err != nil {
					return errors.Wrap(err, "error while computing block root using state data")
				}
			}

			if err := s.saver.Preserve(ctx, aRoot, aState); err != nil {
				return err
			}

			log.WithFields(
				logrus.Fields{
					"slot": aState.Slot(),
					"root": hex.EncodeToString(bytesutil.Trunc(aRoot[:])),
				}).Info("Saved state in DB")
		}
	}

	// Update finalized info in memory.
	fInfo, ok, err := s.epochBoundaryStateCache.getByBlockRoot(fRoot)
	if err != nil {
		return err
	}
	if ok {
		s.SaveFinalizedState(fSlot, fRoot, fInfo.state)
	}

	return nil
}
