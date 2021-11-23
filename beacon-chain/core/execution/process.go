package execution

import (
	"bytes"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/encoding/ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/time/slots"
)

// IsMergeComplete returns true if the transition merge has happened.
//
// Spec code:
// def is_merge_complete(state: BeaconState) -> bool:
//    return state.latest_execution_payload_header != ExecutionPayloadHeader()
func IsMergeComplete(st state.BeaconState) (bool, error) {
	h, err := st.LatestExecutionPayloadHeader()
	if err != nil {
		return false, err
	}
	// TODO_MERGE: Benchmark this for faster compare.
	return !ssz.DeepEqual(h, EmptypayloadHeader()), nil
}

// IsMergeBlock returns true if input block can become the merge block.
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
	// TODO_MERGE: Benchmark this for faster compare.
	return !ssz.DeepEqual(payload, EmptyPayload()), nil
}

// Enabled returns true if the beacon chain can begin executing.
//
// Spec code:
// def is_execution_enabled(state: BeaconState, body: BeaconBlockBody) -> bool:
//    return is_merge_block(state, body) or is_merge_complete(state)
func Enabled(st state.BeaconState, blk block.BeaconBlockBody) (bool, error) {
	mergeBlock, err := IsMergeBlock(st, blk)
	if err != nil {
		return false, err
	}
	if mergeBlock {
		return true, nil
	}
	return IsMergeComplete(st)
}

// ProcessPayload processes input execution payload using beacon state.
//
// Spec code:
// def process_execution_payload(state: BeaconState, payload: ExecutionPayload, execution_engine: ExecutionEngine) -> None:
//    # Verify consistency of the parent hash with respect to the previous execution payload header
//    if is_merge_complete(state):
//        assert payload.parent_hash == state.latest_execution_payload_header.block_hash
//    # Verify random
//    assert payload.random == get_randao_mix(state, get_current_epoch(state))
//    # Verify timestamp
//    assert payload.timestamp == compute_timestamp_at_slot(state, state.slot)
//    # Verify the execution payload is valid
//    assert execution_engine.execute_payload(payload)
//    # Cache execution payload header
//    state.latest_execution_payload_header = ExecutionPayloadHeader(
//        parent_hash=payload.parent_hash,
//        FeeRecipient=payload.FeeRecipient,
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
	if err := validatePayloadWhenMergeCompletes(st, payload); err != nil {
		return nil, err
	}

	if err := validatePayload(st, payload); err != nil {
		return nil, err
	}

	// This deviate with spec definition. It supposed to perform `execution_engine.on_payload(payload)` here.
	// Core pkg contains all pure functions. They don't have access to execution engine i.e. rpc service.
	// The soonest we can do this is after state transition.

	header, err := payloadToHeader(payload)
	if err != nil {
		return nil, err
	}
	if err := st.SetLatestExecutionPayloadHeader(header); err != nil {
		return nil, err
	}
	return st, nil
}

// This validates if payload is valid according to beacon state.
// These validation steps ONLY apply to post merge.
func validatePayloadWhenMergeCompletes(st state.BeaconState, payload *ethpb.ExecutionPayload) error {
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
	return nil
}

// This validates if payload is valid according to beacon state.
// These validation steps apply to both pre merge and post merge.
func validatePayload(st state.BeaconState, payload *ethpb.ExecutionPayload) error {
	random, err := helpers.RandaoMix(st, time.CurrentEpoch(st))
	if err != nil {
		return err
	}
	if !bytes.Equal(payload.Random, random) {
		return errors.New("incorrect random")
	}
	t, err := slots.ToTime(st.GenesisTime(), st.Slot())
	if err != nil {
		return err
	}
	if payload.Timestamp != uint64(t.Unix()) {
		return errors.New("incorrect timestamp")
	}
	return nil
}

// This converts `payload` into execution payload header format.
func payloadToHeader(payload *ethpb.ExecutionPayload) (*ethpb.ExecutionPayloadHeader, error) {
	txRoot, err := ssz.TransactionsRoot(payload.Transactions)
	if err != nil {
		return nil, err
	}

	return &ethpb.ExecutionPayloadHeader{
		ParentHash:       bytesutil.SafeCopyBytes(payload.ParentHash),
		FeeRecipient:     bytesutil.SafeCopyBytes(payload.FeeRecipient),
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

// EmptyPayload represents `ExecutionPayload()` in spec.
func EmptyPayload() *ethpb.ExecutionPayload {
	return &ethpb.ExecutionPayload{
		ParentHash:    make([]byte, 32),
		FeeRecipient:  make([]byte, 20),
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

// This represents `ExecutionPayloadHeader()` in spec.
func EmptypayloadHeader() *ethpb.ExecutionPayloadHeader {
	return &ethpb.ExecutionPayloadHeader{
		ParentHash:       make([]byte, 32),
		FeeRecipient:     make([]byte, 20),
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
