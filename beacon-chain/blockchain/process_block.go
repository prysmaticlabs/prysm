package blockchain

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	coreTime "github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/config/features"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/monitoring/tracing"
	ethpbv1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/attestation"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/time/slots"
	"go.opencensus.io/trace"
)

// A custom slot deadline for processing state slots in our cache.
const slotDeadline = 5 * time.Second

// A custom deadline for deposit trie insertion.
const depositDeadline = 20 * time.Second

// This defines size of the upper bound for initial sync block cache.
var initialSyncBlockCacheSize = uint64(2 * params.BeaconConfig().SlotsPerEpoch)

// onBlock is called when a gossip block is received. It runs regular state transition on the block.
// The block's signing root should be computed before calling this method to avoid redundant
// computation in this method and methods it calls into.
//
// Spec pseudocode definition:
//   def on_block(store: Store, signed_block: SignedBeaconBlock) -> None:
//    block = signed_block.message
//    # Parent block must be known
//    assert block.parent_root in store.block_states
//    # Make a copy of the state to avoid mutability issues
//    pre_state = copy(store.block_states[block.parent_root])
//    # Blocks cannot be in the future. If they are, their consideration must be delayed until the are in the past.
//    assert get_current_slot(store) >= block.slot
//
//    # Check that block is later than the finalized epoch slot (optimization to reduce calls to get_ancestor)
//    finalized_slot = compute_start_slot_at_epoch(store.finalized_checkpoint.epoch)
//    assert block.slot > finalized_slot
//    # Check block is a descendant of the finalized block at the checkpoint finalized slot
//    assert get_ancestor(store, block.parent_root, finalized_slot) == store.finalized_checkpoint.root
//
//    # Check the block is valid and compute the post-state
//    state = pre_state.copy()
//    state_transition(state, signed_block, True)
//    # Add new block to the store
//    store.blocks[hash_tree_root(block)] = block
//    # Add new state for this block to the store
//    store.block_states[hash_tree_root(block)] = state
//
//    # Update justified checkpoint
//    if state.current_justified_checkpoint.epoch > store.justified_checkpoint.epoch:
//        if state.current_justified_checkpoint.epoch > store.best_justified_checkpoint.epoch:
//            store.best_justified_checkpoint = state.current_justified_checkpoint
//        if should_update_justified_checkpoint(store, state.current_justified_checkpoint):
//            store.justified_checkpoint = state.current_justified_checkpoint
//
//    # Update finalized checkpoint
//    if state.finalized_checkpoint.epoch > store.finalized_checkpoint.epoch:
//        store.finalized_checkpoint = state.finalized_checkpoint
//
//        # Potentially update justified if different from store
//        if store.justified_checkpoint != state.current_justified_checkpoint:
//            # Update justified if new justified is later than store justified
//            if state.current_justified_checkpoint.epoch > store.justified_checkpoint.epoch:
//                store.justified_checkpoint = state.current_justified_checkpoint
//                return
//
//            # Update justified if store justified is not in chain with finalized checkpoint
//            finalized_slot = compute_start_slot_at_epoch(store.finalized_checkpoint.epoch)
//            ancestor_at_finalized_slot = get_ancestor(store, store.justified_checkpoint.root, finalized_slot)
//            if ancestor_at_finalized_slot != store.finalized_checkpoint.root:
//                store.justified_checkpoint = state.current_justified_checkpoint
func (s *Service) onBlock(ctx context.Context, signed block.SignedBeaconBlock, blockRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.onBlock")
	defer span.End()
	if err := helpers.BeaconBlockIsNil(signed); err != nil {
		return err
	}
	b := signed.Block()

	preState, err := s.getBlockPreState(ctx, b)
	if err != nil {
		return err
	}

	postState, err := transition.ExecuteStateTransition(ctx, preState, signed)
	if err != nil {
		return err
	}

	if err := s.savePostStateInfo(ctx, blockRoot, signed, postState, false /* reg sync */); err != nil {
		return err
	}

	// If slasher is configured, forward the attestations in the block via
	// an event feed for processing.
	if features.Get().EnableSlasher {
		// Feed the indexed attestation to slasher if enabled. This action
		// is done in the background to avoid adding more load to this critical code path.
		go func() {
			// Using a different context to prevent timeouts as this operation can be expensive
			// and we want to avoid affecting the critical code path.
			ctx := context.TODO()
			for _, att := range signed.Block().Body().Attestations() {
				committee, err := helpers.BeaconCommitteeFromState(ctx, preState, att.Data.Slot, att.Data.CommitteeIndex)
				if err != nil {
					log.WithError(err).Error("Could not get attestation committee")
					tracing.AnnotateError(span, err)
					return
				}
				indexedAtt, err := attestation.ConvertToIndexed(ctx, att, committee)
				if err != nil {
					log.WithError(err).Error("Could not convert to indexed attestation")
					tracing.AnnotateError(span, err)
					return
				}
				s.cfg.SlasherAttestationsFeed.Send(indexedAtt)
			}
		}()
	}

	// Update justified check point.
	currJustifiedEpoch := s.justifiedCheckpt.Epoch
	if postState.CurrentJustifiedCheckpoint().Epoch > currJustifiedEpoch {
		if err := s.updateJustified(ctx, postState); err != nil {
			return err
		}
	}

	newFinalized := postState.FinalizedCheckpointEpoch() > s.finalizedCheckpt.Epoch
	if newFinalized {
		if err := s.finalizedImpliesNewJustified(ctx, postState); err != nil {
			return errors.Wrap(err, "could not save new justified")
		}
		s.prevFinalizedCheckpt = s.finalizedCheckpt
		s.finalizedCheckpt = postState.FinalizedCheckpoint()
	}

	balances, err := s.justifiedBalances.get(ctx, bytesutil.ToBytes32(s.justifiedCheckpt.Root))
	if err != nil {
		msg := fmt.Sprintf("could not read balances for state w/ justified checkpoint %#x", s.justifiedCheckpt.Root)
		return errors.Wrap(err, msg)
	}
	if err := s.updateHead(ctx, balances); err != nil {
		log.WithError(err).Warn("Could not update head")
	}

	if err := s.pruneCanonicalAttsFromPool(ctx, blockRoot, signed); err != nil {
		return err
	}

	// Send notification of the processed block to the state feed.
	s.cfg.StateNotifier.StateFeed().Send(&feed.Event{
		Type: statefeed.BlockProcessed,
		Data: &statefeed.BlockProcessedData{
			Slot:        signed.Block().Slot(),
			BlockRoot:   blockRoot,
			SignedBlock: signed,
			Verified:    true,
		},
	})

	// Updating next slot state cache can happen in the background. It shouldn't block rest of the process.
	if features.Get().EnableNextSlotStateCache {
		go func() {
			// Use a custom deadline here, since this method runs asynchronously.
			// We ignore the parent method's context and instead create a new one
			// with a custom deadline, therefore using the background context instead.
			slotCtx, cancel := context.WithTimeout(context.Background(), slotDeadline)
			defer cancel()
			if err := transition.UpdateNextSlotCache(slotCtx, blockRoot[:], postState); err != nil {
				log.WithError(err).Debug("could not update next slot state cache")
			}
		}()
	}

	// Save justified check point to db.
	if postState.CurrentJustifiedCheckpoint().Epoch > currJustifiedEpoch {
		if err := s.cfg.BeaconDB.SaveJustifiedCheckpoint(ctx, postState.CurrentJustifiedCheckpoint()); err != nil {
			return err
		}
	}

	// Update finalized check point.
	if newFinalized {
		if err := s.updateFinalized(ctx, postState.FinalizedCheckpoint()); err != nil {
			return err
		}
		fRoot := bytesutil.ToBytes32(postState.FinalizedCheckpoint().Root)
		if err := s.cfg.ForkChoiceStore.Prune(ctx, fRoot); err != nil {
			return errors.Wrap(err, "could not prune proto array fork choice nodes")
		}
		go func() {
			// Send an event regarding the new finalized checkpoint over a common event feed.
			s.cfg.StateNotifier.StateFeed().Send(&feed.Event{
				Type: statefeed.FinalizedCheckpoint,
				Data: &ethpbv1.EventFinalizedCheckpoint{
					Epoch: postState.FinalizedCheckpoint().Epoch,
					Block: postState.FinalizedCheckpoint().Root,
					State: signed.Block().StateRoot(),
				},
			})

			// Use a custom deadline here, since this method runs asynchronously.
			// We ignore the parent method's context and instead create a new one
			// with a custom deadline, therefore using the background context instead.
			depCtx, cancel := context.WithTimeout(context.Background(), depositDeadline)
			defer cancel()
			if err := s.insertFinalizedDeposits(depCtx, fRoot); err != nil {
				log.WithError(err).Error("Could not insert finalized deposits.")
			}
		}()

	}

	defer reportAttestationInclusion(b)

	return s.handleEpochBoundary(ctx, postState)
}

