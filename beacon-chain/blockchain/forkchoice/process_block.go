package forkchoice

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// Allow for blocks "from the future" within a certain tolerance.
const timeShiftTolerance = 10 // ms

// OnBlock is called whenever a block is received. It runs state transition on the block and
// update fork choice store struct.
//
// Spec pseudocode definition:
//   def on_block(store: Store, block: BeaconBlock) -> None:
//    # Make a copy of the state to avoid mutability issues
//    assert block.parent_root in store.block_states
//    pre_state = store.block_states[block.parent_root].copy()
//    # Blocks cannot be in the future. If they are, their consideration must be delayed until the are in the past.
//    assert store.time >= pre_state.genesis_time + block.slot * SECONDS_PER_SLOT
//    # Add new block to the store
//    store.blocks[signing_root(block)] = block
//    # Check block is a descendant of the finalized block
//    assert (
//        get_ancestor(store, signing_root(block), store.blocks[store.finalized_checkpoint.root].slot) ==
//        store.finalized_checkpoint.root
//    )
//    # Check that block is later than the finalized epoch slot
//    assert block.slot > compute_start_slot_of_epoch(store.finalized_checkpoint.epoch)
//    # Check the block is valid and compute the post-state
//    state = state_transition(pre_state, block)
//    # Add new state for this block to the store
//    store.block_states[signing_root(block)] = state
//
//    # Update justified checkpoint
//    if state.current_justified_checkpoint.epoch > store.justified_checkpoint.epoch:
//        store.justified_checkpoint = state.current_justified_checkpoint
//
//    # Update finalized checkpoint
//    if state.finalized_checkpoint.epoch > store.finalized_checkpoint.epoch:
//        store.finalized_checkpoint = state.finalized_checkpoint
func (s *Store) OnBlock(ctx context.Context, b *ethpb.BeaconBlock) error {
	ctx, span := trace.StartSpan(ctx, "forkchoice.onBlock")
	defer span.End()

	// Verify incoming block has a valid pre state.
	preState, err := s.verifyBlkPreState(ctx, b)
	if err != nil {
		return err
	}
	preStateValidatorCount := len(preState.Validators)

	// Verify block slot time is not from the feature.
	if err := verifyBlkSlotTime(preState.GenesisTime, b.Slot); err != nil {
		return err
	}

	// Verify block is a descendent of a finalized block.
	root, err := ssz.SigningRoot(b)
	if err != nil {
		return errors.Wrapf(err, "could not get signing root of block %d", b.Slot)
	}
	if err := s.verifyBlkDescendant(ctx, bytesutil.ToBytes32(b.ParentRoot), b.Slot); err != nil {
		return err
	}

	// Verify block is later than the finalized epoch slot.
	if err := s.verifyBlkFinalizedSlot(b); err != nil {
		return err
	}

	log.WithField("slot", b.Slot).Info("Executing state transition on block")

	// Apply new state transition for the block to the store.
	// Make block root as bad to reject in sync.
	postState, err := state.ExecuteStateTransition(ctx, preState, b)
	if err != nil {
		return errors.Wrap(err, "could not execute state transition")
	}

	if err := s.db.SaveBlock(ctx, b); err != nil {
		return errors.Wrapf(err, "could not save block from slot %d", b.Slot)
	}
	if err := s.db.SaveState(ctx, postState, root); err != nil {
		return errors.Wrap(err, "could not save state")
	}

	// Update justified check point.
	if postState.CurrentJustifiedCheckpoint.Epoch > s.justifiedCheckpt.Epoch {
		s.justifiedCheckpt = postState.CurrentJustifiedCheckpoint
		if err := s.db.SaveJustifiedCheckpoint(ctx, postState.CurrentJustifiedCheckpoint); err != nil {
			return errors.Wrap(err, "could not save justified checkpoint")
		}
	}
	// Update finalized check point.
	// Prune the block cache and helper caches on every new finalized epoch.
	if postState.FinalizedCheckpoint.Epoch > s.finalizedCheckpt.Epoch {
		helpers.ClearAllCaches()
		s.finalizedCheckpt = postState.FinalizedCheckpoint
		if err := s.db.SaveFinalizedCheckpoint(ctx, postState.FinalizedCheckpoint); err != nil {
			return errors.Wrap(err, "could not save finalized checkpoint")
		}
	}

	// Update validator indices in database as needed.
	if err := s.saveNewValidators(ctx, preStateValidatorCount, postState); err != nil {
		return errors.Wrap(err, "could not save finalized checkpoint")
	}

	// Epoch boundary bookkeeping such as logging epoch summaries.
	if helpers.IsEpochStart(postState.Slot) {
		logEpochData(postState)
		reportEpochMetrics(postState)
	}

	return nil
}

