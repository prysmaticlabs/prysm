package blocks

import (
	"bytes"

	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
)

// MergeComplete returns true if the transition to merge has completed.
// Meaning the header header in beacon state is not `ExecutionPayloadHeader()` (i.e. not empty).
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

func EmptyPayload() *ethpb.ExecutionPayload {
	return &ethpb.ExecutionPayload{
		ParentHash:    make([]byte, fieldparams.RootLength),
		FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
		StateRoot:     make([]byte, fieldparams.RootLength),
		ReceiptRoot:   make([]byte, fieldparams.RootLength),
		LogsBloom:     make([]byte, fieldparams.LogsBloomLength),
		Random:        make([]byte, fieldparams.RootLength),
		BaseFeePerGas: make([]byte, fieldparams.RootLength),
		BlockHash:     make([]byte, fieldparams.RootLength),
	}
}
