package forkchoice

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain/metrics"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// OnBlock is called when a gossip block is received. It runs regular state transition on the block and
// update fork choice store.
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
func (s *Store) OnBlock(ctx context.Context, signed *ethpb.SignedBeaconBlock) (*stateTrie.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "forkchoice.onBlock")
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

	if err := s.db.SaveBlock(ctx, signed); err != nil {
		return nil, errors.Wrapf(err, "could not save block from slot %d", b.Slot)
	}
	if err := s.db.SaveState(ctx, postState, root); err != nil {
		return nil, errors.Wrap(err, "could not save state")
	}

	// Update justified check point.
	if cpt := postState.CurrentJustifiedCheckpoint(); cpt != nil && cpt.Epoch > s.justifiedCheckpt.Epoch {
		if err := s.updateJustified(ctx, postState); err != nil {
			return nil, err
		}
	}

	// Update finalized check point.
	// Prune the block cache and helper caches on every new finalized epoch.
	if cpt := postState.FinalizedCheckpoint(); cpt != nil && cpt.Epoch > s.finalizedCheckpt.Epoch {
		if err := s.db.SaveFinalizedCheckpoint(ctx, cpt); err != nil {
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
		s.finalizedCheckpt = cpt
	}

	// Update validator indices in database as needed.
	if err := s.saveNewValidators(ctx, preStateValidatorCount, postState); err != nil {
		return nil, errors.Wrap(err, "could not save finalized checkpoint")
	}
	// Save the unseen attestations from block to db.
	// Only save attestation in DB for archival node.
	if flags.Get().EnableArchive {
		if err := s.saveNewBlockAttestations(ctx, b.Body.Attestations); err != nil {
			return nil, errors.Wrap(err, "could not save attestations")
		}
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

	return postState, nil
}

// OnBlockCacheFilteredTree calls OnBlock with additional of caching of filtered block tree
// for efficient fork choice processing.
func (s *Store) OnBlockCacheFilteredTree(ctx context.Context, signed *ethpb.SignedBeaconBlock) (*stateTrie.BeaconState, error) {
	state, err := s.OnBlock(ctx, signed)
	if err != nil {
		return nil, err
	}
	if !featureconfig.Get().DisableForkChoice && featureconfig.Get().EnableBlockTreeCache && !featureconfig.Get().ProtoArrayForkChoice {
		tree, err := s.getFilterBlockTree(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not calculate filtered block tree")
		}
		s.filteredBlockTreeLock.Lock()
		s.filteredBlockTree = tree
		s.filteredBlockTreeLock.Unlock()
	}

	return state, nil
}

// OnBlockInitialSyncStateTransition is called when an initial sync block is received.
// It runs state transition on the block and without any BLS verification. The BLS verification
// includes proposer signature, randao and attestation's aggregated signature. It also does not save
// attestations.
func (s *Store) OnBlockInitialSyncStateTransition(ctx context.Context, signed *ethpb.SignedBeaconBlock) (*stateTrie.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "forkchoice.onBlock")
	defer span.End()

	if signed == nil || signed.Block == nil {
		return nil, errors.New("nil block")
	}

	b := signed.Block

	s.initSyncStateLock.Lock()
	defer s.initSyncStateLock.Unlock()

	// Retrieve incoming block's pre state.
	preState, err := s.verifyBlkPreState(ctx, b)
	if err != nil {
		return nil, err
	}
	preStateValidatorCount := preState.NumValidators()

	log.WithField("slot", b.Slot).Debug("Executing state transition on block")

	postState, err := state.ExecuteStateTransitionNoVerifyAttSigs(ctx, preState, signed)
	if err != nil {
		return nil, errors.Wrap(err, "could not execute state transition")
	}

	if err := s.db.SaveBlock(ctx, signed); err != nil {
		return nil, errors.Wrapf(err, "could not save block from slot %d", b.Slot)
	}
	root, err := ssz.HashTreeRoot(b)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get signing root of block %d", b.Slot)
	}

	if featureconfig.Get().InitSyncCacheState {
		s.initSyncState[root] = postState.Copy()
	} else {
		if err := s.db.SaveState(ctx, postState, root); err != nil {
			return nil, errors.Wrap(err, "could not save state")
		}
	}

	// Update justified check point.
	if cpt := postState.CurrentJustifiedCheckpoint(); cpt != nil && cpt.Epoch > s.justifiedCheckpt.Epoch {
		if err := s.updateJustified(ctx, postState); err != nil {
			return nil, err
		}
	}

	// Update finalized check point.
	// Prune the block cache and helper caches on every new finalized epoch.
	if cpt := postState.FinalizedCheckpoint(); cpt != nil && cpt.Epoch > s.finalizedCheckpt.Epoch {
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

		if err := s.db.SaveFinalizedCheckpoint(ctx, cpt); err != nil {
			return nil, errors.Wrap(err, "could not save finalized checkpoint")
		}

		s.prevFinalizedCheckpt = s.finalizedCheckpt
		s.finalizedCheckpt = cpt
	}

	// Update validator indices in database as needed.
	if err := s.saveNewValidators(ctx, preStateValidatorCount, postState); err != nil {
		return nil, errors.Wrap(err, "could not save finalized checkpoint")
	}

	if flags.Get().EnableArchive {
		// Save the unseen attestations from block to db.
		if err := s.saveNewBlockAttestations(ctx, b.Body.Attestations); err != nil {
			return nil, errors.Wrap(err, "could not save attestations")
		}
	}

	// Epoch boundary bookkeeping such as logging epoch summaries.
	if postState.Slot() >= s.nextEpochBoundarySlot {
		metrics.ReportEpochMetrics(postState)

		s.nextEpochBoundarySlot = helpers.StartSlot(helpers.NextEpoch(postState))
	}

	return postState, nil
}