// verifyBlkPreState validates input block has a valid pre-state.
func (s *Store) verifyBlkPreState(ctx context.Context, b *ethpb.BeaconBlock) (*pb.BeaconState, error) {
	preState, err := s.db.State(ctx, bytesutil.ToBytes32(b.ParentRoot))
	if err != nil {
		return nil, errors.Wrapf(err, "could not get pre state for slot %d", b.Slot)
	}
	if preState == nil {
		return nil, fmt.Errorf("pre state of slot %d does not exist", b.Slot)
	}
	return preState, nil
}

// verifyBlkDescendant validates input block root is a descendant of the
// current finalized block root.
func (s *Store) verifyBlkDescendant(ctx context.Context, root [32]byte, slot uint64) error {
	finalizedBlk, err := s.db.Block(ctx, bytesutil.ToBytes32(s.finalizedCheckpt.Root))
	if err != nil || finalizedBlk == nil {
		return errors.Wrap(err, "could not get finalized block")
	}

	bFinalizedRoot, err := s.ancestor(ctx, root[:], finalizedBlk.Slot)
	if err != nil {
		return errors.Wrap(err, "could not get finalized block root")
	}
	if !bytes.Equal(bFinalizedRoot, s.finalizedCheckpt.Root) {
		return fmt.Errorf("block from slot %d is not a descendent of the current finalized block", slot)
	}
	return nil
}

// verifyBlkFinalizedSlot validates input block is not less than or equal
// to current finalized slot.
func (s *Store) verifyBlkFinalizedSlot(b *ethpb.BeaconBlock) error {
	finalizedSlot := helpers.StartSlot(s.finalizedCheckpt.Epoch)
	if finalizedSlot >= b.Slot {
		return fmt.Errorf("block is equal or earlier than finalized block, slot %d < slot %d", b.Slot, finalizedSlot)
	}
	return nil
}

// saveNewValidators saves newly added validator index from state to db. Does nothing if validator count has not
// changed.
func (s *Store) saveNewValidators(ctx context.Context, preStateValidatorCount int, postState *pb.BeaconState) error {
	postStateValidatorCount := len(postState.Validators)
	if preStateValidatorCount != postStateValidatorCount {
		for i := preStateValidatorCount; i < postStateValidatorCount; i++ {
			pubKey := postState.Validators[i].PublicKey
			if err := s.db.SaveValidatorIndex(ctx, bytesutil.ToBytes48(pubKey), uint64(i)); err != nil {
				return errors.Wrapf(err, "could not save activated validator: %d", i)
			}
			log.WithFields(logrus.Fields{
				"index":               i,
				"pubKey":              hex.EncodeToString(bytesutil.Trunc(pubKey)),
				"totalValidatorCount": i + 1,
			}).Info("New validator index saved in DB")
		}
	}
	return nil
}

// verifyBlkSlotTime validates the input block slot is not from the future.
func verifyBlkSlotTime(gensisTime uint64, blkSlot uint64) error {
	slotTime := gensisTime + blkSlot*params.BeaconConfig().SecondsPerSlot
	currentTime := uint64(roughtime.Now().Unix())
	if slotTime > currentTime+timeShiftTolerance {
		return fmt.Errorf("could not process block from the future, slot time %d > current time %d", slotTime, currentTime)
	}
	return nil
}
