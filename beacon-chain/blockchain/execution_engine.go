package blockchain

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/db/kv"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/execution"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	consensusblocks "github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var defaultLatestValidHash = bytesutil.PadTo([]byte{0xff}, 32)
var errNoAttribute = errors.New("could not get payload attributes")

// notifyForkchoiceUpdateArg is the argument for the forkchoice update notification `notifyForkchoiceUpdate`.
type notifyForkchoiceUpdateArg struct {
	headState state.BeaconState
	headRoot  [32]byte
	headBlock interfaces.BeaconBlock
}

type callForkchoiceUpdatedReturn struct {
	hasAttr       bool
	proposerId    types.ValidatorIndex
	payloadID     *enginev1.PayloadIDBytes
	lastValidHash []byte
	err           error
}

// callFforkchoiceUpdatedV1 wraps a call to the engine methods `engine_forkchoiceUpdatedV1`
func (s *Service) callForkchoiceUpdatedV1(
	ctx context.Context,
	st state.BeaconState,
	nextSlot types.Slot,
	fcs *enginev1.ForkchoiceState) callForkchoiceUpdatedReturn {

	hasAttr, attr, proposerId, err := s.getPayloadAttribute(ctx, st, nextSlot)
	if err != nil {
		return callForkchoiceUpdatedReturn{false, 0, nil, nil, errNoAttribute}
	}

	payloadID, lastValidHash, err := s.cfg.ExecutionEngineCaller.ForkchoiceUpdated(ctx, fcs, attr)
	return callForkchoiceUpdatedReturn{
		hasAttr:       hasAttr,
		proposerId:    proposerId,
		payloadID:     payloadID,
		lastValidHash: lastValidHash,
		err:           err,
	}
}

// callFforkchoiceUpdatedV2 wraps a call to the engine methods `engine_forkchoiceUpdatedV2`
func (s *Service) callForkchoiceUpdatedV2(
	ctx context.Context,
	st state.BeaconState,
	nextSlot types.Slot,
	fcs *enginev1.ForkchoiceState) callForkchoiceUpdatedReturn {

	hasAttr, attr, proposerId, err := s.getPayloadAttributeV2(ctx, st, nextSlot)
	if err != nil {
		return callForkchoiceUpdatedReturn{false, 0, nil, nil, errNoAttribute}
	}

	payloadID, lastValidHash, err := s.cfg.ExecutionEngineCaller.ForkchoiceUpdatedV2(ctx, fcs, attr)
	return callForkchoiceUpdatedReturn{
		hasAttr:       hasAttr,
		proposerId:    proposerId,
		payloadID:     payloadID,
		lastValidHash: lastValidHash,
		err:           err,
	}
}

