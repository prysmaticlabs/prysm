package validator

import (
	"context"
	"fmt"
	"math/big"

	"github.com/holiman/uint256"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/prysmaticlabs/prysm/time/slots"
	"github.com/sirupsen/logrus"
)

// This returns the execution payload of a given slot. The function has full awareness of pre and post merge.
// Payload is computed given the respected time of merge.
//
// Spec code:
// def prepare_execution_payload(state: BeaconState,
//                              pow_chain: Dict[Hash32, PowBlock],
//                              finalized_block_hash: Hash32,
//                              fee_recipient: ExecutionAddress,
//                              execution_engine: ExecutionEngine) -> Optional[PayloadId]:
//    if not is_merge_complete(state):
//        is_terminal_block_hash_set = TERMINAL_BLOCK_HASH != Hash32()
//        is_activation_epoch_reached = get_current_epoch(state.slot) >= TERMINAL_BLOCK_HASH_ACTIVATION_EPOCH
//        if is_terminal_block_hash_set and not is_activation_epoch_reached:
//            # Terminal block hash is set but activation epoch is not yet reached, no prepare payload call is needed
//            return None
//
//        terminal_pow_block = get_terminal_pow_block(pow_chain)
//        if terminal_pow_block is None:
//            # Pre-merge, no prepare payload call is needed
//            return None
//        # Signify merge via producing on top of the terminal PoW block
//        parent_hash = terminal_pow_block.block_hash
//    else:
//        # Post-merge, normal payload
//        parent_hash = state.latest_execution_payload_header.block_hash
//
//    # Set the forkchoice head and initiate the payload build process
//    payload_attributes = PayloadAttributes(
//        timestamp=compute_timestamp_at_slot(state, state.slot),
//        random=get_randao_mix(state, get_current_epoch(state)),
//        fee_recipient=fee_recipient,
//    )
//    return execution_engine.notify_forkchoice_updated(parent_hash, finalized_block_hash, payload_attributes)
func (vs *Server) getExecutionPayload(
	ctx context.Context, slot types.Slot,
) (*enginev1.ExecutionPayload, error) {
	// TODO_MERGE: Reuse the same head state as in building phase0 block attestation.
	st, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, err
	}
	st, err = transition.ProcessSlots(ctx, st, slot)
	if err != nil {
		return nil, err
	}

	var parentHash []byte
	var hasTerminalBlock bool
	complete, err := blocks.MergeTransitionComplete(st)
	if err != nil {
		return nil, err
	}

	if !complete {
		if bytesutil.ToBytes32(params.BeaconConfig().TerminalBlockHash.Bytes()) != [32]byte{} {
			// `TERMINAL_BLOCK_HASH` is used as an override, the activation epoch must be reached.
			isActivationEpochReached := params.BeaconConfig().TerminalBlockHashActivationEpoch <= slots.ToEpoch(slot)
			if !isActivationEpochReached {
				return blocks.EmptyPayload(), nil
			}
		}

		parentHash, hasTerminalBlock, err = vs.getTerminalBlockHash(ctx)
		if err != nil {
			return nil, err
		}
		if !hasTerminalBlock {
			// No terminal block signals this is pre merge, empty payload is used.
			return blocks.EmptyPayload(), nil
		}
		// Terminal block found signals production on top of terminal PoW block.
	} else {
		// Post merge, normal payload is used.
		header, err := st.LatestExecutionPayloadHeader()
		if err != nil {
			return nil, err
		}
		parentHash = header.BlockHash
	}

	t, err := slots.ToTime(st.GenesisTime(), slot)
	if err != nil {
		return nil, err
	}
	random, err := helpers.RandaoMix(st, time.CurrentEpoch(st))
	if err != nil {
		return nil, err
	}

	finalizedBlock, err := vs.BeaconDB.Block(ctx, bytesutil.ToBytes32(st.FinalizedCheckpoint().Root))
	if err != nil {
		return nil, err
	}
	finalizedBlockHash := params.BeaconConfig().ZeroHash[:]
	if finalizedBlock != nil && finalizedBlock.Version() == version.Bellatrix {
		finalizedPayload, err := finalizedBlock.Block().Body().ExecutionPayload()
		if err != nil {
			return nil, err
		}
		finalizedBlockHash = finalizedPayload.BlockHash
	}

	f := &enginev1.ForkchoiceState{
		HeadBlockHash:      parentHash,
		SafeBlockHash:      parentHash,
		FinalizedBlockHash: finalizedBlockHash,
	}
	p := &enginev1.PayloadAttributes{
		Timestamp:             uint64(t.Unix()),
		Random:                random,
		SuggestedFeeRecipient: params.BeaconConfig().FeeRecipient.Bytes(),
	}
	res, err := vs.ExecutionEngineCaller.ForkchoiceUpdated(ctx, f, p)
	if err != nil {
		return nil, errors.Wrap(err, "could not prepare payload")
	}
	log.WithFields(logrus.Fields{
		"status:": res.Status.Status,
		"hash:":   fmt.Sprintf("%#x", f.HeadBlockHash),
	}).Info("Successfully called forkchoiceUpdated with attribute")

	if res == nil || res.PayloadId == nil {
		return nil, errors.New("forkchoice returned nil")
	}

	log.WithFields(logrus.Fields{
		"id":   fmt.Sprintf("%#x", &res.PayloadId),
		"slot": slot,
		"hash": fmt.Sprintf("%#x", parentHash),
	}).Info("Received payload ID")
	var id [8]byte
	copy(id[:], res.PayloadId[:])
	return vs.ExecutionEngineCaller.GetPayload(ctx, id)
}

