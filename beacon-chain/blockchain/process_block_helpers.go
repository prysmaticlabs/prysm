package blockchain

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/monitoring/tracing"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/attestation"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/time/slots"
	"go.opencensus.io/trace"
)

// CurrentSlot returns the current slot based on time.
func (s *Service) CurrentSlot() types.Slot {
	return slots.CurrentSlot(uint64(s.genesisTime.Unix()))
}

// getBlockPreState returns the pre state of an incoming block. It uses the parent root of the block
// to retrieve the state in DB. It verifies the pre state's validity and the incoming block
// is in the correct time window.
func (s *Service) getBlockPreState(ctx context.Context, b block.BeaconBlock) (state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "blockChain.getBlockPreState")
	defer span.End()

	// Verify incoming block has a valid pre state.
	if err := s.verifyBlkPreState(ctx, b); err != nil {
		return nil, err
	}

	preState, err := s.cfg.StateGen.StateByRoot(ctx, bytesutil.ToBytes32(b.ParentRoot()))
	if err != nil {
		return nil, errors.Wrapf(err, "could not get pre state for slot %d", b.Slot())
	}
	if preState == nil || preState.IsNil() {
		return nil, errors.Wrapf(err, "nil pre state for slot %d", b.Slot())
	}

	// Verify block slot time is not from the future.
	if err := slots.VerifyTime(preState.GenesisTime(), b.Slot(), params.BeaconNetworkConfig().MaximumGossipClockDisparity); err != nil {
		return nil, err
	}

	// Verify block is later than the finalized epoch slot.
	if err := s.verifyBlkFinalizedSlot(b); err != nil {
		return nil, err
	}

	return preState, nil
}

// verifyBlkPreState validates input block has a valid pre-state.
func (s *Service) verifyBlkPreState(ctx context.Context, b block.BeaconBlock) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.verifyBlkPreState")
	defer span.End()

	parentRoot := bytesutil.ToBytes32(b.ParentRoot())
	// Loosen the check to HasBlock because state summary gets saved in batches
	// during initial syncing. There's no risk given a state summary object is just a
	// a subset of the block object.
	if !s.cfg.BeaconDB.HasStateSummary(ctx, parentRoot) && !s.cfg.BeaconDB.HasBlock(ctx, parentRoot) {
		return errors.New("could not reconstruct parent state")
	}

	if err := s.VerifyBlkDescendant(ctx, bytesutil.ToBytes32(b.ParentRoot())); err != nil {
		return err
	}

	has, err := s.cfg.StateGen.HasState(ctx, parentRoot)
	if err != nil {
		return err
	}
	if !has {
		if err := s.cfg.BeaconDB.SaveBlocks(ctx, s.getInitSyncBlocks()); err != nil {
			return errors.Wrap(err, "could not save initial sync blocks")
		}
		s.clearInitSyncBlocks()
	}
	return nil
}