// notifyForkchoiceUpdate signals execution engine the fork choice updates. Execution engine should:
// 1. Re-organizes the execution payload chain and corresponding state to make head_block_hash the head.
// 2. Applies finality to the execution state: it irreversibly persists the chain of all execution payloads and corresponding state, up to and including finalized_block_hash.
func (s *Service) notifyForkchoiceUpdate(ctx context.Context, arg *notifyForkchoiceUpdateArg) (*enginev1.PayloadIDBytes, error) {
	ctx, span := trace.StartSpan(ctx, "blockChain.notifyForkchoiceUpdate")
	defer span.End()

	headBlk := arg.headBlock
	if headBlk == nil || headBlk.IsNil() || headBlk.Body().IsNil() {
		log.Error("Head block is nil")
		return nil, nil
	}
	// Must not call fork choice updated until the transition conditions are met on the Pow network.
	isExecutionBlk, err := blocks.IsExecutionBlock(headBlk.Body())
	if err != nil {
		log.WithError(err).Error("Could not determine if head block is execution block")
		return nil, nil
	}
	if !isExecutionBlk {
		return nil, nil
	}
	headPayload, err := headBlk.Body().Execution()
	if err != nil {
		log.WithError(err).Error("Could not get execution payload for head block")
		return nil, nil
	}
	finalizedHash := s.ForkChoicer().FinalizedPayloadBlockHash()
	justifiedHash := s.ForkChoicer().JustifiedPayloadBlockHash()
	fcs := &enginev1.ForkchoiceState{
		HeadBlockHash:      headPayload.BlockHash(),
		SafeBlockHash:      justifiedHash[:],
		FinalizedBlockHash: finalizedHash[:],
	}

	nextSlot := s.CurrentSlot() + 1 // Cache payload ID for next slot proposer.
	var fcuReturn callForkchoiceUpdatedReturn
	if slots.ToEpoch(nextSlot) >= params.BeaconConfig().CapellaForkEpoch {
		fcuReturn = s.callForkchoiceUpdatedV2(ctx, arg.headState, nextSlot, fcs)
	} else {
		fcuReturn = s.callForkchoiceUpdatedV1(ctx, arg.headState, nextSlot, fcs)
	}
	if fcuReturn.err != nil {
		switch fcuReturn.err {
		case errNoAttribute:
			return nil, errNoAttribute
		case execution.ErrAcceptedSyncingPayloadStatus:
			forkchoiceUpdatedOptimisticNodeCount.Inc()
			log.WithFields(logrus.Fields{
				"headSlot":                  headBlk.Slot(),
				"headPayloadBlockHash":      fmt.Sprintf("%#x", bytesutil.Trunc(headPayload.BlockHash())),
				"finalizedPayloadBlockHash": fmt.Sprintf("%#x", bytesutil.Trunc(finalizedHash[:])),
			}).Info("Called fork choice updated with optimistic block")
			return fcuReturn.payloadID, nil
		case execution.ErrInvalidPayloadStatus:
			forkchoiceUpdatedInvalidNodeCount.Inc()
			headRoot := arg.headRoot
			if len(fcuReturn.lastValidHash) == 0 {
				fcuReturn.lastValidHash = defaultLatestValidHash
			}
			invalidRoots, err := s.ForkChoicer().SetOptimisticToInvalid(ctx, headRoot, headBlk.ParentRoot(), bytesutil.ToBytes32(fcuReturn.lastValidHash))
			if err != nil {
				log.WithError(err).Error("Could not set head root to invalid")
				return nil, nil
			}
			if err := s.removeInvalidBlockAndState(ctx, invalidRoots); err != nil {
				log.WithError(err).Error("Could not remove invalid block and state")
				return nil, nil
			}

			r, err := s.cfg.ForkChoiceStore.Head(ctx, s.justifiedBalances.balances)
			if err != nil {
				log.WithFields(logrus.Fields{
					"slot":                 headBlk.Slot(),
					"blockRoot":            fmt.Sprintf("%#x", bytesutil.Trunc(headRoot[:])),
					"invalidChildrenCount": len(invalidRoots),
				}).Warn("Pruned invalid blocks, could not update head root")
				return nil, invalidBlock{error: ErrInvalidPayload, root: arg.headRoot, invalidAncestorRoots: invalidRoots}
			}
			b, err := s.getBlock(ctx, r)
			if err != nil {
				log.WithError(err).Error("Could not get head block")
				return nil, nil
			}
			st, err := s.cfg.StateGen.StateByRoot(ctx, r)
			if err != nil {
				log.WithError(err).Error("Could not get head state")
				return nil, nil
			}
			pid, err := s.notifyForkchoiceUpdate(ctx, &notifyForkchoiceUpdateArg{
				headState: st,
				headRoot:  r,
				headBlock: b.Block(),
			})
			if err != nil {
				return nil, err // Returning err because it's recursive here.
			}

			if err := s.saveHead(ctx, r, b, st); err != nil {
				log.WithError(err).Error("could not save head after pruning invalid blocks")
			}

			log.WithFields(logrus.Fields{
				"slot":                 headBlk.Slot(),
				"blockRoot":            fmt.Sprintf("%#x", bytesutil.Trunc(headRoot[:])),
				"invalidChildrenCount": len(invalidRoots),
				"newHeadRoot":          fmt.Sprintf("%#x", bytesutil.Trunc(r[:])),
			}).Warn("Pruned invalid blocks")
			return pid, invalidBlock{error: ErrInvalidPayload, root: arg.headRoot, invalidAncestorRoots: invalidRoots}

		default:
			log.WithError(err).Error(ErrUndefinedExecutionEngineError)
			return nil, nil
		}
	}
	forkchoiceUpdatedValidNodeCount.Inc()
	if err := s.cfg.ForkChoiceStore.SetOptimisticToValid(ctx, arg.headRoot); err != nil {
		log.WithError(err).Error("Could not set head root to valid")
		return nil, nil
	}
	// If the forkchoice update call has an attribute, update the proposer payload ID cache.
	if fcuReturn.hasAttr && fcuReturn.payloadID != nil {
		var pId [8]byte
		copy(pId[:], fcuReturn.payloadID[:])
		s.cfg.ProposerSlotIndexCache.SetProposerAndPayloadIDs(
			nextSlot,
			fcuReturn.proposerId,
			pId,
			arg.headRoot,
		)
	} else if fcuReturn.hasAttr && fcuReturn.payloadID == nil {
		log.WithFields(logrus.Fields{
			"blockHash": fmt.Sprintf("%#x", headPayload.BlockHash()),
			"slot":      headBlk.Slot(),
		}).Error("Received nil payload ID on VALID engine response")
	}
	return fcuReturn.payloadID, nil
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
	payload, err := blk.Block().Body().Execution()
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not get execution payload")
	}
	return bytesutil.ToBytes32(payload.BlockHash()), nil
}

