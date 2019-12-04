package forkchoice

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
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
//        store.justified_checkpoint = state.current_justified_checkpoint
//
//    # Update finalized checkpoint
//    if state.finalized_checkpoint.epoch > store.finalized_checkpoint.epoch:
//        store.finalized_checkpoint = state.finalized_checkpoint
func (s *Store) OnBlock(ctx context.Context, b *ethpb.BeaconBlock) error {
	ctx, span := trace.StartSpan(ctx, "forkchoice.onBlock")
	defer span.End()

	// Retrieve incoming block's pre state.
	preState, err := s.getBlockPreState(ctx, b)
	if err != nil {
		return err
	}
	preStateValidatorCount := len(preState.Validators)

	root, err := ssz.SigningRoot(b)
	if err != nil {
		return errors.Wrapf(err, "could not get signing root of block %d", b.Slot)
	}
	log.WithFields(logrus.Fields{
		"slot": b.Slot,
		"root": fmt.Sprintf("0x%s...", hex.EncodeToString(root[:])[:8]),
	}).Info("Executing state transition on block")
	postState, err := state.ExecuteStateTransition(ctx, preState, b)
	if err != nil {
		return errors.Wrap(err, "could not execute state transition")
	}
	if err := s.updateBlockAttestationsVotes(ctx, b.Body.Attestations); err != nil {
		return errors.Wrap(err, "could not update votes for attestations in block")
	}

	if err := s.db.SaveBlock(ctx, b); err != nil {
		return errors.Wrapf(err, "could not save block from slot %d", b.Slot)
	}
	if err := s.db.SaveState(ctx, postState, root); err != nil {
		return errors.Wrap(err, "could not save state")
	}

	// Update justified check point.
	if postState.CurrentJustifiedCheckpoint.Epoch > s.JustifiedCheckpt().Epoch {
		s.justifiedCheckpt = postState.CurrentJustifiedCheckpoint
		if err := s.db.SaveJustifiedCheckpoint(ctx, postState.CurrentJustifiedCheckpoint); err != nil {
			return errors.Wrap(err, "could not save justified checkpoint")
		}
	}

	// Update finalized check point.
	// Prune the block cache and helper caches on every new finalized epoch.
	if postState.FinalizedCheckpoint.Epoch > s.finalizedCheckpt.Epoch {
		s.clearSeenAtts()
		helpers.ClearAllCaches()
		if err := s.db.SaveFinalizedCheckpoint(ctx, postState.FinalizedCheckpoint); err != nil {
			return errors.Wrap(err, "could not save finalized checkpoint")
		}

		startSlot := helpers.StartSlot(s.prevFinalizedCheckpt.Epoch) + 1
		if featureconfig.Get().PruneEpochBoundaryStates {
			startSlot = helpers.StartSlot(s.prevFinalizedCheckpt.Epoch)
		}
		endSlot := helpers.StartSlot(s.finalizedCheckpt.Epoch)
		if endSlot > startSlot {
			if err := s.rmStatesOlderThanLastFinalized(ctx, startSlot, endSlot); err != nil {
				return errors.Wrapf(err, "could not delete states prior to finalized check point, range: %d, %d",
					startSlot, endSlot)
			}
		}

		s.prevFinalizedCheckpt = s.finalizedCheckpt
		s.finalizedCheckpt = postState.FinalizedCheckpoint
	}

	// Update validator indices in database as needed.
	if err := s.saveNewValidators(ctx, preStateValidatorCount, postState); err != nil {
		return errors.Wrap(err, "could not save finalized checkpoint")
	}
	// Save the unseen attestations from block to db.
	if err := s.saveNewBlockAttestations(ctx, b.Body.Attestations); err != nil {
		return errors.Wrap(err, "could not save attestations")
	}

	// Epoch boundary bookkeeping such as logging epoch summaries.
	if helpers.IsEpochStart(postState.Slot) {
		logEpochData(postState)
		reportEpochMetrics(postState)

		// Update committee shuffled indices at the end of every epoch
		if featureconfig.Get().EnableNewCache {
			if err := helpers.UpdateCommitteeCache(postState); err != nil {
				return err
			}
		}
	}

	return nil
}

