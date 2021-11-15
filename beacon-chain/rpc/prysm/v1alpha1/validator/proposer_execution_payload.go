package validator

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/execution"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
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
func (vs *Server) getExecutionPayload(ctx context.Context, slot types.Slot) (*ethpb.ExecutionPayload, error) {
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
	complete, err := execution.IsMergeComplete(st)
	if err != nil {
		return nil, err
	}

	if !complete {
		if bytesutil.ToBytes32(params.BeaconConfig().TerminalBlockHash.Bytes()) != [32]byte{} {
			// `TERMINAL_BLOCK_HASH` is used as an override, the activation epoch must be reached.
			isActivationEpochReached := params.BeaconConfig().TerminalBlockHashActivationEpoch <= slots.ToEpoch(slot)
			if !isActivationEpochReached {
				return execution.EmptyPayload(), nil
			}
		}

		parentHash, hasTerminalBlock, err = vs.getTerminalBlockHash(ctx)
		if err != nil {
			return nil, err
		}
		if !hasTerminalBlock {
			// No terminal block signals this is pre merge, empty payload is used.
			return execution.EmptyPayload(), nil
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
	if finalizedBlock != nil && finalizedBlock.Version() == version.Merge {
		finalizedPayload, err := finalizedBlock.Block().Body().ExecutionPayload()
		if err != nil {
			return nil, err
		}
		finalizedBlockHash = finalizedPayload.BlockHash
	}

	f := catalyst.ForkchoiceStateV1{
		HeadBlockHash:      common.BytesToHash(parentHash),
		SafeBlockHash:      common.BytesToHash(parentHash),
		FinalizedBlockHash: common.BytesToHash(finalizedBlockHash),
	}
	p := catalyst.PayloadAttributesV1{
		Timestamp:    uint64(t.Unix()),
		Random:       common.BytesToHash(random),
		FeeRecipient: params.BeaconConfig().Coinbase,
	}
	id, err := vs.ExecutionEngineCaller.PreparePayload(ctx, f, p)
	if err != nil {
		return nil, errors.Wrap(err, "could not prepare payload")
	}
	data, err := vs.ExecutionEngineCaller.GetPayload(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "could not get payload")
	}

	return executableDataToExecutionPayload(data), nil
}

func executableDataToExecutionPayload(ed *catalyst.ExecutableDataV1) *ethpb.ExecutionPayload {
	return &ethpb.ExecutionPayload{
		ParentHash:    bytesutil.PadTo(ed.ParentHash.Bytes(), 32),
		Coinbase:      bytesutil.PadTo(ed.Coinbase.Bytes(), 20),
		StateRoot:     bytesutil.PadTo(ed.StateRoot.Bytes(), 32),
		ReceiptRoot:   bytesutil.PadTo(ed.ReceiptRoot.Bytes(), 32),
		LogsBloom:     bytesutil.PadTo(ed.LogsBloom, 256),
		Random:        bytesutil.PadTo(ed.Random.Bytes(), 32),
		BlockNumber:   ed.Number,
		GasLimit:      ed.GasLimit,
		GasUsed:       ed.GasUsed,
		Timestamp:     ed.Timestamp,
		ExtraData:     ed.ExtraData,
		BaseFeePerGas: bytesutil.PadTo(ed.BaseFeePerGas.Bytes(), 32),
		BlockHash:     bytesutil.PadTo(ed.BlockHash.Bytes(), 32),
		Transactions:  ed.Transactions,
	}
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
	blk, err := vs.ExecutionEngineCaller.LatestExecutionBlock()
	if err != nil {
		return nil, false, errors.Wrap(err, "could not get latest execution block")
	}
	parentBlk, err := vs.ExecutionEngineCaller.ExecutionBlockByHash(common.HexToHash(blk.ParentHash))
	if err != nil {
		return nil, false, errors.Wrap(err, "could not get parent execution block")
	}
	if parentBlk == nil {
		return nil, false, nil
	}

	terminalTotalDifficulty := new(big.Int)
	terminalTotalDifficulty.SetUint64(params.BeaconConfig().TerminalTotalDifficulty)

	currentTotalDifficulty := common.HexToHash(blk.TotalDifficulty).Big()
	parentTotalDifficulty := common.HexToHash(parentBlk.TotalDifficulty).Big()
	blkNumber := blk.Number
	// TODO_MERGE: This can theoretically loop indefinitely. More discussion: https://github.com/ethereum/consensus-specs/issues/2636
	logged := false
	for {
		blockReachedTTD := currentTotalDifficulty.Cmp(terminalTotalDifficulty) >= 0
		parentReachedTTD := terminalTotalDifficulty.Cmp(parentTotalDifficulty) >= 0

		if blockReachedTTD && parentReachedTTD {
			log.WithFields(logrus.Fields{
				"currentTotalDifficulty":  currentTotalDifficulty,
				"parentTotalDifficulty":   parentTotalDifficulty,
				"terminalTotalDifficulty": terminalTotalDifficulty,
				"terminalBlockHash":       fmt.Sprintf("%#x", common.HexToHash(blk.Hash)),
				"terminalBlockNumber":     blkNumber,
			}).Info("'Terminal difficulty reached")
			return common.HexToHash(blk.Hash).Bytes(), true, err
		} else {
			if !logged {
				log.WithFields(logrus.Fields{
					"currentTotalDifficulty":  currentTotalDifficulty,
					"parentTotalDifficulty":   parentTotalDifficulty,
					"terminalTotalDifficulty": terminalTotalDifficulty,
					"terminalBlockHash":       fmt.Sprintf("%#x", common.HexToHash(blk.Hash)),
					"terminalBlockNumber":     blkNumber,
				}).Info("Terminal difficulty NOT reached")
				logged = true
			}

			blk := parentBlk
			blkNumber = blk.Number
			// TODO_MERGE: Add pow block cache to avoid requesting seen block.

			parentBlk, err = vs.ExecutionEngineCaller.ExecutionBlockByHash(common.HexToHash(blk.ParentHash))
			if err != nil {
				return nil, false, err
			}
			if parentBlk == nil {
				return nil, false, nil
			}
			currentTotalDifficulty = common.HexToHash(blk.TotalDifficulty).Big()
			parentTotalDifficulty = common.HexToHash(parentBlk.TotalDifficulty).Big()
		}
	}
}
