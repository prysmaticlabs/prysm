package blockchain

import (
	"context"
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain/metrics"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

const initialSyncCacheSize = 60
const minimumCacheSize = initialSyncCacheSize / 3

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
func (s *Service) onBlock(ctx context.Context, signed *ethpb.SignedBeaconBlock) (*stateTrie.BeaconState, error) {
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
	preStateValidatorCount := preState.NumValidators()

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

	if err := s.insertBlockToForkChoiceStore(ctx, b, root, postState); err != nil {
		return nil, errors.Wrapf(err, "could not insert block %d to fork choice store", b.Slot)
	}

	if err := s.beaconDB.SaveState(ctx, postState, root); err != nil {
		return nil, errors.Wrap(err, "could not save state")
	}

	// Update justified check point.
	if postState.CurrentJustifiedCheckpoint().Epoch > s.justifiedCheckpt.Epoch {
		if err := s.updateJustified(ctx, postState); err != nil {
			return nil, err
		}
	}

	// Update finalized check point. Prune the block cache and helper caches on every new finalized epoch.
	if postState.FinalizedCheckpointEpoch() > s.finalizedCheckpt.Epoch {
		if err := s.beaconDB.SaveFinalizedCheckpoint(ctx, postState.FinalizedCheckpoint()); err != nil {
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

		// Prune proto array fork choice nodes, all nodes before finalized check point will
		// be pruned.
		s.forkChoiceStore.Prune(ctx, bytesutil.ToBytes32(postState.FinalizedCheckpoint().Root))

		s.prevFinalizedCheckpt = s.finalizedCheckpt
		s.finalizedCheckpt = postState.FinalizedCheckpoint()

		if err := s.finalizedImpliesNewJustified(ctx, postState); err != nil {
			return nil, errors.Wrap(err, "could not save new justified")
		}
	}

	// Update validator indices in database as needed.
	if err := s.saveNewValidators(ctx, preStateValidatorCount, postState); err != nil {
		return nil, errors.Wrap(err, "could not save finalized checkpoint")
	}

	// Epoch boundary bookkeeping such as logging epoch summaries.
	if postState.Slot() >= s.nextEpochBoundarySlot {
		logEpochData(postState)
		metrics.ReportEpochMetrics(postState)

		// Update committees cache at epoch boundary slot.
		if err := helpers.UpdateCommitteeCache(postState, helpers.CurrentEpoch(postState)); err != nil {
			return nil, err
		}
		if err := helpers.UpdateProposerIndicesInCache(postState, helpers.CurrentEpoch(postState)); err != nil {
			return nil, err
		}

		s.nextEpochBoundarySlot = helpers.StartSlot(helpers.NextEpoch(postState))
	}

	// Delete the processed block attestations from attestation pool.
	if err := s.deletePoolAtts(b.Body.Attestations); err != nil {
		return nil, err
	}

	return postState, nil
}

// onBlockInitialSyncStateTransition is called when an initial sync block is received.
// It runs state transition on the block and without any BLS verification. The excluded BLS verification
// includes attestation's aggregated signature. It also does not save attestations.
func (s *Service) onBlockInitialSyncStateTransition(ctx context.Context, signed *ethpb.SignedBeaconBlock) error {
	ctx, span := trace.StartSpan(ctx, "blockchain.onBlock")
	defer span.End()

	if signed == nil || signed.Block == nil {
		return errors.New("nil block")
	}

	b := signed.Block

	s.initSyncStateLock.Lock()
	defer s.initSyncStateLock.Unlock()

	// Retrieve incoming block's pre state.
	preState, err := s.verifyBlkPreState(ctx, b)
	if err != nil {
		return err
	}
	preStateValidatorCount := preState.NumValidators()

	postState, err := state.ExecuteStateTransitionNoVerifyAttSigs(ctx, preState, signed)
	if err != nil {
		return errors.Wrap(err, "could not execute state transition")
	}

	if err := s.beaconDB.SaveBlock(ctx, signed); err != nil {
		return errors.Wrapf(err, "could not save block from slot %d", b.Slot)
	}
	root, err := ssz.HashTreeRoot(b)
	if err != nil {
		return errors.Wrapf(err, "could not get signing root of block %d", b.Slot)
	}

	if err := s.insertBlockToForkChoiceStore(ctx, b, root, postState); err != nil {
		return errors.Wrapf(err, "could not insert block %d to fork choice store", b.Slot)
	}

	if featureconfig.Get().InitSyncCacheState {
		s.initSyncState[root] = postState.Copy()
		s.filterBoundaryCandidates(ctx, root, postState)
	} else {
		if err := s.beaconDB.SaveState(ctx, postState, root); err != nil {
			return errors.Wrap(err, "could not save state")
		}
	}

	// Update justified check point.
	if postState.CurrentJustifiedCheckpoint().Epoch > s.justifiedCheckpt.Epoch {
		if err := s.updateJustified(ctx, postState); err != nil {
			return err
		}
	}

	// Update finalized check point. Prune the block cache and helper caches on every new finalized epoch.
	if postState.FinalizedCheckpointEpoch() > s.finalizedCheckpt.Epoch {
		startSlot := helpers.StartSlot(s.prevFinalizedCheckpt.Epoch)
		endSlot := helpers.StartSlot(s.finalizedCheckpt.Epoch)
		if endSlot > startSlot {
			if err := s.rmStatesOlderThanLastFinalized(ctx, startSlot, endSlot); err != nil {
				return errors.Wrapf(err, "could not delete states prior to finalized check point, range: %d, %d",
					startSlot, endSlot)
			}
		}

		if err := s.saveInitState(ctx, postState); err != nil {
			return errors.Wrap(err, "could not save init sync finalized state")
		}

		if err := s.beaconDB.SaveFinalizedCheckpoint(ctx, postState.FinalizedCheckpoint()); err != nil {
			return errors.Wrap(err, "could not save finalized checkpoint")
		}

		s.prevFinalizedCheckpt = s.finalizedCheckpt
		s.finalizedCheckpt = postState.FinalizedCheckpoint()

		if err := s.finalizedImpliesNewJustified(ctx, postState); err != nil {
			return errors.Wrap(err, "could not save new justified")
		}
	}

	// Update validator indices in database as needed.
	if err := s.saveNewValidators(ctx, preStateValidatorCount, postState); err != nil {
		return errors.Wrap(err, "could not save finalized checkpoint")
	}

	if featureconfig.Get().InitSyncCacheState {
		numOfStates := len(s.boundaryRoots)
		if numOfStates > initialSyncCacheSize {
			if err = s.persistCachedStates(ctx, numOfStates); err != nil {
				return err
			}
		}
	}

	// Epoch boundary bookkeeping such as logging epoch summaries.
	if postState.Slot() >= s.nextEpochBoundarySlot {
		metrics.ReportEpochMetrics(postState)
		s.nextEpochBoundarySlot = helpers.StartSlot(helpers.NextEpoch(postState))

		// Update committees cache at epoch boundary slot.
		if err := helpers.UpdateCommitteeCache(postState, helpers.CurrentEpoch(postState)); err != nil {
			return err
		}
		if err := helpers.UpdateProposerIndicesInCache(postState, helpers.CurrentEpoch(postState)); err != nil {
			return err
		}

		if featureconfig.Get().InitSyncCacheState {
			if helpers.IsEpochStart(postState.Slot()) {
				if err := s.beaconDB.SaveState(ctx, postState, root); err != nil {
					return errors.Wrap(err, "could not save state")
				}
			}
		}
	}

	return nil
}

// This feeds in the block and block's attestations to fork choice store. It's allows fork choice store
// to gain information on the most current chain.
func (s *Service) insertBlockToForkChoiceStore(ctx context.Context, blk *ethpb.BeaconBlock, root [32]byte, state *stateTrie.BeaconState) error {
	if err := s.fillInForkChoiceMissingBlocks(ctx, blk, state); err != nil {
		return err
	}

	// Feed in block to fork choice store.
	if err := s.forkChoiceStore.ProcessBlock(ctx,
		blk.Slot, root, bytesutil.ToBytes32(blk.ParentRoot),
		state.CurrentJustifiedCheckpoint().Epoch,
		state.FinalizedCheckpointEpoch()); err != nil {
		return errors.Wrap(err, "could not process block for proto array fork choice")
	}

	// Feed in block's attestations to fork choice store.
	for _, a := range blk.Body.Attestations {
		committee, err := helpers.BeaconCommitteeFromState(state, a.Data.Slot, a.Data.CommitteeIndex)
		if err != nil {
			return err
		}
		indices, err := attestationutil.AttestingIndices(a.AggregationBits, committee)
		if err != nil {
			return err
		}
		s.forkChoiceStore.ProcessAttestation(ctx, indices, bytesutil.ToBytes32(a.Data.BeaconBlockRoot), a.Data.Target.Epoch)
	}

	return nil
}

func (s *Service) persistCachedStates(ctx context.Context, numOfStates int) error {
	oldStates := make([]*stateTrie.BeaconState, 0, numOfStates)

	// Add slots to the map and add epoch boundary states to the slice.
	for _, rt := range s.boundaryRoots[:numOfStates-minimumCacheSize] {
		oldStates = append(oldStates, s.initSyncState[rt])
	}

	err := s.beaconDB.SaveStates(ctx, oldStates, s.boundaryRoots[:numOfStates-minimumCacheSize])
	if err != nil {
		return err
	}
	for _, rt := range s.boundaryRoots[:numOfStates-minimumCacheSize] {
		delete(s.initSyncState, rt)
	}
	s.boundaryRoots = s.boundaryRoots[numOfStates-minimumCacheSize:]
	return nil
}

// filter out boundary candidates from our currently processed batch of states.
func (s *Service) filterBoundaryCandidates(ctx context.Context, root [32]byte, postState *stateTrie.BeaconState) {
	stateSlice := make([][32]byte, 0, len(s.initSyncState))
	// Add slots to the map and add epoch boundary states to the slice.
	for rt := range s.initSyncState {
		stateSlice = append(stateSlice, rt)
	}

	sort.Slice(stateSlice, func(i int, j int) bool {
		return s.initSyncState[stateSlice[i]].Slot() < s.initSyncState[stateSlice[j]].Slot()
	})
	epochLength := params.BeaconConfig().SlotsPerEpoch

	// Only trigger on epoch start.
	if !helpers.IsEpochStart(postState.Slot()) {
		return
	}
	if len(s.boundaryRoots) > 0 {
		// Retrieve previous boundary root.
		previousBoundaryRoot := s.boundaryRoots[len(s.boundaryRoots)-1]
		previousSlot := s.initSyncState[previousBoundaryRoot].Slot()

		// Round up slot number to account for skipped slots.
		previousSlot = helpers.RoundUpToNearestEpoch(previousSlot)
		if postState.Slot()-previousSlot > epochLength {
			targetSlot := postState.Slot()
			tempRoots := [][32]byte{}

			// Loop through current states to filter for valid boundary states.
			for i := len(stateSlice) - 1; stateSlice[i] != previousBoundaryRoot && i >= 0; i-- {
				currentSlot := s.initSyncState[stateSlice[i]].Slot()
				// Store states from the start/end of epochs.
				if currentSlot-1 == targetSlot-epochLength ||
					currentSlot+1 == targetSlot-epochLength {
					tempRoots = append(tempRoots, stateSlice[i])
					continue
				}
				// Skip if the current slot is larger than the previous epoch
				// boundary.
				if currentSlot > targetSlot-epochLength {
					continue
				}
				tempRoots = append(tempRoots, stateSlice[i])

				// Switch target slot if the current slot is greater than
				// 1 epoch boundary from the previously saved boundary slot.
				if currentSlot > previousSlot+epochLength {
					currentSlot = helpers.RoundUpToNearestEpoch(currentSlot)
					targetSlot = currentSlot
					continue
				}
				break
			}
			// Reverse to append the roots in ascending order corresponding
			// to the respective slots.
			tempRoots = bytesutil.ReverseBytes32Slice(tempRoots)
			s.boundaryRoots = append(s.boundaryRoots, tempRoots...)
		}
	}
	s.boundaryRoots = append(s.boundaryRoots, root)
	s.pruneOldStates()
	s.pruneNonBoundaryStates()

}

// prune for states past the current finalized checkpoint.
func (s *Service) pruneOldStates() {
	prunedBoundaryRoots := [][32]byte{}
	for _, rt := range s.boundaryRoots {
		if s.initSyncState[rt].Slot() < helpers.StartSlot(s.FinalizedCheckpt().Epoch) {
			delete(s.initSyncState, rt)
			continue
		}
		prunedBoundaryRoots = append(prunedBoundaryRoots, rt)
	}
	s.boundaryRoots = prunedBoundaryRoots
}

// prune cache for non-boundary states.
func (s *Service) pruneNonBoundaryStates() {
	boundaryMap := make(map[[32]byte]bool)
	for i := range s.boundaryRoots {
		boundaryMap[s.boundaryRoots[i]] = true
	}
	for rt := range s.initSyncState {
		if !boundaryMap[rt] {
			delete(s.initSyncState, rt)
		}
	}
}
