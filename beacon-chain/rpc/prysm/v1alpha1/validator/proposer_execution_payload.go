package validator

import (
	"context"
	"math/big"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/execution"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/time/slots"
)

// This returns the execution payload of a given slot. The function has full awareness of pre and post merge.
// Payload is computed given the respected time of merge.
//
// Spec code:
// def prepare_execution_payload(state: BeaconState,
//                              pow_chain: Sequence[PowBlock],
//                              fee_recipient: ExecutionAddress,
//                              execution_engine: ExecutionEngine) -> Optional[PayloadId]:
//    if not is_merge_complete(state):
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
//    timestamp = compute_timestamp_at_slot(state, state.slot)
//    random = get_randao_mix(state, get_current_epoch(state))
//    return execution_engine.prepare_payload(parent_hash, timestamp, random, fee_recipient)
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
		parentHash, hasTerminalBlock, err = vs.getTerminalBlockHash(ctx, slot)
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
		parentHash = header.ParentHash
	}

	t, err := slots.ToTime(st.GenesisTime(), slot)
	if err != nil {
		return nil, err
	}
	random, err := helpers.RandaoMix(st, time.CurrentEpoch(st))
	if err != nil {
		return nil, err
	}
	id, err := vs.ExecutionEngineCaller.PreparePayload(ctx, parentHash, uint64(t.Unix()), random, params.BeaconConfig().FeeRecipient.Bytes())
	if err != nil {
		return nil, err
	}
	return vs.ExecutionEngineCaller.GetPayload(ctx, id)
}

// This returns the valid terminal block hash with an existence bool value.
//
// Spec code:
// def get_terminal_pow_block(pow_chain: Sequence[PowBlock]) -> Optional[PowBlock]:
//    if TERMINAL_BLOCK_HASH != Hash32():
//        # Terminal block hash override takes precedence over terminal total difficulty
//        pow_block_overrides = [block for block in pow_chain if block.block_hash == TERMINAL_BLOCK_HASH]
//        if not any(pow_block_overrides):
//            return None
//        return pow_block_overrides[0]
//
//    return get_pow_block_at_terminal_total_difficulty(pow_chain)
func (vs *Server) getTerminalBlockHash(ctx context.Context, slot types.Slot) ([]byte, bool, error) {
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
// def get_pow_block_at_terminal_total_difficulty(pow_chain: Sequence[PowBlock]) -> Optional[PowBlock]:
//    # `pow_chain` abstractly represents all blocks in the PoW chain
//    for block in pow_chain:
//        parent = get_pow_block(block.parent_hash)
//        block_reached_ttd = block.total_difficulty >= TERMINAL_TOTAL_DIFFICULTY
//        parent_reached_ttd = parent.total_difficulty >= TERMINAL_TOTAL_DIFFICULTY
//        if block_reached_ttd and not parent_reached_ttd:
//            return block
//
//    return None
func (vs *Server) getPowBlockHashAtTerminalTotalDifficulty(ctx context.Context) ([]byte, bool, error) {
	b, err := vs.BlockFetcher.BlockByNumber(ctx, nil /* latest block */)
	if err != nil {
		return nil, false, err
	}
	terminalTotalDifficulty := new(big.Int)
	terminalTotalDifficulty.SetBytes(params.BeaconConfig().TerminalTotalDifficulty)
	var terminalBlockHash []byte

	// TODO_MERGE: This can theoretically loop indefinitely. More discussion: https://github.com/ethereum/consensus-specs/issues/2636
	for {
		if b.TotalDifficulty().Cmp(terminalTotalDifficulty) >= 0 {
			terminalBlockHash = b.Hash().Bytes()
			// Prevent infinite loops.
			if b.ParentHash() == b.Hash() {
				return nil, false, errors.New("invalid block")
			}
			// TODO_MERGE: Add pow block cache to avoid requesting previous block.
			b, err = vs.BlockFetcher.BlockByHash(ctx, b.ParentHash())
			if err != nil {
				return nil, false, err
			}
		} else {
			return terminalBlockHash, true, err
		}
	}
}
