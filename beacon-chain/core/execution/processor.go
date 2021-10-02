package execution

import (
	"bytes"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/encoding/ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/time/slots"
)

// IsMergeComplete returns true if merge has been completed.
//
// Spec code:
// def is_merge_complete(state: BeaconState) -> bool:
//    return state.latest_execution_payload_header != ExecutionPayloadHeader()
func IsMergeComplete(st state.BeaconState) (bool, error) {
	h, err := st.LatestExecutionPayloadHeader()
	if err != nil {
		return false, err
	}
	return !ssz.DeepEqual(h, emptyPayload()), nil
}

// IsMergeBlock returns true if input block is the merge block.
//
// Spec code:
// def is_merge_block(state: BeaconState, body: BeaconBlockBody) -> bool:
//    return not is_merge_complete(state) and body.execution_payload != ExecutionPayload()
func IsMergeBlock(st state.BeaconState, blk block.BeaconBlockBody) (bool, error) {
	mergeComplete, err := IsMergeComplete(st)
	if err != nil {
		return false, err
	}
	if mergeComplete {
		return false, err
	}

	payload, err := blk.ExecutionPayload()
	if err != nil {
		return false, err
	}
	return !ssz.DeepEqual(payload, emptyPayloadHeader()), nil
}

// IsExecutionEnabled returns true if the execution is enabled.
//
// Spec code:
// def is_execution_enabled(state: BeaconState, body: BeaconBlockBody) -> bool:
//    return is_merge_block(state, body) or is_merge_complete(state)
func IsExecutionEnabled(st state.BeaconState, blk block.BeaconBlockBody) (bool, error) {
	mergeBlock, err := IsMergeBlock(st, blk)
	if err != nil {
		return false, err
	}
	if mergeBlock {
		return true, nil
	}
	return IsMergeComplete(st)
}

// ProcessPayload processes execution payload.
//
// Spec code:
// def process_execution_payload(state: BeaconState, payload: ExecutionPayload, execution_engine: ExecutionEngine) -> None:
//    # Verify consistency of the parent hash, block number, base fee per gas and gas limit
//    # with respect to the previous execution payload header
//    if is_merge_complete(state):
//        assert payload.parent_hash == state.latest_execution_payload_header.block_hash
//        assert payload.block_number == state.latest_execution_payload_header.block_number + uint64(1)
//        assert is_valid_gas_limit(payload, state.latest_execution_payload_header)
//    # Verify random
//    assert payload.random == get_randao_mix(state, get_current_epoch(state))
//    # Verify timestamp
//    assert payload.timestamp == compute_timestamp_at_slot(state, state.slot)
//    # Verify the execution payload is valid
//    assert execution_engine.on_payload(payload)
//    # Cache execution payload header
//    state.latest_execution_payload_header = ExecutionPayloadHeader(
//        parent_hash=payload.parent_hash,
//        coinbase=payload.coinbase,
//        state_root=payload.state_root,
//        receipt_root=payload.receipt_root,
//        logs_bloom=payload.logs_bloom,
//        random=payload.random,
//        block_number=payload.block_number,
//        gas_limit=payload.gas_limit,
//        gas_used=payload.gas_used,
//        timestamp=payload.timestamp,
//        extra_data=payload.extra_data,
//        base_fee_per_gas=payload.base_fee_per_gas,
//        block_hash=payload.block_hash,
//        transactions_root=hash_tree_root(payload.transactions),
//    )
func ProcessPayload(st state.BeaconState, payload *ethpb.ExecutionPayload) (state.BeaconState, error) {
	if err := verifyPayload(st, payload); err != nil {
		return nil, err
	}
	random, err := helpers.RandaoMix(st, time.CurrentEpoch(st))
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(payload.Random, random) {
		return nil, errors.New("incorrect random")
	}
	t, err := slots.ToTime(st.GenesisTime(), st.Slot())
	if err != nil {
		return nil, err
	}
	if payload.Timestamp != uint64(t.Unix()) {
		return nil, errors.New("incorrect timestamp")
	}

	// TODO@: Verify the execution payload is valid

	header, err := payloadToHeader(payload)
	if err != nil {
		return nil, err
	}
	if err := st.SetLatestExecutionPayloadHeader(header); err != nil {
		return nil, err
	}
	return st, nil
}

