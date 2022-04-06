package blockchain

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/time/slots"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// notifyForkchoiceUpdate signals execution engine the fork choice updates. Execution engine should:
// 1. Re-organizes the execution payload chain and corresponding state to make head_block_hash the head.
// 2. Applies finality to the execution state: it irreversibly persists the chain of all execution payloads and corresponding state, up to and including finalized_block_hash.
func (s *Service) notifyForkchoiceUpdate(ctx context.Context, headState state.BeaconState, headBlk block.BeaconBlock, headRoot [32]byte, finalizedRoot [32]byte) (*enginev1.PayloadIDBytes, error) {
	ctx, span := trace.StartSpan(ctx, "blockChain.notifyForkchoiceUpdate")
	defer span.End()

	if headBlk == nil || headBlk.IsNil() || headBlk.Body().IsNil() {
		return nil, errors.New("nil head block")
	}
	// Must not call fork choice updated until the transition conditions are met on the Pow network.
	isExecutionBlk, err := blocks.IsExecutionBlock(headBlk.Body())
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
	if finalizedBlock == nil || finalizedBlock.IsNil() {
		finalizedBlock = s.getInitSyncBlock(s.ensureRootNotZeros(finalizedRoot))
		if finalizedBlock == nil || finalizedBlock.IsNil() {
			return nil, errors.Errorf("finalized block with root %#x does not exist in the db or our cache", s.ensureRootNotZeros(finalizedRoot))
		}
	}
	var finalizedHash []byte
	if blocks.IsPreBellatrixVersion(finalizedBlock.Block().Version()) {
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

	nextSlot := s.CurrentSlot() + 1 // Cache payload ID for next slot proposer.
	hasAttr, attr, proposerId, err := s.getPayloadAttribute(ctx, headState, nextSlot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get payload attribute")
	}

	payloadID, _, err := s.cfg.ExecutionEngineCaller.ForkchoiceUpdated(ctx, fcs, attr)
	if err != nil {
		switch err {
		case powchain.ErrAcceptedSyncingPayloadStatus:
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
	if hasAttr { // If the forkchoice update call has an attribute, update the proposer payload ID cache.
		var pId [8]byte
		copy(pId[:], payloadID[:])
		s.cfg.ProposerSlotIndexCache.SetProposerAndPayloadIDs(nextSlot, proposerId, pId)
	}
	return payloadID, nil
}

// notifyForkchoiceUpdate signals execution engine on a new payload.
// It returns true if the EL has returned VALID for the block
func (s *Service) notifyNewPayload(ctx context.Context, preStateVersion, postStateVersion int,
	preStateHeader, postStateHeader *ethpb.ExecutionPayloadHeader, blk block.SignedBeaconBlock) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "blockChain.notifyNewPayload")
	defer span.End()

	// Execution payload is only supported in Bellatrix and beyond. Pre
	// merge blocks are never optimistic
	if blocks.IsPreBellatrixVersion(postStateVersion) {
		return true, nil
	}
	if err := helpers.BeaconBlockIsNil(blk); err != nil {
		return false, err
	}
	body := blk.Block().Body()
	enabled, err := blocks.IsExecutionEnabledUsingHeader(postStateHeader, body)
	if err != nil {
		return false, errors.Wrap(err, "could not determine if execution is enabled")
	}
	if !enabled {
		return true, nil
	}
	payload, err := body.ExecutionPayload()
	if err != nil {
		return false, errors.Wrap(err, "could not get execution payload")
	}
	lastValidHash, err := s.cfg.ExecutionEngineCaller.NewPayload(ctx, payload)
	if err != nil {
		switch err {
		case powchain.ErrAcceptedSyncingPayloadStatus:
			log.WithFields(logrus.Fields{
				"slot":      blk.Block().Slot(),
				"blockHash": fmt.Sprintf("%#x", bytesutil.Trunc(payload.BlockHash)),
			}).Info("Called new payload with optimistic block")
			return false, nil
		case powchain.ErrInvalidPayloadStatus:
			root, err := blk.Block().HashTreeRoot()
			if err != nil {
				return false, err
			}
			invalidRoots, err := s.ForkChoicer().SetOptimisticToInvalid(ctx, root, bytesutil.ToBytes32(lastValidHash))
			if err != nil {
				return false, err
			}
			if err := s.removeInvalidBlockAndState(ctx, invalidRoots); err != nil {
				return false, err
			}
			return false, errors.New("could not validate an INVALID payload from execution engine")
		default:
			return false, errors.Wrap(err, "could not validate execution payload from execution engine")
		}
	}

	// During the transition event, the transition block should be verified for sanity.
	if blocks.IsPreBellatrixVersion(preStateVersion) {
		// Handle case where pre-state is Altair but block contains payload.
		// To reach here, the block must have contained a valid payload.
		return true, s.validateMergeBlock(ctx, blk)
	}
	atTransition, err := blocks.IsMergeTransitionBlockUsingPreStatePayloadHeader(preStateHeader, body)
	if err != nil {
		return true, errors.Wrap(err, "could not check if merge block is terminal")
	}
	if !atTransition {
		return true, nil
	}
	return true, s.validateMergeBlock(ctx, blk)
}

// optimisticCandidateBlock returns true if this block can be optimistically synced.
//
// Spec pseudocode definition:
// def is_optimistic_candidate_block(opt_store: OptimisticStore, current_slot: Slot, block: BeaconBlock) -> bool:
//    if is_execution_block(opt_store.blocks[block.parent_root]):
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

	parentIsExecutionBlock, err := blocks.IsExecutionBlock(parent.Block().Body())
	if err != nil {
		return false, err
	}
	return parentIsExecutionBlock, nil
}

