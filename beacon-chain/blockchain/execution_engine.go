package blockchain

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/time/slots"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var (
	ErrInvalidPayload                = errors.New("recevied an INVALID payload from execution engine")
	ErrUndefinedExecutionEngineError = errors.New("received an undefined ee error")
)

// notifyForkchoiceUpdateArg is the argument for the forkchoice update notification `notifyForkchoiceUpdate`.
type notifyForkchoiceUpdateArg struct {
	headState state.BeaconState
	headRoot  [32]byte
	headBlock interfaces.BeaconBlock
}

// notifyForkchoiceUpdate signals execution engine the fork choice updates. Execution engine should:
// 1. Re-organizes the execution payload chain and corresponding state to make head_block_hash the head.
// 2. Applies finality to the execution state: it irreversibly persists the chain of all execution payloads and corresponding state, up to and including finalized_block_hash.
func (s *Service) notifyForkchoiceUpdate(ctx context.Context, arg *notifyForkchoiceUpdateArg) (*enginev1.PayloadIDBytes, error) {
	ctx, span := trace.StartSpan(ctx, "blockChain.notifyForkchoiceUpdate")
	defer span.End()

	headBlk := arg.headBlock
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
	finalizedHash := s.store.FinalizedPayloadBlockHash()
	justifiedHash := s.store.JustifiedPayloadBlockHash()
	fcs := &enginev1.ForkchoiceState{
		HeadBlockHash:      headPayload.BlockHash,
		SafeBlockHash:      justifiedHash[:],
		FinalizedBlockHash: finalizedHash[:],
	}

	nextSlot := s.CurrentSlot() + 1 // Cache payload ID for next slot proposer.
	hasAttr, attr, proposerId, err := s.getPayloadAttribute(ctx, arg.headState, nextSlot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get payload attribute")
	}

	payloadID, lastValidHash, err := s.cfg.ExecutionEngineCaller.ForkchoiceUpdated(ctx, fcs, attr)
	if err != nil {
		switch err {
		case powchain.ErrAcceptedSyncingPayloadStatus:
			forkchoiceUpdatedOptimisticNodeCount.Inc()
			log.WithFields(logrus.Fields{
				"headSlot":                  headBlk.Slot(),
				"headPayloadBlockHash":      fmt.Sprintf("%#x", bytesutil.Trunc(headPayload.BlockHash)),
				"finalizedPayloadBlockHash": fmt.Sprintf("%#x", bytesutil.Trunc(finalizedHash[:])),
			}).Info("Called fork choice updated with optimistic block")
			return payloadID, s.optimisticCandidateBlock(ctx, headBlk)
		case powchain.ErrInvalidPayloadStatus:
			newPayloadInvalidNodeCount.Inc()
			headRoot := arg.headRoot
			invalidRoots, err := s.ForkChoicer().SetOptimisticToInvalid(ctx, headRoot, bytesutil.ToBytes32(headBlk.ParentRoot()), bytesutil.ToBytes32(lastValidHash))
			if err != nil {
				return nil, err
			}
			if err := s.removeInvalidBlockAndState(ctx, invalidRoots); err != nil {
				return nil, err
			}

			r, err := s.updateHead(ctx, s.justifiedBalances.balances)
			if err != nil {
				return nil, err
			}
			b, err := s.getBlock(ctx, r)
			if err != nil {
				return nil, err
			}
			st, err := s.cfg.StateGen.StateByRoot(ctx, r)
			if err != nil {
				return nil, err
			}
			pid, err := s.notifyForkchoiceUpdate(ctx, &notifyForkchoiceUpdateArg{
				headState: st,
				headRoot:  r,
				headBlock: b.Block(),
			})
			if err != nil {
				return nil, err
			}

			log.WithFields(logrus.Fields{
				"slot":         headBlk.Slot(),
				"blockRoot":    fmt.Sprintf("%#x", headRoot),
				"invalidCount": len(invalidRoots),
			}).Warn("Pruned invalid blocks")
			return pid, ErrInvalidPayload

		default:
			return nil, errors.WithMessage(ErrUndefinedExecutionEngineError, err.Error())
		}
	}
	forkchoiceUpdatedValidNodeCount.Inc()
	if err := s.cfg.ForkChoiceStore.SetOptimisticToValid(ctx, arg.headRoot); err != nil {
		return nil, errors.Wrap(err, "could not set block to valid")
	}
	if hasAttr { // If the forkchoice update call has an attribute, update the proposer payload ID cache.
		var pId [8]byte
		copy(pId[:], payloadID[:])
		s.cfg.ProposerSlotIndexCache.SetProposerAndPayloadIDs(nextSlot, proposerId, pId)
	}
	return payloadID, nil
}

