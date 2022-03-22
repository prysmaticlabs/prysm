package blockchain

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/powchain/engine-api-client/v1"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/sirupsen/logrus"
)

// notifyForkchoiceUpdate signals execution engine the fork choice updates. Execution engine should:
// 1. Re-organizes the execution payload chain and corresponding state to make head_block_hash the head.
// 2. Applies finality to the execution state: it irreversibly persists the chain of all execution payloads and corresponding state, up to and including finalized_block_hash.
func (s *Service) notifyForkchoiceUpdate(ctx context.Context, headBlk block.BeaconBlock, headRoot [32]byte, finalizedRoot [32]byte) (*enginev1.PayloadIDBytes, error) {
	if headBlk == nil || headBlk.IsNil() || headBlk.Body().IsNil() {
		return nil, errors.New("nil head block")
	}
	// Must not call fork choice updated until the transition conditions are met on the Pow network.
	if isPreBellatrix(headBlk.Version()) {
		return nil, nil
	}
	isExecutionBlk, err := blocks.ExecutionBlock(headBlk.Body())
	if err != nil {
		return nil, errors.Wrap(err, "could not determine if block is execution block")
	}
	if !isExecutionBlk {
		return nil, nil
	}
	headPayload, err := headBlk.Body().ExecutionPayload()
	if err != nil {
		return nil, errors.Wrap(err, "could not get execution payload")
	}
	finalizedBlock, err := s.cfg.BeaconDB.Block(ctx, s.ensureRootNotZeros(finalizedRoot))
	if err != nil {
		return nil, errors.Wrap(err, "could not get finalized block")
	}
	var finalizedHash []byte
	if isPreBellatrix(finalizedBlock.Block().Version()) {
		finalizedHash = params.BeaconConfig().ZeroHash[:]
	} else {
		payload, err := finalizedBlock.Block().Body().ExecutionPayload()
		if err != nil {
			return nil, errors.Wrap(err, "could not get finalized block execution payload")
		}
		finalizedHash = payload.BlockHash
	}

	fcs := &enginev1.ForkchoiceState{
		HeadBlockHash:      headPayload.BlockHash,
		SafeBlockHash:      headPayload.BlockHash,
		FinalizedBlockHash: finalizedHash,
	}

	// payload attribute is only required when requesting payload, here we are just updating fork choice, so it is nil.
	payloadID, _, err := s.cfg.ExecutionEngineCaller.ForkchoiceUpdated(ctx, fcs, nil /*payload attribute*/)
	if err != nil {
		switch err {
		case v1.ErrAcceptedSyncingPayloadStatus:
			log.WithFields(logrus.Fields{
				"headSlot":      headBlk.Slot(),
				"headHash":      fmt.Sprintf("%#x", bytesutil.Trunc(headPayload.BlockHash)),
				"finalizedHash": fmt.Sprintf("%#x", bytesutil.Trunc(finalizedHash)),
			}).Info("Called fork choice updated with optimistic block")
			return payloadID, nil
		default:
			return nil, errors.Wrap(err, "could not notify forkchoice update from execution engine")
		}
	}
	if err := s.cfg.ForkChoiceStore.SetOptimisticToValid(ctx, headRoot); err != nil {
		return nil, errors.Wrap(err, "could not set block to valid")
	}
	return payloadID, nil
}