// getPayloadAttributes returns the payload attributes for the given state and slot.
// The attribute is required to initiate a payload build process in the context of an `engine_forkchoiceUpdated` call.
func (s *Service) getPayloadAttribute(ctx context.Context, st state.BeaconState, slot types.Slot) (bool, *enginev1.PayloadAttributes, types.ValidatorIndex, error) {
	proposerID, _, ok := s.cfg.ProposerSlotIndexCache.GetProposerPayloadIDs(slot)
	if !ok { // There's no need to build attribute if there is no proposer for slot.
		return false, nil, 0, nil
	}

	// Get previous randao.
	st = st.Copy()
	st, err := transition.ProcessSlotsIfPossible(ctx, st, slot)
	if err != nil {
		return false, nil, 0, err
	}
	prevRando, err := helpers.RandaoMix(st, time.CurrentEpoch(st))
	if err != nil {
		return false, nil, 0, nil
	}

	// Get fee recipient.
	feeRecipient := params.BeaconConfig().DefaultFeeRecipient
	recipient, err := s.cfg.BeaconDB.FeeRecipientByValidatorID(ctx, proposerID)
	switch {
	case errors.Is(err, kv.ErrNotFoundFeeRecipient):
		if feeRecipient.String() == fieldparams.EthBurnAddressHex {
			logrus.WithFields(logrus.Fields{
				"validatorIndex": proposerID,
				"burnAddress":    fieldparams.EthBurnAddressHex,
			}).Error("Fee recipient not set. Using burn address")
		}
	case err != nil:
		return false, nil, 0, errors.Wrap(err, "could not get fee recipient in db")
	default:
		feeRecipient = recipient
	}

	// Get timestamp.
	t, err := slots.ToTime(uint64(s.genesisTime.Unix()), slot)
	if err != nil {
		return false, nil, 0, err
	}
	attr := &enginev1.PayloadAttributes{
		Timestamp:             uint64(t.Unix()),
		PrevRandao:            prevRando,
		SuggestedFeeRecipient: feeRecipient.Bytes(),
	}
	return true, attr, proposerID, nil
}

// removeInvalidBlockAndState removes the invalid block and its corresponding state from the cache and DB.
func (s *Service) removeInvalidBlockAndState(ctx context.Context, blkRoots [][32]byte) error {
	for _, root := range blkRoots {
		if err := s.cfg.StateGen.DeleteStateFromCaches(ctx, root); err != nil {
			return err
		}

		// Delete block also deletes the state as well.
		if err := s.cfg.BeaconDB.DeleteBlock(ctx, root); err != nil {
			// TODO(10487): If a caller requests to delete a root that's justified and finalized. We should gracefully shutdown.
			// This is an irreparable condition, it would me a justified or finalized block has become invalid.
			return err
		}
	}
	return nil
}