func (s *Service) onBlockBatch(ctx context.Context, blks []block.SignedBeaconBlock,
	blockRoots [][32]byte) ([]*ethpb.Checkpoint, []*ethpb.Checkpoint, error) {
	ctx, span := trace.StartSpan(ctx, "blockChain.onBlockBatch")
	defer span.End()

	if len(blks) == 0 || len(blockRoots) == 0 {
		return nil, nil, errors.New("no blocks provided")
	}
	if err := helpers.BeaconBlockIsNil(blks[0]); err != nil {
		return nil, nil, err
	}
	b := blks[0].Block()

	// Retrieve incoming block's pre state.
	if err := s.verifyBlkPreState(ctx, b); err != nil {
		return nil, nil, err
	}
	preState, err := s.cfg.StateGen.StateByRootInitialSync(ctx, bytesutil.ToBytes32(b.ParentRoot()))
	if err != nil {
		return nil, nil, err
	}
	if preState == nil || preState.IsNil() {
		return nil, nil, fmt.Errorf("nil pre state for slot %d", b.Slot())
	}

	jCheckpoints := make([]*ethpb.Checkpoint, len(blks))
	fCheckpoints := make([]*ethpb.Checkpoint, len(blks))
	sigSet := &bls.SignatureBatch{
		Signatures: [][]byte{},
		PublicKeys: []bls.PublicKey{},
		Messages:   [][32]byte{},
	}
	var set *bls.SignatureBatch
	boundaries := make(map[[32]byte]state.BeaconState)
	for i, b := range blks {
		set, preState, err = transition.ExecuteStateTransitionNoVerifyAnySig(ctx, preState, b)
		if err != nil {
			return nil, nil, err
		}
		// Save potential boundary states.
		if slots.IsEpochStart(preState.Slot()) {
			boundaries[blockRoots[i]] = preState.Copy()
			if err := s.handleEpochBoundary(ctx, preState); err != nil {
				return nil, nil, errors.Wrap(err, "could not handle epoch boundary state")
			}
		}
		jCheckpoints[i] = preState.CurrentJustifiedCheckpoint()
		fCheckpoints[i] = preState.FinalizedCheckpoint()
		sigSet.Join(set)
	}
	verify, err := sigSet.Verify()
	if err != nil {
		return nil, nil, err
	}
	if !verify {
		return nil, nil, errors.New("batch block signature verification failed")
	}
	for r, st := range boundaries {
		if err := s.cfg.StateGen.SaveState(ctx, r, st); err != nil {
			return nil, nil, err
		}
	}
	// Also saves the last post state which to be used as pre state for the next batch.
	lastB := blks[len(blks)-1]
	lastBR := blockRoots[len(blockRoots)-1]
	if err := s.cfg.StateGen.SaveState(ctx, lastBR, preState); err != nil {
		return nil, nil, err
	}
	if err := s.saveHeadNoDB(ctx, lastB, lastBR, preState); err != nil {
		return nil, nil, err
	}
	return fCheckpoints, jCheckpoints, nil
}

