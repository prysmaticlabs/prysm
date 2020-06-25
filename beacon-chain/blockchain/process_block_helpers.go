package blockchain

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// CurrentSlot returns the current slot based on time.
func (s *Service) CurrentSlot() uint64 {
	now := roughtime.Now().Unix()
	genesis := s.genesisTime.Unix()
	if now < genesis {
		return 0
	}
	return uint64(now-genesis) / params.BeaconConfig().SecondsPerSlot
}

// getBlockPreState returns the pre state of an incoming block. It uses the parent root of the block
// to retrieve the state in DB. It verifies the pre state's validity and the incoming block
// is in the correct time window.
func (s *Service) getBlockPreState(ctx context.Context, b *ethpb.BeaconBlock) (*stateTrie.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "forkchoice.getBlockPreState")
	defer span.End()

	// Verify incoming block has a valid pre state.
	preState, err := s.verifyBlkPreState(ctx, b)
	if err != nil {
		return nil, err
	}

	preState, err = s.stateGen.StateByRoot(ctx, bytesutil.ToBytes32(b.ParentRoot))
	if err != nil {
		return nil, errors.Wrapf(err, "could not get pre state for slot %d", b.Slot)
	}
	if preState == nil {
		return nil, errors.Wrapf(err, "nil pre state for slot %d", b.Slot)
	}

	// Verify block slot time is not from the future.
	if err := helpers.VerifySlotTime(preState.GenesisTime(), b.Slot, params.BeaconNetworkConfig().MaximumGossipClockDisparity); err != nil {
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
func (s *Service) verifyBlkPreState(ctx context.Context, b *ethpb.BeaconBlock) (*stateTrie.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "chainService.verifyBlkPreState")
	defer span.End()

	parentRoot := bytesutil.ToBytes32(b.ParentRoot)
	// Loosen the check to HasBlock because state summary gets saved in batches
	// during initial syncing. There's no risk given a state summary object is just a
	// a subset of the block object.
	if !s.stateGen.StateSummaryExists(ctx, parentRoot) && !s.beaconDB.HasBlock(ctx, parentRoot) {
		return nil, errors.New("could not reconstruct parent state")
	}
	if !s.stateGen.HasState(ctx, parentRoot) {
		if err := s.beaconDB.SaveBlocks(ctx, s.getInitSyncBlocks()); err != nil {
			return nil, errors.Wrap(err, "could not save initial sync blocks")
		}
		s.clearInitSyncBlocks()
	}
	preState, err := s.stateGen.StateByRootInitialSync(ctx, parentRoot)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get pre state for slot %d", b.Slot)
	}
	if preState == nil {
		return nil, errors.Wrapf(err, "nil pre state for slot %d", b.Slot)
	}

	return preState, nil // No copy needed from newly hydrated state gen object.
}

