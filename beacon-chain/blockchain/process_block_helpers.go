package blockchain

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
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

	//  For new state management, this ensures the state does not get mutated since initial syncing
	//  uses verifyBlkPreState.
	if featureconfig.Get().NewStateMgmt {
		preState, err = s.stateGen.StateByRoot(ctx, bytesutil.ToBytes32(b.ParentRoot))
		if err != nil {
			return nil, errors.Wrapf(err, "could not get pre state for slot %d", b.Slot)
		}
		if preState == nil {
			return nil, errors.Wrapf(err, "nil pre state for slot %d", b.Slot)
		}
	}

	// Verify block slot time is not from the feature.
	if err := helpers.VerifySlotTime(preState.GenesisTime(), b.Slot, helpers.TimeShiftTolerance); err != nil {
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

	if featureconfig.Get().NewStateMgmt {
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

	preState := s.initSyncState[bytesutil.ToBytes32(b.ParentRoot)]
	var err error
	if preState == nil {
		if featureconfig.Get().CheckHeadState {
			headRoot, err := s.HeadRoot(ctx)
			if err != nil {
				return nil, errors.Wrapf(err, "could not get head root")
			}
			if bytes.Equal(headRoot, b.ParentRoot) {
				return s.HeadState(ctx)
			}
		}
		preState, err = s.beaconDB.State(ctx, bytesutil.ToBytes32(b.ParentRoot))
		if err != nil {
			return nil, errors.Wrapf(err, "could not get pre state for slot %d", b.Slot)
		}
		if preState == nil {
			if bytes.Equal(s.finalizedCheckpt.Root, b.ParentRoot) {
				return nil, fmt.Errorf("pre state of slot %d does not exist", b.Slot)
			}
			preState, err = s.generateState(ctx, bytesutil.ToBytes32(s.finalizedCheckpt.Root), bytesutil.ToBytes32(b.ParentRoot))
			if err != nil {
				return nil, err
			}
		}
		return preState, nil // No copy needed from newly hydrated DB object.
	}
	return preState.Copy(), nil
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

// rmStatesOlderThanLastFinalized deletes the states in db since last finalized check point.
func (s *Service) rmStatesOlderThanLastFinalized(ctx context.Context, startSlot uint64, endSlot uint64) error {
	ctx, span := trace.StartSpan(ctx, "forkchoice.rmStatesBySlots")
	defer span.End()

	// Make sure start slot is not a skipped slot
	for i := startSlot; i > 0; i-- {
		filter := filters.NewFilter().SetStartSlot(i).SetEndSlot(i)
		b, err := s.beaconDB.Blocks(ctx, filter)
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
		b, err := s.beaconDB.Blocks(ctx, filter)
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
	roots, err := s.beaconDB.BlockRoots(ctx, filter)
	if err != nil {
		return err
	}

	roots, err = s.filterBlockRoots(ctx, roots)
	if err != nil {
		return err
	}

	if err := s.beaconDB.DeleteStates(ctx, roots); err != nil {
		log.Warnf("Could not delete states: %v", err)
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
	justifiedRoot := bytesutil.ToBytes32(newJustifiedCheckpt.Root)
	var err error
	if !featureconfig.Get().NoInitSyncBatchSaveBlocks && s.hasInitSyncBlock(justifiedRoot) {
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
	cachedJustifiedRoot := bytesutil.ToBytes32(s.justifiedCheckpt.Root)
	if !featureconfig.Get().NoInitSyncBatchSaveBlocks && s.hasInitSyncBlock(cachedJustifiedRoot) {
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
	b, err := s.ancestor(ctx, newJustifiedCheckpt.Root, justifiedBlock.Slot)
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
	}

	if !featureconfig.Get().NewStateMgmt {
		justifiedRoot := bytesutil.ToBytes32(cpt.Root)

		justifiedState := s.initSyncState[justifiedRoot]
		// If justified state is nil, resume back to normal syncing process and save
		// justified check point.
		var err error
		if justifiedState == nil {
			if s.beaconDB.HasState(ctx, justifiedRoot) {
				return s.beaconDB.SaveJustifiedCheckpoint(ctx, cpt)
			}
			justifiedState, err = s.generateState(ctx, bytesutil.ToBytes32(s.finalizedCheckpt.Root), justifiedRoot)
			if err != nil {
				log.Error(err)
				return s.beaconDB.SaveJustifiedCheckpoint(ctx, cpt)
			}
		}
		if err := s.beaconDB.SaveState(ctx, justifiedState, justifiedRoot); err != nil {
			return errors.Wrap(err, "could not save justified state")
		}
	}

	return s.beaconDB.SaveJustifiedCheckpoint(ctx, cpt)
}

// This saves every finalized state in DB during initial sync, needed as part of optimization to
// use cache state during initial sync in case of restart.
func (s *Service) saveInitState(ctx context.Context, state *stateTrie.BeaconState) error {
	cpt := state.FinalizedCheckpoint()
	finalizedRoot := bytesutil.ToBytes32(cpt.Root)
	fs := s.initSyncState[finalizedRoot]
	if fs == nil {
		var err error
		fs, err = s.beaconDB.State(ctx, finalizedRoot)
		if err != nil {
			return err
		}
		if fs == nil {
			fs, err = s.generateState(ctx, bytesutil.ToBytes32(s.prevFinalizedCheckpt.Root), finalizedRoot)
			if err != nil {
				// This might happen if the client was in sync and is now re-syncing for whatever reason.
				log.Warn("Initial sync cache did not have finalized state root cached")
				return err
			}
		}
	}

	if err := s.beaconDB.SaveState(ctx, fs, finalizedRoot); err != nil {
		return errors.Wrap(err, "could not save state")
	}
	return nil
}

// This filters block roots that are not known as head root and finalized root in DB.
// It serves as the last line of defence before we prune states.
func (s *Service) filterBlockRoots(ctx context.Context, roots [][32]byte) ([][32]byte, error) {
	f, err := s.beaconDB.FinalizedCheckpoint(ctx)
	if err != nil {
		return nil, err
	}
	fRoot := f.Root
	h, err := s.beaconDB.HeadBlock(ctx)
	if err != nil {
		return nil, err
	}
	hRoot, err := stateutil.BlockRoot(h.Block)
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

// ancestor returns the block root of an ancestry block from the input block root.
//
// Spec pseudocode definition:
//   def get_ancestor(store: Store, root: Hash, slot: Slot) -> Hash:
//    block = store.blocks[root]
//    if block.slot > slot:
//      return get_ancestor(store, block.parent_root, slot)
//    elif block.slot == slot:
//      return root
//    else:
//      return Bytes32()  # root is older than queried slot: no results.
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

	if !featureconfig.Get().NoInitSyncBatchSaveBlocks && s.hasInitSyncBlock(bytesutil.ToBytes32(root)) {
		signed = s.getInitSyncBlock(bytesutil.ToBytes32(root))
	}

	if signed == nil || signed.Block == nil {
		return nil, errors.New("nil block")
	}
	b := signed.Block

	// If we dont have the ancestor in the DB, simply return nil so rest of fork choice
	// operation can proceed. This is not an error condition.
	if b == nil || b.Slot < slot {
		return nil, nil
	}

	if b.Slot == slot {
		return root, nil
	}

	return s.ancestor(ctx, b.ParentRoot, slot)
}

// This updates justified check point in store, if the new justified is later than stored justified or
// the store's justified is not in chain with finalized check point.
//
// Spec definition:
//   if (
//            state.current_justified_checkpoint.epoch > store.justified_checkpoint.epoch
//            or get_ancestor(store, store.justified_checkpoint.root, finalized_slot) != store.finalized_checkpoint.root
//        ):
//            store.justified_checkpoint = state.current_justified_checkpoint
func (s *Service) finalizedImpliesNewJustified(ctx context.Context, state *stateTrie.BeaconState) error {
	finalizedBlkSigned, err := s.beaconDB.Block(ctx, bytesutil.ToBytes32(s.finalizedCheckpt.Root))
	if err != nil || finalizedBlkSigned == nil || finalizedBlkSigned.Block == nil {
		return errors.Wrap(err, "could not get finalized block")
	}
	finalizedBlk := finalizedBlkSigned.Block

	anc, err := s.ancestor(ctx, s.justifiedCheckpt.Root, finalizedBlk.Slot)
	if err != nil {
		return err
	}

	// Either the new justified is later than stored justified or not in chain with finalized check pint.
	if cpt := state.CurrentJustifiedCheckpoint(); cpt != nil && cpt.Epoch > s.justifiedCheckpt.Epoch || !bytes.Equal(anc, s.finalizedCheckpt.Root) {
		s.justifiedCheckpt = state.CurrentJustifiedCheckpoint()
	}

	return nil
}

// This retrieves missing blocks from DB (ie. the blocks that couldn't received over sync) and inserts them to fork choice store.
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
		r, err := ssz.HashTreeRoot(b)
		if err != nil {
			return err
		}

		if err := s.forkChoiceStore.ProcessBlock(ctx,
			b.Slot, r, bytesutil.ToBytes32(b.ParentRoot),
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
