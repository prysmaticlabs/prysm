package blockchain

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// This defines size of the upper bound for initial sync block cache.
var initialSyncBlockCacheSize = 2 * params.BeaconConfig().SlotsPerEpoch

// onBlock is called when a gossip block is received. It runs regular state transition on the block.
// The block's signing root should be computed before calling this method to avoid redundant
// computation in this method and methods it calls into.
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
//    assert block.slot > compute_start_slot_at_epoch(store.finalized_checkpoint.epoch)
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
func (s *Service) onBlock(ctx context.Context, signed *ethpb.SignedBeaconBlock, blockRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.onBlock")
	defer span.End()

	if signed == nil || signed.Block == nil {
		return errors.New("nil block")
	}
	b := signed.Block

	preState, err := s.getBlockPreState(ctx, b)
	if err != nil {
		return err
	}

	postState, err := state.ExecuteStateTransition(ctx, preState, signed)
	if err != nil {
		return errors.Wrap(err, "could not execute state transition")
	}

	if err := s.beaconDB.SaveBlock(ctx, signed); err != nil {
		return errors.Wrapf(err, "could not save block from slot %d", b.Slot)
	}

	if err := s.insertBlockAndAttestationsToForkChoiceStore(ctx, b, blockRoot, postState); err != nil {
		return errors.Wrapf(err, "could not insert block %d to fork choice store", b.Slot)
	}

	if err := s.stateGen.SaveState(ctx, blockRoot, postState); err != nil {
		return errors.Wrap(err, "could not save state")
	}

	// Update justified check point.
	if postState.CurrentJustifiedCheckpoint().Epoch > s.justifiedCheckpt.Epoch {
		if err := s.updateJustified(ctx, postState); err != nil {
			return err
		}
	}

	// Update finalized check point.
	if postState.FinalizedCheckpointEpoch() > s.finalizedCheckpt.Epoch {
		if err := s.beaconDB.SaveBlocks(ctx, s.getInitSyncBlocks()); err != nil {
			return err
		}
		s.clearInitSyncBlocks()

		if err := s.updateFinalized(ctx, postState.FinalizedCheckpoint()); err != nil {
			return err
		}

		fRoot := bytesutil.ToBytes32(postState.FinalizedCheckpoint().Root)
		if err := s.forkChoiceStore.Prune(ctx, fRoot); err != nil {
			return errors.Wrap(err, "could not prune proto array fork choice nodes")
		}

		if err := s.finalizedImpliesNewJustified(ctx, postState); err != nil {
			return errors.Wrap(err, "could not save new justified")
		}

		if err := s.stateGen.MigrateToCold(ctx, fRoot); err != nil {
			return errors.Wrap(err, "could not migrate to cold")
		}

		// Update deposit cache.
		s.depositCache.InsertFinalizedDeposits(ctx, int64(postState.Eth1DepositIndex()))
	}

	// Epoch boundary bookkeeping such as logging epoch summaries.
	if postState.Slot() >= s.nextEpochBoundarySlot {
		logEpochData(postState)
		reportEpochMetrics(postState)

		// Update caches at epoch boundary slot.
		if err := helpers.UpdateCommitteeCache(postState, helpers.CurrentEpoch(postState)); err != nil {
			return err
		}
		if err := helpers.UpdateProposerIndicesInCache(postState, helpers.CurrentEpoch(postState)); err != nil {
			return err
		}

		s.nextEpochBoundarySlot = helpers.StartSlot(helpers.NextEpoch(postState))
	}

	defer reportAttestationInclusion(b)

	return nil
}

// onBlockInitialSyncStateTransition is called when an initial sync block is received.
// It runs state transition on the block and without any BLS verification. The excluded BLS verification
// includes attestation's aggregated signature. It also does not save attestations.
// The block's signing root should be computed before calling this method to avoid redundant
// computation in this method and methods it calls into.
func (s *Service) onBlockInitialSyncStateTransition(ctx context.Context, signed *ethpb.SignedBeaconBlock, blockRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.onBlock")
	defer span.End()

	if signed == nil || signed.Block == nil {
		return errors.New("nil block")
	}

	b := signed.Block

	// Retrieve incoming block's pre state.
	if err := s.verifyBlkPreState(ctx, b); err != nil {
		return err
	}

	preState, err := s.stateGen.StateByRootInitialSync(ctx, bytesutil.ToBytes32(signed.Block.ParentRoot))
	if err != nil {
		return err
	}
	if preState == nil {
		return fmt.Errorf("nil pre state for slot %d", b.Slot)
	}

	// To invalidate cache for parent root because pre state will get mutated.
	s.stateGen.DeleteHotStateInCache(bytesutil.ToBytes32(b.ParentRoot))

	// Exit early if the pre state slot is higher than incoming block's slot.
	if preState.Slot() >= signed.Block.Slot {
		return nil
	}

	var postState *stateTrie.BeaconState
	if featureconfig.Get().InitSyncNoVerify {
		postState, err = state.ExecuteStateTransitionNoVerifyAttSigs(ctx, preState, signed)
	} else {
		postState, err = state.ExecuteStateTransition(ctx, preState, signed)
	}
	if err != nil {
		return errors.Wrap(err, "could not execute state transition")
	}
	return s.handlePostStateInSync(ctx, signed, blockRoot, postState)
}