// getPayloadHash returns the payload hash given the block root.
// if the block is before bellatrix fork epoch, it returns the zero hash.
func (s *Service) getPayloadHash(ctx context.Context, root []byte) ([32]byte, error) {
	blk, err := s.getBlock(ctx, s.ensureRootNotZeros(bytesutil.ToBytes32(root)))
	if err != nil {
		return [32]byte{}, err
	}
	if blocks.IsPreBellatrixVersion(blk.Block().Version()) {
		return params.BeaconConfig().ZeroHash, nil
	}
	payload, err := blk.Block().Body().ExecutionPayload()
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not get execution payload")
	}
	return bytesutil.ToBytes32(payload.BlockHash), nil
}

// notifyForkchoiceUpdate signals execution engine on a new payload.
// It returns true if the EL has returned VALID for the block
func (s *Service) notifyNewPayload(ctx context.Context, postStateVersion int,
	postStateHeader *ethpb.ExecutionPayloadHeader, blk interfaces.SignedBeaconBlock) (bool, error) {
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
	switch err {
	case nil:
		newPayloadValidNodeCount.Inc()
		return true, nil
	case powchain.ErrAcceptedSyncingPayloadStatus:
		newPayloadOptimisticNodeCount.Inc()
		log.WithFields(logrus.Fields{
			"slot":             blk.Block().Slot(),
			"payloadBlockHash": fmt.Sprintf("%#x", bytesutil.Trunc(payload.BlockHash)),
		}).Info("Called new payload with optimistic block")
		return false, s.optimisticCandidateBlock(ctx, blk.Block())
	case powchain.ErrInvalidPayloadStatus:
		newPayloadInvalidNodeCount.Inc()
		root, err := blk.Block().HashTreeRoot()
		if err != nil {
			return false, err
		}
		invalidRoots, err := s.ForkChoicer().SetOptimisticToInvalid(ctx, root, bytesutil.ToBytes32(blk.Block().ParentRoot()), bytesutil.ToBytes32(lastValidHash))
		if err != nil {
			return false, err
		}
		if err := s.removeInvalidBlockAndState(ctx, invalidRoots); err != nil {
			return false, err
		}
		log.WithFields(logrus.Fields{
			"slot":         blk.Block().Slot(),
			"blockRoot":    fmt.Sprintf("%#x", root),
			"invalidCount": len(invalidRoots),
		}).Warn("Pruned invalid blocks")
		return false, ErrInvalidPayload
	default:
		return false, errors.WithMessage(ErrUndefinedExecutionEngineError, err.Error())
	}
}

// optimisticCandidateBlock returns an error if this block can't be optimistically synced.
// It replaces boolean in spec code with `errNotOptimisticCandidate`.
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
func (s *Service) optimisticCandidateBlock(ctx context.Context, blk interfaces.BeaconBlock) error {
	if blk.Slot()+params.BeaconConfig().SafeSlotsToImportOptimistically <= s.CurrentSlot() {
		return nil
	}
	parent, err := s.getBlock(ctx, bytesutil.ToBytes32(blk.ParentRoot()))
	if err != nil {
		return err
	}
	parentIsExecutionBlock, err := blocks.IsExecutionBlock(parent.Block().Body())
	if err != nil {
		return err
	}
	if parentIsExecutionBlock {
		return nil
	}

	return errNotOptimisticCandidate
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
			}).Warn("Fee recipient is currently using the burn address, " +
				"you will not be rewarded transaction fees on this setting. " +
				"Please set a different eth address as the fee recipient. " +
				"Please refer to our documentation for instructions")
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