// VerifyBlkDescendant validates input block root is a descendant of the
// current finalized block root.
func (s *Service) VerifyBlkDescendant(ctx context.Context, root [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.VerifyBlkDescendant")
	defer span.End()
	fRoot := s.ensureRootNotZeros(bytesutil.ToBytes32(s.finalizedCheckpt.Root))
	finalizedBlkSigned, err := s.cfg.BeaconDB.Block(ctx, fRoot)
	if err != nil {
		return err
	}
	if finalizedBlkSigned == nil || finalizedBlkSigned.IsNil() || finalizedBlkSigned.Block().IsNil() {
		return errors.New("nil finalized block")
	}
	finalizedBlk := finalizedBlkSigned.Block()
	bFinalizedRoot, err := s.ancestor(ctx, root[:], finalizedBlk.Slot())
	if err != nil {
		return errors.Wrap(err, "could not get finalized block root")
	}
	if bFinalizedRoot == nil {
		return fmt.Errorf("no finalized block known for block %#x", bytesutil.Trunc(root[:]))
	}

	if !bytes.Equal(bFinalizedRoot, fRoot[:]) {
		err := fmt.Errorf("block %#x is not a descendent of the current finalized block slot %d, %#x != %#x",
			bytesutil.Trunc(root[:]), finalizedBlk.Slot(), bytesutil.Trunc(bFinalizedRoot),
			bytesutil.Trunc(fRoot[:]))
		tracing.AnnotateError(span, err)
		return err
	}
	return nil
}

// verifyBlkFinalizedSlot validates input block is not less than or equal
// to current finalized slot.
func (s *Service) verifyBlkFinalizedSlot(b block.BeaconBlock) error {
	finalizedSlot, err := slots.EpochStart(s.finalizedCheckpt.Epoch)
	if err != nil {
		return err
	}
	if finalizedSlot >= b.Slot() {
		return fmt.Errorf("block is equal or earlier than finalized block, slot %d < slot %d", b.Slot(), finalizedSlot)
	}
	return nil
}

// shouldUpdateCurrentJustified prevents bouncing attack, by only update conflicting justified
// checkpoints in the fork choice if in the early slots of the epoch.
// Otherwise, delay incorporation of new justified checkpoint until next epoch boundary.
//
// Spec code:
// def should_update_justified_checkpoint(store: Store, new_justified_checkpoint: Checkpoint) -> bool:
//    """
//    To address the bouncing attack, only update conflicting justified
//    checkpoints in the fork choice if in the early slots of the epoch.
//    Otherwise, delay incorporation of new justified checkpoint until next epoch boundary.
//
//    See https://ethresear.ch/t/prevention-of-bouncing-attack-on-ffg/6114 for more detailed analysis and discussion.
//    """
//    if compute_slots_since_epoch_start(get_current_slot(store)) < SAFE_SLOTS_TO_UPDATE_JUSTIFIED:
//        return True
//
//    justified_slot = compute_start_slot_at_epoch(store.justified_checkpoint.epoch)
//    if not get_ancestor(store, new_justified_checkpoint.root, justified_slot) == store.justified_checkpoint.root:
//        return False
//
//    return True
func (s *Service) shouldUpdateCurrentJustified(ctx context.Context, newJustifiedCheckpt *ethpb.Checkpoint) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "blockChain.shouldUpdateCurrentJustified")
	defer span.End()

	if slots.SinceEpochStarts(s.CurrentSlot()) < params.BeaconConfig().SafeSlotsToUpdateJustified {
		return true, nil
	}

	jSlot, err := slots.EpochStart(s.justifiedCheckpt.Epoch)
	if err != nil {
		return false, err
	}
	justifiedRoot := s.ensureRootNotZeros(bytesutil.ToBytes32(newJustifiedCheckpt.Root))
	b, err := s.ancestor(ctx, justifiedRoot[:], jSlot)
	if err != nil {
		return false, err
	}
	if !bytes.Equal(b, s.justifiedCheckpt.Root) {
		return false, nil
	}

	return true, nil
}

func (s *Service) updateJustified(ctx context.Context, state state.ReadOnlyBeaconState) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.updateJustified")
	defer span.End()

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
	}

	return nil
}

// This caches input checkpoint as justified for the service struct. It rotates current justified to previous justified,
// caches justified checkpoint balances for fork choice and save justified checkpoint in DB.
// This method does not have defense against fork choice bouncing attack, which is why it's only recommend to be used during initial syncing.
func (s *Service) updateJustifiedInitSync(ctx context.Context, cp *ethpb.Checkpoint) error {
	s.prevJustifiedCheckpt = s.justifiedCheckpt

	if err := s.cfg.BeaconDB.SaveJustifiedCheckpoint(ctx, cp); err != nil {
		return err
	}
	s.justifiedCheckpt = cp

	return nil
}