func (s *Service) onBlockBatch(ctx context.Context, blks []*ethpb.SignedBeaconBlock,
	blockRoots [][32]byte) (*stateTrie.BeaconState, []*ethpb.Checkpoint, []*ethpb.Checkpoint, error) {
	ctx, span := trace.StartSpan(ctx, "blockChain.onBlock")
	defer span.End()

	if len(blks) == 0 || len(blockRoots) == 0 {
		return nil, nil, nil, errors.New("no blocks provided")
	}
	if blks[0] == nil || blks[0].Block == nil {
		return nil, nil, nil, errors.New("nil block")
	}
	b := blks[0].Block

	// Retrieve incoming block's pre state.
	if err := s.verifyBlkPreState(ctx, b); err != nil {
		return nil, nil, nil, err
	}
	preState, err := s.stateGen.StateByRootInitialSync(ctx, bytesutil.ToBytes32(b.ParentRoot))
	if err != nil {
		return nil, nil, nil, err
	}
	if preState == nil {
		return nil, nil, nil, fmt.Errorf("nil pre state for slot %d", b.Slot)
	}

	jCheckpoints := make([]*ethpb.Checkpoint, len(blks))
	fCheckpoints := make([]*ethpb.Checkpoint, len(blks))
	sigSet := &bls.SignatureSet{
		Signatures: []bls.Signature{},
		PublicKeys: []bls.PublicKey{},
		Messages:   [][32]byte{},
	}
	set := new(bls.SignatureSet)
	for i, b := range blks {
		set, preState, err = state.ExecuteStateTransitionNoVerifyAnySig(ctx, preState, b)
		if err != nil {
			return nil, nil, nil, err
		}
		r, err := stateutil.BlockRoot(b.Block)
		if err != nil {
			return nil, nil, nil, err
		}
		if err := s.stateGen.SaveState(ctx, r, preState); err != nil {
			return nil, nil, nil, errors.Wrap(err, "could not save state")
		}
		jCheckpoints[i] = preState.CurrentJustifiedCheckpoint()
		fCheckpoints[i] = preState.FinalizedCheckpoint()
		sigSet.Join(set)
	}
	verify, err := bls.VerifyMultipleSignatures(sigSet.Signatures, sigSet.Messages, sigSet.PublicKeys)
	if err != nil {
		return nil, nil, nil, err
	}
	if !verify {
		return nil, nil, nil, errors.New("batch block signature verification failed")
	}
	return preState, fCheckpoints, jCheckpoints, nil
}

// handles the state post transition and saves the appropriate checkpoints and forkchoice
// data.
func (s *Service) handlePostStateInSync(ctx context.Context, signed *ethpb.SignedBeaconBlock,
	blockRoot [32]byte, postState *stateTrie.BeaconState) error {

	b := signed.Block

	s.saveInitSyncBlock(blockRoot, signed)

	if err := s.stateGen.SaveState(ctx, blockRoot, postState); err != nil {
		return errors.Wrap(err, "could not save state")
	}

	if err := s.insertBlockAndAttestationsToForkChoiceStore(ctx, b, blockRoot, postState); err != nil {
		return errors.Wrapf(err, "could not insert block %d to fork choice store", b.Slot)
	}

	// Rate limit how many blocks (2 epochs worth of blocks) a node keeps in the memory.
	if uint64(len(s.getInitSyncBlocks())) > initialSyncBlockCacheSize {
		if err := s.beaconDB.SaveBlocks(ctx, s.getInitSyncBlocks()); err != nil {
			return err
		}
		s.clearInitSyncBlocks()
	}

	// Update finalized check point. Prune the block cache and helper caches on every new finalized epoch.
	if postState.FinalizedCheckpointEpoch() > s.finalizedCheckpt.Epoch {
		if err := s.beaconDB.SaveBlocks(ctx, s.getInitSyncBlocks()); err != nil {
			return err
		}
		s.clearInitSyncBlocks()

		if err := s.updateFinalized(ctx, postState.FinalizedCheckpoint()); err != nil {
			return err
		}

		fRoot := bytesutil.ToBytes32(postState.FinalizedCheckpoint().Root)
		if err := s.stateGen.MigrateToCold(ctx, fRoot); err != nil {
			return errors.Wrap(err, "could not migrate to cold")
		}
	}

	return s.handleEpochBoundary(postState)
}