// notifyForkchoiceUpdate signals execution engine on a new payload
func (s *Service) notifyNewPayload(ctx context.Context, preStateVersion int, header *ethpb.ExecutionPayloadHeader, postState state.BeaconState, blk block.SignedBeaconBlock, root [32]byte) error {
	if postState == nil {
		return errors.New("pre and post states must not be nil")
	}
	// Execution payload is only supported in Bellatrix and beyond. Pre
	// merge blocks are never optimistic
	if isPreBellatrix(postState.Version()) {
		return s.cfg.ForkChoiceStore.SetOptimisticToValid(ctx, root)
	}
	if err := helpers.BeaconBlockIsNil(blk); err != nil {
		return err
	}
	body := blk.Block().Body()
	enabled, err := blocks.ExecutionEnabled(postState, blk.Block().Body())
	if err != nil {
		return errors.Wrap(err, "could not determine if execution is enabled")
	}
	if !enabled {
		return s.cfg.ForkChoiceStore.SetOptimisticToValid(ctx, root)
	}
	payload, err := body.ExecutionPayload()
	if err != nil {
		return errors.Wrap(err, "could not get execution payload")
	}
	_, err = s.cfg.ExecutionEngineCaller.NewPayload(ctx, payload)
	if err != nil {
		switch err {
		case v1.ErrAcceptedSyncingPayloadStatus:
			log.WithFields(logrus.Fields{
				"slot":      postState.Slot(),
				"blockHash": fmt.Sprintf("%#x", bytesutil.Trunc(payload.BlockHash)),
			}).Info("Called new payload with optimistic block")
			return nil
		case v1.ErrInvalidPayloadStatus:
			invalidRoots, err := s.ForkChoicer().SetOptimisticToInvalid(ctx, root)
			if err != nil {
				return err
			}
			return s.removeInvalidBlockAndState(ctx, invalidRoots)
		default:
			return errors.Wrap(err, "could not validate execution payload from execution engine")
		}
	}

	if err := s.cfg.ForkChoiceStore.SetOptimisticToValid(ctx, root); err != nil {
		return errors.Wrap(err, "could not set optimistic status")
	}

	// During the transition event, the transition block should be verified for sanity.
	if isPreBellatrix(preStateVersion) {
		// Handle case where pre-state is Altair but block contains payload.
		// To reach here, the block must have contained a valid payload.
		return s.validateMergeBlock(ctx, blk)
	}
	atTransition, err := blocks.IsMergeTransitionBlockUsingPayloadHeader(header, body)
	if err != nil {
		return errors.Wrap(err, "could not check if merge block is terminal")
	}
	if !atTransition {
		return nil
	}
	return s.validateMergeBlock(ctx, blk)
}

// isPreBellatrix returns true if input version is before bellatrix fork.
func isPreBellatrix(v int) bool {
	return v == version.Phase0 || v == version.Altair
}

// optimisticCandidateBlock returns true if this block can be optimistically synced.
//
// Spec pseudocode definition:
// def is_optimistic_candidate_block(opt_store: OptimisticStore, current_slot: Slot, block: BeaconBlock) -> bool:
//    if is_execution_block(opt_store.blocks[block.parent_root]):
//        return True
//
//    justified_root = opt_store.block_states[opt_store.head_block_root].current_justified_checkpoint.root
//    if is_execution_block(opt_store.blocks[justified_root]):
//        return True
//
//    if block.slot + SAFE_SLOTS_TO_IMPORT_OPTIMISTICALLY <= current_slot:
//        return True
//
//    return False
func (s *Service) optimisticCandidateBlock(ctx context.Context, blk block.BeaconBlock) (bool, error) {
	if blk.Slot()+params.BeaconConfig().SafeSlotsToImportOptimistically <= s.CurrentSlot() {
		return true, nil
	}

	parent, err := s.cfg.BeaconDB.Block(ctx, bytesutil.ToBytes32(blk.ParentRoot()))
	if err != nil {
		return false, err
	}
	if parent == nil {
		return false, errNilParentInDB
	}

	parentIsExecutionBlock, err := blocks.ExecutionBlock(parent.Block().Body())
	if err != nil {
		return false, err
	}
	if parentIsExecutionBlock {
		return true, nil
	}

	j := s.store.JustifiedCheckpt()
	if j == nil {
		return false, errNilJustifiedInStore
	}
	jBlock, err := s.cfg.BeaconDB.Block(ctx, bytesutil.ToBytes32(j.Root))
	if err != nil {
		return false, err
	}
	return blocks.ExecutionBlock(jBlock.Block().Body())
}

// removeInvalidBlockAndState removes the invalid block and its corresponding state from the cache and DB.
func (s *Service) removeInvalidBlockAndState(ctx context.Context, blkRoots [][32]byte) error {
	for _, root := range blkRoots {
		if err := s.cfg.StateGen.DeleteStateFromCaches(ctx, root); err != nil {
			return err
		}

		// Delete block also deletes the state as well.
		if err := s.cfg.BeaconDB.DeleteBlock(ctx, root); err != nil {
			if err == kv.ErrDeleteJustifiedAndFinalized {
				log.Panic("Invalid justified / finalized block in DB. Please resync from last weak subjectivity checkpoint")
			}
			return err
		}
	}
	return nil
}