func (s *Service) updateFinalized(ctx context.Context, cp *ethpb.Checkpoint) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.updateFinalized")
	defer span.End()

	// Blocks need to be saved so that we can retrieve finalized block from
	// DB when migrating states.
	if err := s.cfg.BeaconDB.SaveBlocks(ctx, s.getInitSyncBlocks()); err != nil {
		return err
	}
	s.clearInitSyncBlocks()

	if err := s.cfg.BeaconDB.SaveFinalizedCheckpoint(ctx, cp); err != nil {
		return err
	}

	fRoot := bytesutil.ToBytes32(cp.Root)
	if err := s.cfg.StateGen.MigrateToCold(ctx, fRoot); err != nil {
		return errors.Wrap(err, "could not migrate to cold")
	}

	return nil
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
func (s *Service) ancestor(ctx context.Context, root []byte, slot types.Slot) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "blockChain.ancestor")
	defer span.End()

	r := bytesutil.ToBytes32(root)
	// Get ancestor root from fork choice store instead of recursively looking up blocks in DB.
	// This is most optimal outcome.
	ar, err := s.ancestorByForkChoiceStore(ctx, r, slot)
	if err != nil {
		// Try getting ancestor root from DB when failed to retrieve from fork choice store.
		// This is the second line of defense for retrieving ancestor root.
		ar, err = s.ancestorByDB(ctx, r, slot)
		if err != nil {
			return nil, err
		}
	}

	return ar, nil
}

// This retrieves an ancestor root using fork choice store. The look up is looping through the a flat array structure.
func (s *Service) ancestorByForkChoiceStore(ctx context.Context, r [32]byte, slot types.Slot) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "blockChain.ancestorByForkChoiceStore")
	defer span.End()

	if !s.cfg.ForkChoiceStore.HasParent(r) {
		return nil, errors.New("could not find root in fork choice store")
	}
	return s.cfg.ForkChoiceStore.AncestorRoot(ctx, r, slot)
}

// This retrieves an ancestor root using DB. The look up is recursively looking up DB. Slower than `ancestorByForkChoiceStore`.
func (s *Service) ancestorByDB(ctx context.Context, r [32]byte, slot types.Slot) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "blockChain.ancestorByDB")
	defer span.End()

	// Stop recursive ancestry lookup if context is cancelled.
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	signed, err := s.cfg.BeaconDB.Block(ctx, r)
	if err != nil {
		return nil, errors.Wrap(err, "could not get ancestor block")
	}

	if s.hasInitSyncBlock(r) {
		signed = s.getInitSyncBlock(r)
	}

	if signed == nil || signed.IsNil() || signed.Block().IsNil() {
		return nil, errors.New("nil block")
	}
	b := signed.Block()
	if b.Slot() == slot || b.Slot() < slot {
		return r[:], nil
	}

	return s.ancestorByDB(ctx, bytesutil.ToBytes32(b.ParentRoot()), slot)
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
func (s *Service) finalizedImpliesNewJustified(ctx context.Context, state state.BeaconState) error {
	// Update justified if it's different than the one cached in the store.
	if !attestation.CheckPointIsEqual(s.justifiedCheckpt, state.CurrentJustifiedCheckpoint()) {
		if state.CurrentJustifiedCheckpoint().Epoch > s.justifiedCheckpt.Epoch {
			s.justifiedCheckpt = state.CurrentJustifiedCheckpoint()
			// we don't need to check if the previous justified checkpoint was an ancestor since the new
			// finalized checkpoint is overriding it.
			return nil
		}

		// Update justified if store justified is not in chain with finalized check point.
		finalizedSlot, err := slots.EpochStart(s.finalizedCheckpt.Epoch)
		if err != nil {
			return err
		}
		justifiedRoot := s.ensureRootNotZeros(bytesutil.ToBytes32(s.justifiedCheckpt.Root))
		anc, err := s.ancestor(ctx, justifiedRoot[:], finalizedSlot)
		if err != nil {
			return err
		}
		if !bytes.Equal(anc, s.finalizedCheckpt.Root) {
			s.justifiedCheckpt = state.CurrentJustifiedCheckpoint()
		}
	}
	return nil
}