// handles a block after the block's batch has been verified, where we can save blocks
// their state summaries and split them off to relative hot/cold storage.
func (s *Service) handleBlockAfterBatchVerify(ctx context.Context, signed *ethpb.SignedBeaconBlock,
	blockRoot [32]byte, fCheckpoint *ethpb.Checkpoint, jCheckpoint *ethpb.Checkpoint) error {
	b := signed.Block

	s.saveInitSyncBlock(blockRoot, signed)
	if err := s.insertBlockToForkChoiceStore(ctx, b, blockRoot, fCheckpoint, jCheckpoint); err != nil {
		return err
	}
	s.stateGen.SaveStateSummary(ctx, signed, blockRoot)

	// Rate limit how many blocks (2 epochs worth of blocks) a node keeps in the memory.
	if uint64(len(s.getInitSyncBlocks())) > initialSyncBlockCacheSize {
		if err := s.beaconDB.SaveBlocks(ctx, s.getInitSyncBlocks()); err != nil {
			return err
		}
		s.clearInitSyncBlocks()
	}

	// Update finalized check point. Prune the block cache and helper caches on every new finalized epoch.
	if fCheckpoint.Epoch > s.finalizedCheckpt.Epoch {
		if err := s.beaconDB.SaveBlocks(ctx, s.getInitSyncBlocks()); err != nil {
			return err
		}
		s.clearInitSyncBlocks()

		if err := s.updateFinalized(ctx, fCheckpoint); err != nil {
			return err
		}

		fRoot := bytesutil.ToBytes32(fCheckpoint.Root)
		if err := s.stateGen.MigrateToCold(ctx, fRoot); err != nil {
			return errors.Wrap(err, "could not migrate to cold")
		}
	}
	return nil
}

// Epoch boundary bookkeeping such as logging epoch summaries.
func (s *Service) handleEpochBoundary(postState *stateTrie.BeaconState) error {
	if postState.Slot() >= s.nextEpochBoundarySlot {
		reportEpochMetrics(postState)
		s.nextEpochBoundarySlot = helpers.StartSlot(helpers.NextEpoch(postState))

		// Update committees cache at epoch boundary slot.
		if err := helpers.UpdateCommitteeCache(postState, helpers.CurrentEpoch(postState)); err != nil {
			return err
		}
		if err := helpers.UpdateProposerIndicesInCache(postState, helpers.CurrentEpoch(postState)); err != nil {
			return err
		}
	}
	return nil
}

// This feeds in the block and block's attestations to fork choice store. It's allows fork choice store
// to gain information on the most current chain.
func (s *Service) insertBlockAndAttestationsToForkChoiceStore(ctx context.Context, blk *ethpb.BeaconBlock, root [32]byte,
	state *stateTrie.BeaconState) error {
	fCheckpoint := state.FinalizedCheckpoint()
	jCheckpoint := state.CurrentJustifiedCheckpoint()
	if err := s.insertBlockToForkChoiceStore(ctx, blk, root, fCheckpoint, jCheckpoint); err != nil {
		return err
	}
	// Feed in block's attestations to fork choice store.
	for _, a := range blk.Body.Attestations {
		committee, err := helpers.BeaconCommitteeFromState(state, a.Data.Slot, a.Data.CommitteeIndex)
		if err != nil {
			return err
		}
		indices := attestationutil.AttestingIndices(a.AggregationBits, committee)
		s.forkChoiceStore.ProcessAttestation(ctx, indices, bytesutil.ToBytes32(a.Data.BeaconBlockRoot), a.Data.Target.Epoch)
	}
	return nil
}

func (s *Service) insertBlockToForkChoiceStore(ctx context.Context, blk *ethpb.BeaconBlock,
	root [32]byte, fCheckpoint *ethpb.Checkpoint, jCheckpoint *ethpb.Checkpoint) error {
	if err := s.fillInForkChoiceMissingBlocks(ctx, blk, fCheckpoint, jCheckpoint); err != nil {
		return err
	}
	// Feed in block to fork choice store.
	if err := s.forkChoiceStore.ProcessBlock(ctx,
		blk.Slot, root, bytesutil.ToBytes32(blk.ParentRoot), bytesutil.ToBytes32(blk.Body.Graffiti),
		jCheckpoint.Epoch,
		fCheckpoint.Epoch); err != nil {
		return errors.Wrap(err, "could not process block for proto array fork choice")
	}
	return nil
}
