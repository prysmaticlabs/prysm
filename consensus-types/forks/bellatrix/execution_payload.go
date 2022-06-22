package bellatrix

import (
	"bytes"

	field_params "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/encoding/ssz"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// PayloadToHeader converts `payload` into execution payload header format.
func PayloadToHeader(payload *enginev1.ExecutionPayload) (*eth.ExecutionPayloadHeader, error) {
	txRoot, err := ssz.TransactionsRoot(payload.Transactions)
	if err != nil {
		return nil, err
	}

	return &eth.ExecutionPayloadHeader{
		ParentHash:       bytesutil.SafeCopyBytes(payload.ParentHash),
		FeeRecipient:     bytesutil.SafeCopyBytes(payload.FeeRecipient),
		StateRoot:        bytesutil.SafeCopyBytes(payload.StateRoot),
		ReceiptsRoot:     bytesutil.SafeCopyBytes(payload.ReceiptsRoot),
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

func IsEmptyPayload(p *enginev1.ExecutionPayload) bool {
	if p == nil {
		return true
	}
	if !bytes.Equal(p.ParentHash, make([]byte, field_params.RootLength)) {
		return false
	}
	if !bytes.Equal(p.FeeRecipient, make([]byte, field_params.FeeRecipientLength)) {
		return false
	}
	if !bytes.Equal(p.StateRoot, make([]byte, field_params.RootLength)) {
		return false
	}
	if !bytes.Equal(p.ReceiptsRoot, make([]byte, field_params.RootLength)) {
		return false
	}
	if !bytes.Equal(p.LogsBloom, make([]byte, field_params.LogsBloomLength)) {
		return false
	}
	if !bytes.Equal(p.PrevRandao, make([]byte, field_params.RootLength)) {
		return false
	}
	if !bytes.Equal(p.BaseFeePerGas, make([]byte, field_params.RootLength)) {
		return false
	}
	if !bytes.Equal(p.BlockHash, make([]byte, field_params.RootLength)) {
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

func IsEmptyHeader(h *eth.ExecutionPayloadHeader) bool {
	if !bytes.Equal(h.ParentHash, make([]byte, field_params.RootLength)) {
		return false
	}
	if !bytes.Equal(h.FeeRecipient, make([]byte, field_params.FeeRecipientLength)) {
		return false
	}
	if !bytes.Equal(h.StateRoot, make([]byte, field_params.RootLength)) {
		return false
	}
	if !bytes.Equal(h.ReceiptsRoot, make([]byte, field_params.RootLength)) {
		return false
	}
	if !bytes.Equal(h.LogsBloom, make([]byte, field_params.LogsBloomLength)) {
		return false
	}
	if !bytes.Equal(h.PrevRandao, make([]byte, field_params.RootLength)) {
		return false
	}
	if !bytes.Equal(h.BaseFeePerGas, make([]byte, field_params.RootLength)) {
		return false
	}
	if !bytes.Equal(h.BlockHash, make([]byte, field_params.RootLength)) {
		return false
	}
	if !bytes.Equal(h.TransactionsRoot, make([]byte, field_params.RootLength)) {
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