// This verifies if `payload` is valid according to `state`. It exits early if merge is not yet ready.
func verifyPayload(st state.BeaconState, payload *ethpb.ExecutionPayload) error {
	complete, err := IsMergeComplete(st)
	if err != nil {
		return err
	}
	if !complete {
		return nil
	}

	header, err := st.LatestExecutionPayloadHeader()
	if err != nil {
		return err
	}
	if !bytes.Equal(payload.ParentHash, header.BlockHash) {
		return errors.New("incorrect block hash")
	}
	if payload.BlockNumber != header.BlockNumber+1 {
		return errors.New("incorrect block number")
	}
	if !isValidGasLimit(payload, header) {
		return errors.New("incorrect gas limit")
	}
	return nil
}

// This checks if gas limit and used in `payload` is valid to `parent.
func isValidGasLimit(payload *ethpb.ExecutionPayload, parent *ethpb.ExecutionPayloadHeader) bool {
	if payload.GasUsed > payload.GasLimit {
		return false
	}
	if payload.GasUsed < params.BeaconConfig().MinGasLimit {
		return false
	}

	parentGasLimit := parent.GasLimit
	if payload.GasLimit >= parentGasLimit+parentGasLimit/params.BeaconConfig().GasLimitDenominator {
		return false
	}
	if payload.GasLimit < parentGasLimit-parentGasLimit/params.BeaconConfig().GasLimitDenominator {
		return false
	}

	return true
}

// This converts `payload` to header format.
func payloadToHeader(payload *ethpb.ExecutionPayload) (*ethpb.ExecutionPayloadHeader, error) {
	/*
	txRoot, err := ssz.TransactionsRoot(payload.Transactions)
	if err != nil {
		return nil, err
	}
	 */
	var txRoot [32]byte
	empty32 := make([]byte, 32)
	copy(txRoot[:], empty32)

	return &ethpb.ExecutionPayloadHeader{
		ParentHash:       bytesutil.SafeCopyBytes(payload.ParentHash),
		Coinbase:         bytesutil.SafeCopyBytes(payload.Coinbase),
		StateRoot:        bytesutil.SafeCopyBytes(payload.StateRoot),
		ReceiptRoot:      bytesutil.SafeCopyBytes(payload.ReceiptRoot),
		LogsBloom:        bytesutil.SafeCopyBytes(payload.LogsBloom),
		Random:           bytesutil.SafeCopyBytes(payload.Random),
		BlockNumber:      payload.BlockNumber,
		GasLimit:         payload.GasLimit,
		GasUsed:          payload.GasUsed,
		Timestamp:        payload.Timestamp,
		ExtraData:        bytesutil.SafeCopyBytes(payload.ExtraData),
		BaseFeePerGas:    bytesutil.SafeCopyBytes(payload.BaseFeePerGas),
		BlockHash:        bytesutil.SafeCopyBytes(payload.BlockHash),
		TransactionsRoot: txRoot[:],
	}, nil
}

func emptyPayload() *ethpb.ExecutionPayload {
	return &ethpb.ExecutionPayload{
		ParentHash:    make([]byte, 32),
		Coinbase:      make([]byte, 20),
		StateRoot:     make([]byte, 32),
		ReceiptRoot:   make([]byte, 32),
		LogsBloom:     make([]byte, 256),
		Random:        make([]byte, 32),
		BlockNumber:   0,
		GasLimit:      0,
		GasUsed:       0,
		Timestamp:     0,
		ExtraData:     nil,
		BaseFeePerGas: make([]byte, 32),
		BlockHash:     make([]byte, 32),
		Transactions:  nil,
	}
}

func emptyPayloadHeader() *ethpb.ExecutionPayloadHeader {
	return &ethpb.ExecutionPayloadHeader{
		ParentHash:       make([]byte, 32),
		Coinbase:         make([]byte, 20),
		StateRoot:        make([]byte, 32),
		ReceiptRoot:      make([]byte, 32),
		LogsBloom:        make([]byte, 256),
		Random:           make([]byte, 32),
		BlockNumber:      0,
		GasLimit:         0,
		GasUsed:          0,
		Timestamp:        0,
		ExtraData:        nil,
		BaseFeePerGas:    make([]byte, 32),
		BlockHash:        make([]byte, 32),
		TransactionsRoot: make([]byte, 32),
	}
}
