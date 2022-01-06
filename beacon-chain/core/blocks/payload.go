package blocks

import (
	"bytes"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/encoding/ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/time/slots"
)

// MergeComplete returns true if the transition to merge has completed.
// Meaning the payload header in beacon state is not `ExecutionPayloadHeader()` (i.e. not empty).
//
// Spec code:
// def is_merge_complete(state: BeaconState) -> bool:
//    return state.latest_execution_payload_header != ExecutionPayloadHeader()
func MergeComplete(st state.BeaconState) (bool, error) {
	h, err := st.LatestExecutionPayloadHeader()
	if err != nil {
		return false, err
	}

	return !isEmptyHeader(h), nil
}

// IsMergeBlock returns true if the input block is the terminal merge block.
// Meaning the header in beacon state is  `ExecutionPayloadHeader()` (i.e. empty).
// And the input block has a non-empty header.
//
// Spec code:
// def is_merge_block(state: BeaconState, body: BeaconBlockBody) -> bool:
//    return not is_merge_complete(state) and body.execution_payload != ExecutionPayload()
func IsMergeBlock(st state.BeaconState, blk block.BeaconBlockBody) (bool, error) {
	mergeComplete, err := MergeComplete(st)
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
	return !isEmptyPayload(payload), nil
}

// ExecutionEnabled returns true if the beacon chain can begin executing.
// Meaning the payload header is beacon state is non-empty or the payload in block body is non-empty.
//
// Spec code:
// def is_execution_enabled(state: BeaconState, body: BeaconBlockBody) -> bool:
//    return is_merge_block(state, body) or is_merge_complete(state)
func ExecutionEnabled(st state.BeaconState, blk block.BeaconBlockBody) (bool, error) {
	mergeBlock, err := IsMergeBlock(st, blk)
	if err != nil {
		return false, err
	}
	if mergeBlock {
		return true, nil
	}
	return MergeComplete(st)
}

// ValidatePayloadWhenMergeCompletes validates if payload is valid versus input beacon state.
// These validation steps ONLY apply to post merge.
//
// Spec code:
//    # Verify consistency of the parent hash with respect to the previous execution payload header
//    if is_merge_complete(state):
//        assert payload.parent_hash == state.latest_execution_payload_header.block_hash
func ValidatePayloadWhenMergeCompletes(st state.BeaconState, payload *ethpb.ExecutionPayload) error {
	complete, err := MergeComplete(st)
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

// ValidatePayload validates if payload is valid versus input beacon state.
// These validation steps apply to both pre merge and post merge.
//
// Spec code:
//    # Verify random
//    assert payload.random == get_randao_mix(state, get_current_epoch(state))
//    # Verify timestamp
//    assert payload.timestamp == compute_timestamp_at_slot(state, state.slot)
func ValidatePayload(st state.BeaconState, payload *ethpb.ExecutionPayload) error {
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

// ProcessPayload processes input execution payload using beacon state.
// ValidatePayloadWhenMergeCompletes validates if payload is valid versus input beacon state.
// These validation steps ONLY apply to post merge.
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
	if err := ValidatePayloadWhenMergeCompletes(st, payload); err != nil {
		return nil, err
	}

	if err := ValidatePayload(st, payload); err != nil {
		return nil, err
	}

	header, err := PayloadToHeader(payload)
	if err != nil {
		return nil, err
	}
	if err := st.SetLatestExecutionPayloadHeader(header); err != nil {
		return nil, err
	}
	return st, nil
}

// PayloadToHeader converts `payload` into execution payload header format.
func PayloadToHeader(payload *ethpb.ExecutionPayload) (*ethpb.ExecutionPayloadHeader, error) {
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

func isEmptyPayload(p *ethpb.ExecutionPayload) bool {
	if !bytes.Equal(p.ParentHash, make([]byte, fieldparams.RootLength)) {
		return false
	}
	if !bytes.Equal(p.FeeRecipient, make([]byte, fieldparams.FeeRecipientLength)) {
		return false
	}
	if !bytes.Equal(p.StateRoot, make([]byte, fieldparams.RootLength)) {
		return false
	}
	if !bytes.Equal(p.ReceiptRoot, make([]byte, fieldparams.RootLength)) {
		return false
	}
	if !bytes.Equal(p.LogsBloom, make([]byte, fieldparams.LogsBloomLength)) {
		return false
	}
	if !bytes.Equal(p.Random, make([]byte, fieldparams.RootLength)) {
		return false
	}
	if !bytes.Equal(p.BaseFeePerGas, make([]byte, fieldparams.RootLength)) {
		return false
	}
	if !bytes.Equal(p.BlockHash, make([]byte, fieldparams.RootLength)) {
		return false
	}
	if len(p.Transactions) != 0 {
		return false
	}
	if len(p.ExtraData) != 0 {
		return false
	}
	if p.BlockNumber != 0 {
		return false
	}
	if p.GasLimit != 0 {
		return false
	}
	if p.GasUsed != 0 {
		return false
	}
	if p.Timestamp != 0 {
		return false
	}
	return true
}

func isEmptyHeader(h *ethpb.ExecutionPayloadHeader) bool {
	if !bytes.Equal(h.ParentHash, make([]byte, fieldparams.RootLength)) {
		return false
	}
	if !bytes.Equal(h.FeeRecipient, make([]byte, fieldparams.FeeRecipientLength)) {
		return false
	}
	if !bytes.Equal(h.StateRoot, make([]byte, fieldparams.RootLength)) {
		return false
	}
	if !bytes.Equal(h.ReceiptRoot, make([]byte, fieldparams.RootLength)) {
		return false
	}
	if !bytes.Equal(h.LogsBloom, make([]byte, fieldparams.LogsBloomLength)) {
		return false
	}
	if !bytes.Equal(h.Random, make([]byte, fieldparams.RootLength)) {
		return false
	}
	if !bytes.Equal(h.BaseFeePerGas, make([]byte, fieldparams.RootLength)) {
		return false
	}
	if !bytes.Equal(h.BlockHash, make([]byte, fieldparams.RootLength)) {
		return false
	}
	if !bytes.Equal(h.TransactionsRoot, make([]byte, fieldparams.RootLength)) {
		return false
	}
	if len(h.ExtraData) != 0 {
		return false
	}
	if h.BlockNumber != 0 {
		return false
	}
	if h.GasLimit != 0 {
		return false
	}
	if h.GasUsed != 0 {
		return false
	}
	if h.Timestamp != 0 {
		return false
	}
	return true
}