// getBlockPreState returns the pre state of an incoming block. It uses the parent root of the block
// to retrieve the state in DB. It verifies the pre state's validity and the incoming block
// is in the correct time window.
func (s *Store) getBlockPreState(ctx context.Context, b *ethpb.BeaconBlock) (*stateTrie.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "forkchoice.getBlockPreState")
	defer span.End()

	// Verify incoming block has a valid pre state.
	preState, err := s.verifyBlkPreState(ctx, b)
	if err != nil {
		return nil, err
	}

	// Verify block slot time is not from the feature.
	if err := helpers.VerifySlotTime(preState.GenesisTime(), b.Slot); err != nil {
		return nil, err
	}

	// Verify block is a descendent of a finalized block.
	if err := s.verifyBlkDescendant(ctx, bytesutil.ToBytes32(b.ParentRoot), b.Slot); err != nil {
		return nil, err
	}

	// Verify block is later than the finalized epoch slot.
	if err := s.verifyBlkFinalizedSlot(b); err != nil {
		return nil, err
	}

	return preState, nil
}

// verifyBlkPreState validates input block has a valid pre-state.
func (s *Store) verifyBlkPreState(ctx context.Context, b *ethpb.BeaconBlock) (*stateTrie.BeaconState, error) {
	if featureconfig.Get().InitSyncCacheState {
		preState := s.initSyncState[bytesutil.ToBytes32(b.ParentRoot)]
		var err error
		if preState == nil {
			preState, err = s.db.State(ctx, bytesutil.ToBytes32(b.ParentRoot))
			if err != nil {
				return nil, errors.Wrapf(err, "could not get pre state for slot %d", b.Slot)
			}
			if preState == nil {
				return nil, fmt.Errorf("pre state of slot %d does not exist", b.Slot)
			}
			return preState, nil // No copy needed from newly hydrated DB object.
		}
		return preState.Copy(), nil
	}
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
	ctx, span := trace.StartSpan(ctx, "forkchoice.verifyBlkDescendant")
	defer span.End()

	finalizedBlkSigned, err := s.db.Block(ctx, bytesutil.ToBytes32(s.finalizedCheckpt.Root))
	if err != nil || finalizedBlkSigned == nil || finalizedBlkSigned.Block == nil {
		return errors.Wrap(err, "could not get finalized block")
	}
	finalizedBlk := finalizedBlkSigned.Block

	bFinalizedRoot, err := s.ancestor(ctx, root[:], finalizedBlk.Slot)
	if err != nil {
		return errors.Wrap(err, "could not get finalized block root")
	}
	if !bytes.Equal(bFinalizedRoot, s.finalizedCheckpt.Root) {
		err := fmt.Errorf(
			"block from slot %d is not a descendent of the current finalized block slot %d, %#x != %#x",
			slot,
			finalizedBlk.Slot,
			bytesutil.Trunc(bFinalizedRoot),
			bytesutil.Trunc(s.finalizedCheckpt.Root),
		)
		traceutil.AnnotateError(span, err)
		return err
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

// saveNewValidators saves newly added validator indices from the state to db.
// Does nothing if validator count has not changed.
func (s *Store) saveNewValidators(ctx context.Context, preStateValidatorCount int, postState *stateTrie.BeaconState) error {
	postStateValidatorCount := postState.NumValidators()
	if preStateValidatorCount != postStateValidatorCount {
		indices := make([]uint64, 0)
		pubKeys := make([][48]byte, 0)
		for i := preStateValidatorCount; i < postStateValidatorCount; i++ {
			indices = append(indices, uint64(i))
			pubKeys = append(pubKeys, postState.PubkeyAtIndex(uint64(i)))
		}
		if err := s.db.SaveValidatorIndices(ctx, pubKeys, indices); err != nil {
			return errors.Wrapf(err, "could not save activated validators: %v", indices)
		}
		log.WithFields(logrus.Fields{
			"indices":             indices,
			"totalValidatorCount": postStateValidatorCount - preStateValidatorCount,
		}).Info("Validator indices saved in DB")
	}
	return nil
}

// saveNewBlockAttestations saves the new attestations in block to DB.
func (s *Store) saveNewBlockAttestations(ctx context.Context, atts []*ethpb.Attestation) error {
	attestations := make([]*ethpb.Attestation, 0, len(atts))
	for _, att := range atts {
		aggregated, err := s.aggregatedAttestations(ctx, att)
		if err != nil {
			continue
		}
		attestations = append(attestations, aggregated...)
	}
	if err := s.db.SaveAttestations(ctx, atts); err != nil {
		return err
	}
	return nil
}

// rmStatesOlderThanLastFinalized deletes the states in db since last finalized check point.
func (s *Store) rmStatesOlderThanLastFinalized(ctx context.Context, startSlot uint64, endSlot uint64) error {
	ctx, span := trace.StartSpan(ctx, "forkchoice.rmStatesBySlots")
	defer span.End()

	// Make sure start slot is not a skipped slot
	for i := startSlot; i > 0; i-- {
		filter := filters.NewFilter().SetStartSlot(i).SetEndSlot(i)
		b, err := s.db.Blocks(ctx, filter)
		if err != nil {
			return err
		}
		if len(b) > 0 {
			startSlot = i
			break
		}
	}

	// Make sure finalized slot is not a skipped slot.
	for i := endSlot; i > 0; i-- {
		filter := filters.NewFilter().SetStartSlot(i).SetEndSlot(i)
		b, err := s.db.Blocks(ctx, filter)
		if err != nil {
			return err
		}
		if len(b) > 0 {
			endSlot = i - 1
			break
		}
	}

	// Do not remove genesis state
	if startSlot == 0 {
		startSlot++
	}
	// If end slot comes less than start slot
	if endSlot < startSlot {
		endSlot = startSlot
	}

	filter := filters.NewFilter().SetStartSlot(startSlot).SetEndSlot(endSlot)
	roots, err := s.db.BlockRoots(ctx, filter)
	if err != nil {
		return err
	}

	roots, err = s.filterBlockRoots(ctx, roots)
	if err != nil {
		return err
	}

	if err := s.db.DeleteStates(ctx, roots); err != nil {
		return err
	}

	return nil
}

// shouldUpdateCurrentJustified prevents bouncing attack, by only update conflicting justified
// checkpoints in the fork choice if in the early slots of the epoch.
// Otherwise, delay incorporation of new justified checkpoint until next epoch boundary.
// See https://ethresear.ch/t/prevention-of-bouncing-attack-on-ffg/6114 for more detailed analysis and discussion.
func (s *Store) shouldUpdateCurrentJustified(ctx context.Context, newJustifiedCheckpt *ethpb.Checkpoint) (bool, error) {
	if helpers.SlotsSinceEpochStarts(s.currentSlot()) < params.BeaconConfig().SafeSlotsToUpdateJustified {
		return true, nil
	}
	newJustifiedBlockSigned, err := s.db.Block(ctx, bytesutil.ToBytes32(newJustifiedCheckpt.Root))
	if err != nil {
		return false, err
	}
	if newJustifiedBlockSigned == nil || newJustifiedBlockSigned.Block == nil {
		return false, errors.New("nil new justified block")
	}
	newJustifiedBlock := newJustifiedBlockSigned.Block
	if newJustifiedBlock.Slot <= helpers.StartSlot(s.justifiedCheckpt.Epoch) {
		return false, nil
	}
	justifiedBlockSigned, err := s.db.Block(ctx, bytesutil.ToBytes32(s.justifiedCheckpt.Root))
	if err != nil {
		return false, err
	}
	if justifiedBlockSigned == nil || justifiedBlockSigned.Block == nil {
		return false, errors.New("nil justified block")
	}
	justifiedBlock := justifiedBlockSigned.Block
	b, err := s.ancestor(ctx, newJustifiedCheckpt.Root, justifiedBlock.Slot)
	if err != nil {
		return false, err
	}
	if !bytes.Equal(b, s.justifiedCheckpt.Root) {
		return false, nil
	}
	return true, nil
}

func (s *Store) updateJustified(ctx context.Context, state *stateTrie.BeaconState) error {
	if cpt := state.CurrentJustifiedCheckpoint(); cpt != nil && cpt.Epoch > s.bestJustifiedCheckpt.Epoch {
		s.bestJustifiedCheckpt = cpt
	}
	canUpdate, err := s.shouldUpdateCurrentJustified(ctx, state.CurrentJustifiedCheckpoint())
	if err != nil {
		return err
	}
	if canUpdate {
		s.justifiedCheckpt = state.CurrentJustifiedCheckpoint()
	}
	cpt := state.CurrentJustifiedCheckpoint()
	if cpt != nil && featureconfig.Get().InitSyncCacheState {
		justifiedRoot := bytesutil.ToBytes32(cpt.Root)
		justifiedState := s.initSyncState[justifiedRoot]
		// If justified state is nil, resume back to normal syncing process and save
		// justified check point.
		if justifiedState == nil {
			return s.db.SaveJustifiedCheckpoint(ctx, cpt)
		}
		if err := s.db.SaveState(ctx, justifiedState, justifiedRoot); err != nil {
			return errors.Wrap(err, "could not save justified state")
		}
		for r, st := range s.initSyncState {
			if st != nil && st.HasInnerState() && state.Slot() > st.Slot()+8 {
				delete(s.initSyncState, r)
			}
		}

	}

	return s.db.SaveJustifiedCheckpoint(ctx, cpt)
}

// currentSlot returns the current slot based on time.
func (s *Store) currentSlot() uint64 {
	return (uint64(time.Now().Unix()) - s.genesisTime) / params.BeaconConfig().SecondsPerSlot
}

// updates justified check point in store if a better check point is known
func (s *Store) updateJustifiedCheckpoint() {
	// Update at epoch boundary slot only
	if !helpers.IsEpochStart(s.currentSlot()) {
		return
	}
	if s.bestJustifiedCheckpt.Epoch > s.justifiedCheckpt.Epoch {
		s.justifiedCheckpt = s.bestJustifiedCheckpt
	}
}

// This saves every finalized state in DB during initial sync, needed as part of optimization to
// use cache state during initial sync in case of restart.
func (s *Store) saveInitState(ctx context.Context, state *stateTrie.BeaconState) error {
	cpt := state.FinalizedCheckpoint()
	if !featureconfig.Get().InitSyncCacheState || cpt == nil {
		return nil
	}
	var err error
	finalizedRoot := bytesutil.ToBytes32(cpt.Root)
	fs := s.initSyncState[finalizedRoot]
	if fs == nil {
		fs, err = s.db.State(ctx, finalizedRoot)
		if err != nil {
			return err
		}
		if fs == nil {
			return errors.Errorf("state with root %#x doesnt exist", finalizedRoot)
		}
	} else {
		if err := s.db.SaveState(ctx, fs, finalizedRoot); err != nil {
			return errors.Wrap(err, "could not save state")
		}
	}
	for r, oldState := range s.initSyncState {
		if oldState.Slot() < cpt.Epoch*params.BeaconConfig().SlotsPerEpoch {
			delete(s.initSyncState, r)
		}
	}
	return nil
}

// This filters block roots that are not known as head root and finalized root in DB.
// It serves as the last line of defence before we prune states.
func (s *Store) filterBlockRoots(ctx context.Context, roots [][32]byte) ([][32]byte, error) {
	f, err := s.db.FinalizedCheckpoint(ctx)
	if err != nil {
		return nil, err
	}
	fRoot := f.Root
	h, err := s.db.HeadBlock(ctx)
	if err != nil {
		return nil, err
	}
	hRoot, err := ssz.SigningRoot(h)
	if err != nil {
		return nil, err
	}

	filtered := make([][32]byte, 0, len(roots))
	for _, root := range roots {
		if bytes.Equal(root[:], fRoot[:]) || bytes.Equal(root[:], hRoot[:]) {
			continue
		}
		filtered = append(filtered, root)
	}

	return filtered, nil
}