// This retrieves missing blocks from DB (ie. the blocks that couldn't be received over sync) and inserts them to fork choice store.
// This is useful for block tree visualizer and additional vote accounting.
func (s *Service) fillInForkChoiceMissingBlocks(ctx context.Context, blk block.BeaconBlock,
	fCheckpoint, jCheckpoint *ethpb.Checkpoint) error {
	pendingNodes := make([]block.BeaconBlock, 0)
	pendingRoots := make([][32]byte, 0)

	parentRoot := bytesutil.ToBytes32(blk.ParentRoot())
	slot := blk.Slot()
	// Fork choice only matters from last finalized slot.
	fSlot, err := slots.EpochStart(s.finalizedCheckpt.Epoch)
	if err != nil {
		return err
	}
	higherThanFinalized := slot > fSlot
	// As long as parent node is not in fork choice store, and parent node is in DB.
	for !s.cfg.ForkChoiceStore.HasNode(parentRoot) && s.cfg.BeaconDB.HasBlock(ctx, parentRoot) && higherThanFinalized {
		b, err := s.cfg.BeaconDB.Block(ctx, parentRoot)
		if err != nil {
			return err
		}

		pendingNodes = append(pendingNodes, b.Block())
		copiedRoot := parentRoot
		pendingRoots = append(pendingRoots, copiedRoot)
		parentRoot = bytesutil.ToBytes32(b.Block().ParentRoot())
		slot = b.Block().Slot()
		higherThanFinalized = slot > fSlot
	}

	// Insert parent nodes to fork choice store in reverse order.
	// Lower slots should be at the end of the list.
	for i := len(pendingNodes) - 1; i >= 0; i-- {
		b := pendingNodes[i]
		r := pendingRoots[i]
		if err := s.cfg.ForkChoiceStore.ProcessBlock(ctx,
			b.Slot(), r, bytesutil.ToBytes32(b.ParentRoot()), bytesutil.ToBytes32(b.Body().Graffiti()),
			jCheckpoint.Epoch,
			fCheckpoint.Epoch); err != nil {
			return errors.Wrap(err, "could not process block for proto array fork choice")
		}
	}

	return nil
}

// inserts finalized deposits into our finalized deposit trie.
func (s *Service) insertFinalizedDeposits(ctx context.Context, fRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.insertFinalizedDeposits")
	defer span.End()

	// Update deposit cache.
	finalizedState, err := s.cfg.StateGen.StateByRoot(ctx, fRoot)
	if err != nil {
		return errors.Wrap(err, "could not fetch finalized state")
	}
	// We update the cache up to the last deposit index in the finalized block's state.
	// We can be confident that these deposits will be included in some block
	// because the Eth1 follow distance makes such long-range reorgs extremely unlikely.
	eth1DepositIndex := int64(finalizedState.Eth1Data().DepositCount - 1)
	s.cfg.DepositCache.InsertFinalizedDeposits(ctx, eth1DepositIndex)
	// Deposit proofs are only used during state transition and can be safely removed to save space.
	if err = s.cfg.DepositCache.PruneProofs(ctx, eth1DepositIndex); err != nil {
		return errors.Wrap(err, "could not prune deposit proofs")
	}
	return nil
}

// The deletes input attestations from the attestation pool, so proposers don't include them in a block for the future.
func (s *Service) deletePoolAtts(atts []*ethpb.Attestation) error {
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

// This ensures that the input root defaults to using genesis root instead of zero hashes. This is needed for handling
// fork choice justification routine.
func (s *Service) ensureRootNotZeros(root [32]byte) [32]byte {
	if root == params.BeaconConfig().ZeroHash {
		return s.originBlockRoot
	}
	return root
}