// OnBlockInitialSyncStateTransition is called when an initial sync block is received.
// It runs state transition on the block and without any BLS verification. The BLS verification
// includes proposer signature, randao and attestation's aggregated signature. It also does not save
// attestations.
func (s *Store) OnBlockInitialSyncStateTransition(ctx context.Context, b *ethpb.BeaconBlock) error {
	ctx, span := trace.StartSpan(ctx, "forkchoice.onBlock")
	defer span.End()

	// Retrieve incoming block's pre state.
	preState, err := s.getBlockPreState(ctx, b)
	if err != nil {
		return err
	}
	preStateValidatorCount := len(preState.Validators)

	log.WithField("slot", b.Slot).Debug("Executing state transition on block")

	postState, err := state.ExecuteStateTransitionNoVerify(ctx, preState, b)
	if err != nil {
		return errors.Wrap(err, "could not execute state transition")
	}

	if err := s.db.SaveBlock(ctx, b); err != nil {
		return errors.Wrapf(err, "could not save block from slot %d", b.Slot)
	}
	root, err := ssz.SigningRoot(b)
	if err != nil {
		return errors.Wrapf(err, "could not get signing root of block %d", b.Slot)
	}
	if err := s.db.SaveState(ctx, postState, root); err != nil {
		return errors.Wrap(err, "could not save state")
	}

	// Update justified check point.
	if postState.CurrentJustifiedCheckpoint.Epoch > s.JustifiedCheckpt().Epoch {
		s.justifiedCheckpt = postState.CurrentJustifiedCheckpoint
		if err := s.db.SaveJustifiedCheckpoint(ctx, postState.CurrentJustifiedCheckpoint); err != nil {
			return errors.Wrap(err, "could not save justified checkpoint")
		}
	}

	// Update finalized check point.
	// Prune the block cache and helper caches on every new finalized epoch.
	if postState.FinalizedCheckpoint.Epoch > s.finalizedCheckpt.Epoch {
		s.clearSeenAtts()
		helpers.ClearAllCaches()

		startSlot := helpers.StartSlot(s.prevFinalizedCheckpt.Epoch) + 1
		if featureconfig.Get().PruneEpochBoundaryStates {
			startSlot = helpers.StartSlot(s.prevFinalizedCheckpt.Epoch)
		}
		endSlot := helpers.StartSlot(s.finalizedCheckpt.Epoch)
		if endSlot > startSlot {
			if err := s.rmStatesOlderThanLastFinalized(ctx, startSlot, endSlot); err != nil {
				return errors.Wrapf(err, "could not delete states prior to finalized check point, range: %d, %d",
					startSlot, endSlot)
			}
		}

		if err := s.db.SaveFinalizedCheckpoint(ctx, postState.FinalizedCheckpoint); err != nil {
			return errors.Wrap(err, "could not save finalized checkpoint")
		}

		s.prevFinalizedCheckpt = s.finalizedCheckpt
		s.finalizedCheckpt = postState.FinalizedCheckpoint
	}

	// Update validator indices in database as needed.
	if err := s.saveNewValidators(ctx, preStateValidatorCount, postState); err != nil {
		return errors.Wrap(err, "could not save finalized checkpoint")
	}

	if flags.Get().EnableArchive {
		// Save the unseen attestations from block to db.
		if err := s.saveNewBlockAttestations(ctx, b.Body.Attestations); err != nil {
			return errors.Wrap(err, "could not save attestations")
		}
	}

	// Epoch boundary bookkeeping such as logging epoch summaries.
	if helpers.IsEpochStart(postState.Slot) {
		reportEpochMetrics(postState)

		// Update committee shuffled indices at the end of every epoch
		if featureconfig.Get().EnableNewCache {
			if err := helpers.UpdateCommitteeCache(postState); err != nil {
				return err
			}
		}
	}

	return nil
}