// verifyBlkDescendant validates input block root is a descendant of the
// current finalized block root.
func (s *Service) verifyBlkDescendant(ctx context.Context, root [32]byte, slot uint64) error {
	ctx, span := trace.StartSpan(ctx, "forkchoice.verifyBlkDescendant")
	defer span.End()

	finalizedBlkSigned, err := s.beaconDB.Block(ctx, bytesutil.ToBytes32(s.finalizedCheckpt.Root))
	if err != nil || finalizedBlkSigned == nil || finalizedBlkSigned.Block == nil {
		return errors.Wrap(err, "could not get finalized block")
	}
	finalizedBlk := finalizedBlkSigned.Block

	bFinalizedRoot, err := s.ancestor(ctx, root[:], finalizedBlk.Slot)
	if err != nil {
		return errors.Wrap(err, "could not get finalized block root")
	}
	if bFinalizedRoot == nil {
		return fmt.Errorf("no finalized block known for block from slot %d", slot)
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
func (s *Service) verifyBlkFinalizedSlot(b *ethpb.BeaconBlock) error {
	finalizedSlot := helpers.StartSlot(s.finalizedCheckpt.Epoch)
	if finalizedSlot >= b.Slot {
		return fmt.Errorf("block is equal or earlier than finalized block, slot %d < slot %d", b.Slot, finalizedSlot)
	}
	return nil
}

// shouldUpdateCurrentJustified prevents bouncing attack, by only update conflicting justified
// checkpoints in the fork choice if in the early slots of the epoch.
// Otherwise, delay incorporation of new justified checkpoint until next epoch boundary.
// See https://ethresear.ch/t/prevention-of-bouncing-attack-on-ffg/6114 for more detailed analysis and discussion.
func (s *Service) shouldUpdateCurrentJustified(ctx context.Context, newJustifiedCheckpt *ethpb.Checkpoint) (bool, error) {
	if helpers.SlotsSinceEpochStarts(s.CurrentSlot()) < params.BeaconConfig().SafeSlotsToUpdateJustified {
		return true, nil
	}
	var newJustifiedBlockSigned *ethpb.SignedBeaconBlock
	justifiedRoot := s.ensureRootNotZeros(bytesutil.ToBytes32(newJustifiedCheckpt.Root))
	var err error
	if s.hasInitSyncBlock(justifiedRoot) {
		newJustifiedBlockSigned = s.getInitSyncBlock(justifiedRoot)
	} else {
		newJustifiedBlockSigned, err = s.beaconDB.Block(ctx, justifiedRoot)
		if err != nil {
			return false, err
		}
	}
	if newJustifiedBlockSigned == nil || newJustifiedBlockSigned.Block == nil {
		return false, errors.New("nil new justified block")
	}

	newJustifiedBlock := newJustifiedBlockSigned.Block
	if newJustifiedBlock.Slot <= helpers.StartSlot(s.justifiedCheckpt.Epoch) {
		return false, nil
	}
	var justifiedBlockSigned *ethpb.SignedBeaconBlock
	cachedJustifiedRoot := s.ensureRootNotZeros(bytesutil.ToBytes32(s.justifiedCheckpt.Root))
	if s.hasInitSyncBlock(cachedJustifiedRoot) {
		justifiedBlockSigned = s.getInitSyncBlock(cachedJustifiedRoot)
	} else {
		justifiedBlockSigned, err = s.beaconDB.Block(ctx, cachedJustifiedRoot)
		if err != nil {
			return false, err
		}
	}

	if justifiedBlockSigned == nil || justifiedBlockSigned.Block == nil {
		return false, errors.New("nil justified block")
	}
	justifiedBlock := justifiedBlockSigned.Block
	b, err := s.ancestor(ctx, justifiedRoot[:], justifiedBlock.Slot)
	if err != nil {
		return false, err
	}
	if !bytes.Equal(b, s.justifiedCheckpt.Root) {
		return false, nil
	}
	return true, nil
}

func (s *Service) updateJustified(ctx context.Context, state *stateTrie.BeaconState) error {
	cpt := state.CurrentJustifiedCheckpoint()
	if cpt.Epoch > s.bestJustifiedCheckpt.Epoch {
		s.bestJustifiedCheckpt = cpt
	}
	canUpdate, err := s.shouldUpdateCurrentJustified(ctx, cpt)
	if err != nil {
		return err
	}

	if canUpdate {
		s.prevJustifiedCheckpt = s.justifiedCheckpt
		s.justifiedCheckpt = cpt
		if err := s.cacheJustifiedStateBalances(ctx, bytesutil.ToBytes32(s.justifiedCheckpt.Root)); err != nil {
			return err
		}
	}

	return s.beaconDB.SaveJustifiedCheckpoint(ctx, cpt)
}

// ancestor returns the block root of an ancestry block from the input block root.
//
// Spec pseudocode definition:
//   def get_ancestor(store: Store, root: Root, slot: Slot) -> Root:
//    block = store.blocks[root]
//    if block.slot > slot:
//        return get_ancestor(store, block.parent_root, slot)
//    elif block.slot == slot:
//        return root
//    else:
//        # root is older than queried slot, thus a skip slot. Return most recent root prior to slot
//        return root
func (s *Service) ancestor(ctx context.Context, root []byte, slot uint64) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "forkchoice.ancestor")
	defer span.End()

	// Stop recursive ancestry lookup if context is cancelled.
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	signed, err := s.beaconDB.Block(ctx, bytesutil.ToBytes32(root))
	if err != nil {
		return nil, errors.Wrap(err, "could not get ancestor block")
	}

	if s.hasInitSyncBlock(bytesutil.ToBytes32(root)) {
		signed = s.getInitSyncBlock(bytesutil.ToBytes32(root))
	}

	if signed == nil || signed.Block == nil {
		return nil, errors.New("nil block")
	}
	b := signed.Block

	if b.Slot == slot || b.Slot < slot {
		return root, nil
	}

	return s.ancestor(ctx, b.ParentRoot, slot)
}