// handles a block after the block's batch has been verified, where we can save blocks
// their state summaries and split them off to relative hot/cold storage.
func (s *Service) handleBlockAfterBatchVerify(ctx context.Context, signed block.SignedBeaconBlock,
	blockRoot [32]byte, fCheckpoint, jCheckpoint *ethpb.Checkpoint) error {
	b := signed.Block()

	s.saveInitSyncBlock(blockRoot, signed)
	if err := s.insertBlockToForkChoiceStore(ctx, b, blockRoot, fCheckpoint, jCheckpoint); err != nil {
		return err
	}
	if err := s.cfg.BeaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{
		Slot: signed.Block().Slot(),
		Root: blockRoot[:],
	}); err != nil {
		return err
	}

	// Rate limit how many blocks (2 epochs worth of blocks) a node keeps in the memory.
	if uint64(len(s.getInitSyncBlocks())) > initialSyncBlockCacheSize {
		if err := s.cfg.BeaconDB.SaveBlocks(ctx, s.getInitSyncBlocks()); err != nil {
			return err
		}
		s.clearInitSyncBlocks()
	}

	if jCheckpoint.Epoch > s.justifiedCheckpt.Epoch {
		if err := s.updateJustifiedInitSync(ctx, jCheckpoint); err != nil {
			return err
		}
	}

	// Update finalized check point. Prune the block cache and helper caches on every new finalized epoch.
	if fCheckpoint.Epoch > s.finalizedCheckpt.Epoch {
		if err := s.updateFinalized(ctx, fCheckpoint); err != nil {
			return err
		}
		s.prevFinalizedCheckpt = s.finalizedCheckpt
		s.finalizedCheckpt = fCheckpoint
	}
	return nil
}