// notifyNewPayload signals execution engine on a new payload.
// It returns true if the EL has returned VALID for the block
func (s *Service) notifyNewPayload(ctx context.Context, postStateVersion int,
	postStateHeader interfaces.ExecutionData, blk interfaces.SignedBeaconBlock) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "blockChain.notifyNewPayload")
	defer span.End()

	// Execution payload is only supported in Bellatrix and beyond. Pre
	// merge blocks are never optimistic
	if blocks.IsPreBellatrixVersion(postStateVersion) {
		return true, nil
	}
	if err := consensusblocks.BeaconBlockIsNil(blk); err != nil {
		return false, err
	}
	body := blk.Block().Body()
	enabled, err := blocks.IsExecutionEnabledUsingHeader(postStateHeader, body)
	if err != nil {
		return false, errors.Wrap(invalidBlock{error: err}, "could not determine if execution is enabled")
	}
	if !enabled {
		return true, nil
	}
	payload, err := body.Execution()
	if err != nil {
		return false, errors.Wrap(invalidBlock{error: err}, "could not get execution payload")
	}
	lastValidHash, err := s.cfg.ExecutionEngineCaller.NewPayload(ctx, payload)
	switch err {
	case nil:
		newPayloadValidNodeCount.Inc()
		return true, nil
	case execution.ErrAcceptedSyncingPayloadStatus:
		newPayloadOptimisticNodeCount.Inc()
		log.WithFields(logrus.Fields{
			"slot":             blk.Block().Slot(),
			"payloadBlockHash": fmt.Sprintf("%#x", bytesutil.Trunc(payload.BlockHash())),
		}).Info("Called new payload with optimistic block")
		return false, nil
	case execution.ErrInvalidPayloadStatus:
		newPayloadInvalidNodeCount.Inc()
		root, err := blk.Block().HashTreeRoot()
		if err != nil {
			return false, err
		}
		invalidRoots, err := s.ForkChoicer().SetOptimisticToInvalid(ctx, root, blk.Block().ParentRoot(), bytesutil.ToBytes32(lastValidHash))
		if err != nil {
			return false, err
		}
		if err := s.removeInvalidBlockAndState(ctx, invalidRoots); err != nil {
			return false, err
		}
		log.WithFields(logrus.Fields{
			"slot":                 blk.Block().Slot(),
			"blockRoot":            fmt.Sprintf("%#x", root),
			"invalidChildrenCount": len(invalidRoots),
		}).Warn("Pruned invalid blocks")
		return false, invalidBlock{
			invalidAncestorRoots: invalidRoots,
			error:                ErrInvalidPayload,
		}
	case execution.ErrInvalidBlockHashPayloadStatus:
		newPayloadInvalidNodeCount.Inc()
		return false, ErrInvalidBlockHashPayloadStatus
	default:
		return false, errors.WithMessage(ErrUndefinedExecutionEngineError, err.Error())
	}
}

// getPayloadAttributes returns the payload attributes for the given state and slot.
// The attribute is required to initiate a payload build process in the context of an `engine_forkchoiceUpdated` call.
func (s *Service) getPayloadAttribute(ctx context.Context, st state.BeaconState, slot types.Slot) (bool, *enginev1.PayloadAttributes, types.ValidatorIndex, error) {
	// Root is `[32]byte{}` since we are retrieving proposer ID of a given slot. During insertion at assignment the root was not known.
	proposerID, _, ok := s.cfg.ProposerSlotIndexCache.GetProposerPayloadIDs(slot, [32]byte{} /* root */)
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
		if feeRecipient.String() == params.BeaconConfig().EthBurnAddressHex {
			logrus.WithFields(logrus.Fields{
				"validatorIndex": proposerID,
				"burnAddress":    params.BeaconConfig().EthBurnAddressHex,
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

// getPayloadAttributesV2 returns the payload attributes for the given state and slot.
// The attribute is required to initiate a payload build process in the context of an `engine_forkchoiceUpdatedV2` call.
func (s *Service) getPayloadAttributeV2(
	ctx context.Context,
	st state.BeaconState,
	slot types.Slot) (bool, *enginev1.PayloadAttributesV2, types.ValidatorIndex, error) {
	// Root is `[32]byte{}` since we are retrieving proposer ID of a given slot.
	// During insertion at assignment the root was not known.
	proposerID, _, ok := s.cfg.ProposerSlotIndexCache.GetProposerPayloadIDs(slot, [32]byte{} /* root */)
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
		if feeRecipient.String() == params.BeaconConfig().EthBurnAddressHex {
			logrus.WithFields(logrus.Fields{
				"validatorIndex": proposerID,
				"burnAddress":    params.BeaconConfig().EthBurnAddressHex,
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
	withdrawals, err := st.ExpectedWithdrawals()
	if err != nil {
		return false, nil, 0, errors.Wrap(err, "could not get expected withdrawals")
	}
	attr := &enginev1.PayloadAttributesV2{
		Timestamp:             uint64(t.Unix()),
		PrevRandao:            prevRando,
		SuggestedFeeRecipient: feeRecipient.Bytes(),
		Withdrawals:           withdrawals,
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
