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
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/prysmaticlabs/prysm/time/slots"
)

// IsMergeTransitionComplete returns true if the transition to Bellatrix has completed.
// Meaning the payload header in beacon state is not `ExecutionPayloadHeader()` (i.e. not empty).
//
// Spec code:
// def is_merge_transition_complete(state: BeaconState) -> bool:
//    return state.latest_execution_payload_header != ExecutionPayloadHeader()
func IsMergeTransitionComplete(st state.BeaconState) (bool, error) {
	if st == nil {
		return false, errors.New("nil state")
	}
	if IsPreBellatrixVersion(st.Version()) {
		return false, nil
	}
	h, err := st.LatestExecutionPayloadHeader()
	if err != nil {
		return false, err
	}
	return !isEmptyHeader(h), nil
}

// IsMergeTransitionBlockUsingPreStatePayloadHeader returns true if the input block is the terminal merge block.
// Terminal merge block must be associated with an empty payload header.
// This assumes the header `h` is referenced as the parent state for block body `body.
func IsMergeTransitionBlockUsingPreStatePayloadHeader(h *ethpb.ExecutionPayloadHeader, body block.BeaconBlockBody) (bool, error) {
	if h == nil || body == nil {
		return false, errors.New("nil header or block body")
	}
	if !isEmptyHeader(h) {
		return false, nil
	}
	return IsExecutionBlock(body)
}

// IsExecutionBlock returns whether the block has a non-empty ExecutionPayload.
//
// Spec code:
// def is_execution_block(block: BeaconBlock) -> bool:
//     return block.body.execution_payload != ExecutionPayload()
func IsExecutionBlock(body block.BeaconBlockBody) (bool, error) {
	if body == nil {
		return false, errors.New("nil block body")
	}
	payload, err := body.ExecutionPayload()
	switch {
	case errors.Is(err, wrapper.ErrUnsupportedField):
		return false, nil
	case err != nil:
		return false, err
	default:
	}
	return !isEmptyPayload(payload), nil
}

// IsExecutionEnabled returns true if the beacon chain can begin executing.
// Meaning the payload header is beacon state is non-empty or the payload in block body is non-empty.
//
// Spec code:
// def is_execution_enabled(state: BeaconState, body: BeaconBlockBody) -> bool:
//    return is_merge_block(state, body) or is_merge_complete(state)
func IsExecutionEnabled(st state.BeaconState, body block.BeaconBlockBody) (bool, error) {
	if st == nil || body == nil {
		return false, errors.New("nil state or block body")
	}
	if IsPreBellatrixVersion(st.Version()) {
		return false, nil
	}
	header, err := st.LatestExecutionPayloadHeader()
	if err != nil {
		return false, err
	}
	return IsExecutionEnabledUsingHeader(header, body)
}

// IsExecutionEnabledUsingHeader returns true if the execution is enabled using post processed payload header and block body.
// This is an optimized version of IsExecutionEnabled where beacon state is not required as an argument.
func IsExecutionEnabledUsingHeader(header *ethpb.ExecutionPayloadHeader, body block.BeaconBlockBody) (bool, error) {
	if !isEmptyHeader(header) {
		return true, nil
	}
	return IsExecutionBlock(body)
}

// IsPreBellatrixVersion returns true if input version is before bellatrix fork.
func IsPreBellatrixVersion(v int) bool {
	return v < version.Bellatrix
}

// ValidatePayloadWhenMergeCompletes validates if payload is valid versus input beacon state.
// These validation steps ONLY apply to post merge.
//
// Spec code:
//    # Verify consistency of the parent hash with respect to the previous execution payload header
//    if is_merge_complete(state):
//        assert payload.parent_hash == state.latest_execution_payload_header.block_hash
func ValidatePayloadWhenMergeCompletes(st state.BeaconState, payload *enginev1.ExecutionPayload) error {
	complete, err := IsMergeTransitionComplete(st)
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
func ValidatePayload(st state.BeaconState, payload *enginev1.ExecutionPayload) error {
	random, err := helpers.RandaoMix(st, time.CurrentEpoch(st))
	if err != nil {
		return err
	}

	if !bytes.Equal(payload.PrevRandao, random) {
		return errors.New("incorrect prev randao")
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
func ProcessPayload(st state.BeaconState, payload *enginev1.ExecutionPayload) (state.BeaconState, error) {
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
func PayloadToHeader(payload *enginev1.ExecutionPayload) (*ethpb.ExecutionPayloadHeader, error) {
	txRoot, err := ssz.TransactionsRoot(payload.Transactions)
	if err != nil {
		return nil, err
	}

	return &ethpb.ExecutionPayloadHeader{
		ParentHash:       bytesutil.SafeCopyBytes(payload.ParentHash),
		FeeRecipient:     bytesutil.SafeCopyBytes(payload.FeeRecipient),
		StateRoot:        bytesutil.SafeCopyBytes(payload.StateRoot),
		ReceiptRoot:      bytesutil.SafeCopyBytes(payload.ReceiptsRoot),
		LogsBloom:        bytesutil.SafeCopyBytes(payload.LogsBloom),
		PrevRandao:       bytesutil.SafeCopyBytes(payload.PrevRandao),
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

func isEmptyPayload(p *enginev1.ExecutionPayload) bool {
	if p == nil {
		return true
	}
	if !bytes.Equal(p.ParentHash, make([]byte, fieldparams.RootLength)) {
		return false
	}
	if !bytes.Equal(p.FeeRecipient, make([]byte, fieldparams.FeeRecipientLength)) {
		return false
	}
	if !bytes.Equal(p.StateRoot, make([]byte, fieldparams.RootLength)) {
		return false
	}
	if !bytes.Equal(p.ReceiptsRoot, make([]byte, fieldparams.RootLength)) {
		return false
	}
	if !bytes.Equal(p.LogsBloom, make([]byte, fieldparams.LogsBloomLength)) {
		return false
	}
	if !bytes.Equal(p.PrevRandao, make([]byte, fieldparams.RootLength)) {
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
	if !bytes.Equal(h.PrevRandao, make([]byte, fieldparams.RootLength)) {
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
