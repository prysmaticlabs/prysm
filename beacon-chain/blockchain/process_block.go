package blockchain

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// onBlock is called when a gossip block is received. It runs regular state transition on the block.
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
//        if state.current_justified_checkpoint.epoch > store.best_justified_checkpoint.epoch:
//            store.best_justified_checkpoint = state.current_justified_checkpoint
//
//    # Update finalized checkpoint
//    if state.finalized_checkpoint.epoch > store.finalized_checkpoint.epoch:
//        store.finalized_checkpoint = state.finalized_checkpoint
func (s *Service) onBlock(ctx context.Context, signed *ethpb.SignedBeaconBlock) (*pb.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "blockchain.onBlock")
	defer span.End()

	if signed == nil || signed.Block == nil {
		return nil, errors.New("nil block")
	}

	b := signed.Block

	// Retrieve incoming block's pre state.
	preState, err := s.getBlockPreState(ctx, b)
	if err != nil {
		return nil, err
	}
	preStateValidatorCount := len(preState.Validators)

	root, err := ssz.HashTreeRoot(b)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get signing root of block %d", b.Slot)
	}
	log.WithFields(logrus.Fields{
		"slot": b.Slot,
		"root": fmt.Sprintf("0x%s...", hex.EncodeToString(root[:])[:8]),
	}).Info("Executing state transition on block")

	postState, err := state.ExecuteStateTransition(ctx, preState, signed)
	if err != nil {
		return nil, errors.Wrap(err, "could not execute state transition")
	}

	if err := s.beaconDB.SaveBlock(ctx, signed); err != nil {
		return nil, errors.Wrapf(err, "could not save block from slot %d", b.Slot)
	}
	if err := s.beaconDB.SaveState(ctx, postState, root); err != nil {
		return nil, errors.Wrap(err, "could not save state")
	}

	// Update justified check point.
	if postState.CurrentJustifiedCheckpoint.Epoch > s.justifiedCheckpt.Epoch {
		if err := s.updateJustified(ctx, postState); err != nil {
			return nil, err
		}
	}

	// Update finalized check point. Prune the block cache and helper caches on every new finalized epoch.
	if postState.FinalizedCheckpoint.Epoch > s.finalizedCheckpt.Epoch {
		if err := s.beaconDB.SaveFinalizedCheckpoint(ctx, postState.FinalizedCheckpoint); err != nil {
			return nil, errors.Wrap(err, "could not save finalized checkpoint")
		}

		startSlot := helpers.StartSlot(s.prevFinalizedCheckpt.Epoch)
		endSlot := helpers.StartSlot(s.finalizedCheckpt.Epoch)
		if endSlot > startSlot {
			if err := s.rmStatesOlderThanLastFinalized(ctx, startSlot, endSlot); err != nil {
				return nil, errors.Wrapf(err, "could not delete states prior to finalized check point, range: %d, %d",
					startSlot, endSlot)
			}
		}

		s.prevFinalizedCheckpt = s.finalizedCheckpt
		s.finalizedCheckpt = postState.FinalizedCheckpoint
	}

	// Update validator indices in database as needed.
	if err := s.saveNewValidators(ctx, preStateValidatorCount, postState); err != nil {
		return nil, errors.Wrap(err, "could not save finalized checkpoint")
	}

	// Epoch boundary bookkeeping such as logging epoch summaries.
	if postState.Slot >= s.nextEpochBoundarySlot {
		logEpochData(postState)
		reportEpochMetrics(postState)

		// Update committees cache at epoch boundary slot.
		if err := helpers.UpdateCommitteeCache(postState, helpers.CurrentEpoch(postState)); err != nil {
			return nil, err
		}
		if err := helpers.UpdateProposerIndicesInCache(postState, helpers.CurrentEpoch(postState)); err != nil {
			return nil, err
		}

		s.nextEpochBoundarySlot = helpers.StartSlot(helpers.NextEpoch(postState))
	}

	return postState, nil
}

// onBlockInitialSyncStateTransition is called when an initial sync block is received.
// It runs state transition on the block and without any BLS verification. The excluded BLS verification
// includes attestation's aggregated signature. It also does not save attestations.
func (s *Service) onBlockInitialSyncStateTransition(ctx context.Context, signed *ethpb.SignedBeaconBlock) (*pb.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "blockchain.onBlock")
	defer span.End()

	if signed == nil || signed.Block == nil {
		return nil, errors.New("nil block")
	}

	b := signed.Block

	s.initSyncStateLock.Lock()
	defer s.initSyncStateLock.Unlock()

	// Retrieve incoming block's pre state.
	preState, err := s.cachedPreState(ctx, b)
	if err != nil {
		return nil, err
	}
	preStateValidatorCount := len(preState.Validators)

	postState, err := state.ExecuteStateTransitionNoVerifyAttSigs(ctx, preState, signed)
	if err != nil {
		return nil, errors.Wrap(err, "could not execute state transition")
	}

	if err := s.beaconDB.SaveBlock(ctx, signed); err != nil {
		return nil, errors.Wrapf(err, "could not save block from slot %d", b.Slot)
	}
	root, err := ssz.HashTreeRoot(b)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get signing root of block %d", b.Slot)
	}

	if featureconfig.Get().InitSyncCacheState {
		s.initSyncState[root] = postState
	} else {
		if err := s.beaconDB.SaveState(ctx, postState, root); err != nil {
			return nil, errors.Wrap(err, "could not save state")
		}
	}

	// Update justified check point.
	if postState.CurrentJustifiedCheckpoint.Epoch > s.justifiedCheckpt.Epoch {
		if err := s.updateJustified(ctx, postState); err != nil {
			return nil, err
		}
	}

	// Update finalized check point. Prune the block cache and helper caches on every new finalized epoch.
	if postState.FinalizedCheckpoint.Epoch > s.finalizedCheckpt.Epoch {
		startSlot := helpers.StartSlot(s.prevFinalizedCheckpt.Epoch)
		endSlot := helpers.StartSlot(s.finalizedCheckpt.Epoch)
		if endSlot > startSlot {
			if err := s.rmStatesOlderThanLastFinalized(ctx, startSlot, endSlot); err != nil {
				return nil, errors.Wrapf(err, "could not delete states prior to finalized check point, range: %d, %d",
					startSlot, endSlot)
			}
		}

		if err := s.saveInitState(ctx, postState); err != nil {
			return nil, errors.Wrap(err, "could not save init sync finalized state")
		}

		if err := s.beaconDB.SaveFinalizedCheckpoint(ctx, postState.FinalizedCheckpoint); err != nil {
			return nil, errors.Wrap(err, "could not save finalized checkpoint")
		}

		s.prevFinalizedCheckpt = s.finalizedCheckpt
		s.finalizedCheckpt = postState.FinalizedCheckpoint
	}

	// Update validator indices in database as needed.
	if err := s.saveNewValidators(ctx, preStateValidatorCount, postState); err != nil {
		return nil, errors.Wrap(err, "could not save finalized checkpoint")
	}

	// Epoch boundary bookkeeping such as logging epoch summaries.
	if postState.Slot >= s.nextEpochBoundarySlot {
		reportEpochMetrics(postState)
		s.nextEpochBoundarySlot = helpers.StartSlot(helpers.NextEpoch(postState))

		// Update committees cache at epoch boundary slot.
		if err := helpers.UpdateCommitteeCache(postState, helpers.CurrentEpoch(postState)); err != nil {
			return nil, err
		}
		if err := helpers.UpdateProposerIndicesInCache(postState, helpers.CurrentEpoch(postState)); err != nil {
			return nil, err
		}
	}

	return postState, nil
}
