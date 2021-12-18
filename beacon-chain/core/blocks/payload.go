package blocks

import (
	"bytes"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
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
	if h.ExtraData != nil {
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