// getBlockPreState returns the pre state of an incoming block. It uses the parent root of the block
// to retrieve the state in DB. It verifies the pre state's validity and the incoming block
// is in the correct time window.
func (s *Store) getBlockPreState(ctx context.Context, b *ethpb.BeaconBlock) (*pb.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "forkchoice.getBlockPreState")
	defer span.End()

	// Verify incoming block has a valid pre state.
	preState, err := s.verifyBlkPreState(ctx, b)
	if err != nil {
		return nil, err
	}

	// Verify block slot time is not from the feature.
	if err := helpers.VerifySlotTime(preState.GenesisTime, b.Slot); err != nil {
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

// updateBlockAttestationsVotes checks the attestations in block and filter out the seen ones,
// the unseen ones get passed to updateBlockAttestationVote for updating fork choice votes.
func (s *Store) updateBlockAttestationsVotes(ctx context.Context, atts []*ethpb.Attestation) error {
	s.seenAttsLock.Lock()
	defer s.seenAttsLock.Unlock()

	for _, att := range atts {
		// If we have not seen the attestation yet
		r, err := hashutil.HashProto(att)
		if err != nil {
			return err
		}
		if s.seenAtts[r] {
			continue
		}
		if err := s.updateBlockAttestationVote(ctx, att); err != nil {
			log.WithError(err).Warn("Attestation failed to update vote")
		}
		s.seenAtts[r] = true
	}
	return nil
}

// updateBlockAttestationVotes checks the attestation to update validator's latest votes.
func (s *Store) updateBlockAttestationVote(ctx context.Context, att *ethpb.Attestation) error {
	tgt := att.Data.Target
	baseState, err := s.db.State(ctx, bytesutil.ToBytes32(tgt.Root))
	if err != nil {
		return errors.Wrap(err, "could not get state for attestation tgt root")
	}
	if baseState == nil {
		return errors.New("no state found in db with attestation tgt root")
	}
	indexedAtt, err := blocks.ConvertToIndexed(ctx, baseState, att)
	if err != nil {
		return errors.Wrap(err, "could not convert attestation to indexed attestation")
	}
	s.voteLock.Lock()
	defer s.voteLock.Unlock()
	for _, i := range append(indexedAtt.CustodyBit_0Indices, indexedAtt.CustodyBit_1Indices...) {
		vote, ok := s.latestVoteMap[i]
		if !ok || tgt.Epoch > vote.Epoch {
			s.latestVoteMap[i] = &pb.ValidatorLatestVote{
				Epoch: tgt.Epoch,
				Root:  tgt.Root,
			}
		}
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
	ctx, span := trace.StartSpan(ctx, "forkchoice.verifyBlkDescendant")
	defer span.End()

	finalizedBlk, err := s.db.Block(ctx, bytesutil.ToBytes32(s.finalizedCheckpt.Root))
	if err != nil || finalizedBlk == nil {
		return errors.Wrap(err, "could not get finalized block")
	}

	bFinalizedRoot, err := s.ancestor(ctx, root[:], finalizedBlk.Slot)
	if err != nil {
		return errors.Wrap(err, "could not get finalized block root")
	}
	if !bytes.Equal(bFinalizedRoot, s.finalizedCheckpt.Root) {
		err := fmt.Errorf("block from slot %d is not a descendent of the current finalized block slot %d, %#x != %#x",
			slot, finalizedBlk.Slot, bytesutil.Trunc(bFinalizedRoot), bytesutil.Trunc(s.finalizedCheckpt.Root))
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

// clearSeenAtts clears seen attestations map, it gets called upon new finalization.
func (s *Store) clearSeenAtts() {
	s.seenAttsLock.Lock()
	s.seenAttsLock.Unlock()
	s.seenAtts = make(map[[32]byte]bool)
}

// rmStatesOlderThanLastFinalized deletes the states in db since last finalized check point.
func (s *Store) rmStatesOlderThanLastFinalized(ctx context.Context, startSlot uint64, endSlot uint64) error {
	ctx, span := trace.StartSpan(ctx, "forkchoice.rmStatesBySlots")
	defer span.End()

	// Make sure start slot is not a skipped slot
	if featureconfig.Get().PruneEpochBoundaryStates {
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

	if err := s.db.DeleteStates(ctx, roots); err != nil {
		return err
	}

	return nil
}