// This returns the valid terminal block hash with an existence bool value.
//
// Spec code:
// def get_terminal_pow_block(pow_chain: Dict[Hash32, PowBlock]) -> Optional[PowBlock]:
//    if TERMINAL_BLOCK_HASH != Hash32():
//        # Terminal block hash override takes precedence over terminal total difficulty
//        if TERMINAL_BLOCK_HASH in pow_chain:
//            return pow_chain[TERMINAL_BLOCK_HASH]
//        else:
//            return None
//
//    return get_pow_block_at_terminal_total_difficulty(pow_chain)
func (vs *Server) getTerminalBlockHash(ctx context.Context) ([]byte, bool, error) {
	terminalBlockHash := params.BeaconConfig().TerminalBlockHash
	// Terminal block hash override takes precedence over terminal total difficult.
	if params.BeaconConfig().TerminalBlockHash != params.BeaconConfig().ZeroHash {
		e, _, err := vs.Eth1BlockFetcher.BlockExists(ctx, terminalBlockHash)
		if err != nil {
			return nil, false, err
		}
		if !e {
			return nil, false, nil
		}

		return terminalBlockHash.Bytes(), true, nil
	}

	return vs.getPowBlockHashAtTerminalTotalDifficulty(ctx)
}

// This returns the valid terminal block hash based on total difficulty.
//
// Spec code:
// def get_pow_block_at_terminal_total_difficulty(pow_chain: Dict[Hash32, PowBlock]) -> Optional[PowBlock]:
//    # `pow_chain` abstractly represents all blocks in the PoW chain
//    for block in pow_chain:
//        parent = pow_chain[block.parent_hash]
//        block_reached_ttd = block.total_difficulty >= TERMINAL_TOTAL_DIFFICULTY
//        parent_reached_ttd = parent.total_difficulty >= TERMINAL_TOTAL_DIFFICULTY
//        if block_reached_ttd and not parent_reached_ttd:
//            return block
//
//    return None
func (vs *Server) getPowBlockHashAtTerminalTotalDifficulty(ctx context.Context) ([]byte, bool, error) {
	ttd := new(big.Int)
	ttd.SetString(params.BeaconConfig().TerminalTotalDifficulty, 10)
	terminalTotalDifficulty, of := uint256.FromBig(ttd)
	if of {
		return nil, false, errors.New("could not convert terminal total difficulty to uint256")
	}

	blk, err := vs.ExecutionEngineCaller.LatestExecutionBlock(ctx)
	if err != nil {
		return nil, false, errors.Wrap(err, "could not get latest execution block")
	}
	log.WithFields(logrus.Fields{
		"number": blk.Number,
		"hash":   fmt.Sprintf("%#x", blk.Hash),
		"td":     blk.TotalDifficulty,
	}).Info("Retrieving latest execution block")

	for {
		currentTotalDifficulty := new(uint256.Int)
		currentTotalDifficulty.SetBytes(bytesutil.ReverseByteOrder(blk.TotalDifficulty))
		blockReachedTTD := currentTotalDifficulty.Cmp(terminalTotalDifficulty) >= 0
		parentHash := bytesutil.ToBytes32(blk.ParentHash)
		if len(blk.ParentHash) == 0 || parentHash == params.BeaconConfig().ZeroHash {
			return nil, false, nil
		}
		parentBlk, err := vs.ExecutionEngineCaller.ExecutionBlockByHash(ctx, parentHash)
		if err != nil {
			return nil, false, errors.Wrap(err, "could not get parent execution block")
		}
		log.WithFields(logrus.Fields{
			"number": parentBlk.Number,
			"hash":   fmt.Sprintf("%#x", parentBlk.Hash),
			"td":     parentBlk.TotalDifficulty,
		}).Info("Retrieving parent execution block")

		if blockReachedTTD {
			parentTotalDifficulty := new(uint256.Int)
			parentTotalDifficulty.SetBytes(bytesutil.ReverseByteOrder(parentBlk.TotalDifficulty))
			parentReachedTTD := parentTotalDifficulty.Cmp(terminalTotalDifficulty) >= 0
			if !parentReachedTTD {
				log.WithFields(logrus.Fields{
					"number":   blk.Number,
					"hash":     fmt.Sprintf("%#x", blk.Hash),
					"td":       blk.TotalDifficulty,
					"parentTd": parentBlk.TotalDifficulty,
					"ttd":      terminalTotalDifficulty,
				}).Info("Retrieved terminal block hash")
				return blk.Hash, true, nil
			}
		}
		blk = parentBlk
	}
}