// This updates justified check point in store, if the new justified is later than stored justified or
// the store's justified is not in chain with finalized check point.
//
// Spec definition:
//   # Potentially update justified if different from store
//        if store.justified_checkpoint != state.current_justified_checkpoint:
//            # Update justified if new justified is later than store justified
//            if state.current_justified_checkpoint.epoch > store.justified_checkpoint.epoch:
//                store.justified_checkpoint = state.current_justified_checkpoint
//                return
//            # Update justified if store justified is not in chain with finalized checkpoint
//            finalized_slot = compute_start_slot_at_epoch(store.finalized_checkpoint.epoch)
//            ancestor_at_finalized_slot = get_ancestor(store, store.justified_checkpoint.root, finalized_slot)
//            if ancestor_at_finalized_slot != store.finalized_checkpoint.root:
//                store.justified_checkpoint = state.current_justified_checkpoint
func (s *Service) finalizedImpliesNewJustified(ctx context.Context, state *stateTrie.BeaconState) error {
	// Update justified if it's different than the one cached in the store.
	if !attestationutil.CheckPointIsEqual(s.justifiedCheckpt, state.CurrentJustifiedCheckpoint()) {
		if state.CurrentJustifiedCheckpoint().Epoch > s.justifiedCheckpt.Epoch {
			s.justifiedCheckpt = state.CurrentJustifiedCheckpoint()
			if err := s.cacheJustifiedStateBalances(ctx, bytesutil.ToBytes32(s.justifiedCheckpt.Root)); err != nil {
				return err
			}
			return nil
		}

		// Update justified if store justified is not in chain with finalized check point.
		finalizedSlot := helpers.StartSlot(s.finalizedCheckpt.Epoch)
		justifiedRoot := s.ensureRootNotZeros(bytesutil.ToBytes32(s.justifiedCheckpt.Root))
		anc, err := s.ancestor(ctx, justifiedRoot[:], finalizedSlot)
		if err != nil {
			return err
		}
		if !bytes.Equal(anc, s.finalizedCheckpt.Root) {
			s.justifiedCheckpt = state.CurrentJustifiedCheckpoint()
			if err := s.cacheJustifiedStateBalances(ctx, bytesutil.ToBytes32(s.justifiedCheckpt.Root)); err != nil {
				return err
			}
		}
	}
	return nil
}

// This retrieves missing blocks from DB (ie. the blocks that couldn't be received over sync) and inserts them to fork choice store.
// This is useful for block tree visualizer and additional vote accounting.
func (s *Service) fillInForkChoiceMissingBlocks(ctx context.Context, blk *ethpb.BeaconBlock, state *stateTrie.BeaconState) error {
	pendingNodes := make([]*ethpb.BeaconBlock, 0)

	parentRoot := bytesutil.ToBytes32(blk.ParentRoot)
	slot := blk.Slot
	// Fork choice only matters from last finalized slot.
	higherThanFinalized := slot > helpers.StartSlot(s.finalizedCheckpt.Epoch)
	// As long as parent node is not in fork choice store, and parent node is in DB.
	for !s.forkChoiceStore.HasNode(parentRoot) && s.beaconDB.HasBlock(ctx, parentRoot) && higherThanFinalized {
		b, err := s.beaconDB.Block(ctx, parentRoot)
		if err != nil {
			return err
		}

		pendingNodes = append(pendingNodes, b.Block)
		parentRoot = bytesutil.ToBytes32(b.Block.ParentRoot)
		slot = b.Block.Slot
		higherThanFinalized = slot > helpers.StartSlot(s.finalizedCheckpt.Epoch)
	}

	// Insert parent nodes to fork choice store in reverse order.
	// Lower slots should be at the end of the list.
	for i := len(pendingNodes) - 1; i >= 0; i-- {
		b := pendingNodes[i]
		r, err := stateutil.BlockRoot(b)
		if err != nil {
			return err
		}

		if err := s.forkChoiceStore.ProcessBlock(ctx,
			b.Slot, r, bytesutil.ToBytes32(b.ParentRoot), bytesutil.ToBytes32(b.Body.Graffiti),
			state.CurrentJustifiedCheckpoint().Epoch,
			state.FinalizedCheckpointEpoch()); err != nil {
			return errors.Wrap(err, "could not process block for proto array fork choice")
		}
	}

	return nil
}

// The deletes input attestations from the attestation pool, so proposers don't include them in a block for the future.
func (s *Service) deletePoolAtts(atts []*ethpb.Attestation) error {
	for _, att := range atts {
		if helpers.IsAggregated(att) {
			if err := s.attPool.DeleteAggregatedAttestation(att); err != nil {
				return err
			}
		} else {
			if err := s.attPool.DeleteUnaggregatedAttestation(att); err != nil {
				return err
			}
		}
	}

	return nil
}

// This ensures that the input root defaults to using genesis root instead of zero hashes. This is needed for handling
// fork choice justification routine.
func (s *Service) ensureRootNotZeros(root [32]byte) [32]byte {
	if root == params.BeaconConfig().ZeroHash {
		return s.genesisRoot
	}
	return root
}