// Epoch boundary bookkeeping such as logging epoch summaries.
func (s *Service) handleEpochBoundary(ctx context.Context, postState state.BeaconState) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.handleEpochBoundary")
	defer span.End()

	if postState.Slot()+1 == s.nextEpochBoundarySlot {
		// Update caches for the next epoch at epoch boundary slot - 1.
		if err := helpers.UpdateCommitteeCache(postState, coreTime.NextEpoch(postState)); err != nil {
			return err
		}
		copied := postState.Copy()
		copied, err := transition.ProcessSlots(ctx, copied, copied.Slot()+1)
		if err != nil {
			return err
		}
		if err := helpers.UpdateProposerIndicesInCache(ctx, copied); err != nil {
			return err
		}
	} else if postState.Slot() >= s.nextEpochBoundarySlot {
		if err := reportEpochMetrics(ctx, postState, s.head.state); err != nil {
			return err
		}
		var err error
		s.nextEpochBoundarySlot, err = slots.EpochStart(coreTime.NextEpoch(postState))
		if err != nil {
			return err
		}

		// Update caches at epoch boundary slot.
		// The following updates have short cut to return nil cheaply if fulfilled during boundary slot - 1.
		if err := helpers.UpdateCommitteeCache(postState, coreTime.CurrentEpoch(postState)); err != nil {
			return err
		}
		if err := helpers.UpdateProposerIndicesInCache(ctx, postState); err != nil {
			return err
		}
	}

	return nil
}

// This feeds in the block and block's attestations to fork choice store. It's allows fork choice store
// to gain information on the most current chain.
func (s *Service) insertBlockAndAttestationsToForkChoiceStore(ctx context.Context, blk block.BeaconBlock, root [32]byte,
	st state.BeaconState) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.insertBlockAndAttestationsToForkChoiceStore")
	defer span.End()

	fCheckpoint := st.FinalizedCheckpoint()
	jCheckpoint := st.CurrentJustifiedCheckpoint()
	if err := s.insertBlockToForkChoiceStore(ctx, blk, root, fCheckpoint, jCheckpoint); err != nil {
		return err
	}
	// Feed in block's attestations to fork choice store.
	for _, a := range blk.Body().Attestations() {
		committee, err := helpers.BeaconCommitteeFromState(ctx, st, a.Data.Slot, a.Data.CommitteeIndex)
		if err != nil {
			return err
		}
		indices, err := attestation.AttestingIndices(a.AggregationBits, committee)
		if err != nil {
			return err
		}
		s.cfg.ForkChoiceStore.ProcessAttestation(ctx, indices, bytesutil.ToBytes32(a.Data.BeaconBlockRoot), a.Data.Target.Epoch)
	}
	return nil
}

func (s *Service) insertBlockToForkChoiceStore(ctx context.Context, blk block.BeaconBlock,
	root [32]byte, fCheckpoint, jCheckpoint *ethpb.Checkpoint) error {
	if err := s.fillInForkChoiceMissingBlocks(ctx, blk, fCheckpoint, jCheckpoint); err != nil {
		return err
	}
	// Feed in block to fork choice store.
	if err := s.cfg.ForkChoiceStore.ProcessBlock(ctx,
		blk.Slot(), root, bytesutil.ToBytes32(blk.ParentRoot()), bytesutil.ToBytes32(blk.Body().Graffiti()),
		jCheckpoint.Epoch,
		fCheckpoint.Epoch); err != nil {
		return errors.Wrap(err, "could not process block for proto array fork choice")
	}
	return nil
}

// This saves post state info to DB or cache. This also saves post state info to fork choice store.
// Post state info consists of processed block and state. Do not call this method unless the block and state are verified.
func (s *Service) savePostStateInfo(ctx context.Context, r [32]byte, b block.SignedBeaconBlock, st state.BeaconState, initSync bool) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.savePostStateInfo")
	defer span.End()
	if initSync {
		s.saveInitSyncBlock(r, b)
	} else if err := s.cfg.BeaconDB.SaveBlock(ctx, b); err != nil {
		return errors.Wrapf(err, "could not save block from slot %d", b.Block().Slot())
	}
	if err := s.cfg.StateGen.SaveState(ctx, r, st); err != nil {
		return errors.Wrap(err, "could not save state")
	}
	if err := s.insertBlockAndAttestationsToForkChoiceStore(ctx, b.Block(), r, st); err != nil {
		return errors.Wrapf(err, "could not insert block %d to fork choice store", b.Block().Slot())
	}
	return nil
}

// This removes the attestations from the mem pool. It will only remove the attestations if input root `r` is canonical,
// meaning the block `b` is part of the canonical chain.
func (s *Service) pruneCanonicalAttsFromPool(ctx context.Context, r [32]byte, b block.SignedBeaconBlock) error {
	if !features.Get().CorrectlyPruneCanonicalAtts {
		return nil
	}

	canonical, err := s.IsCanonical(ctx, r)
	if err != nil {
		return err
	}
	if !canonical {
		return nil
	}

	atts := b.Block().Body().Attestations()
	for _, att := range atts {
		if helpers.IsAggregated(att) {
			if err := s.cfg.AttPool.DeleteAggregatedAttestation(att); err != nil {
				return err
			}
		} else {
			if err := s.cfg.AttPool.DeleteUnaggregatedAttestation(att); err != nil {
				return err
			}
		}
	}
	return nil
}
