package helpers

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// ExecutionPayloadProtobuf converts eth1 execution payload from JSON format
// to Prysm's protobuf format.
func ExecutionPayloadToProtobuf(payload *catalyst.ExecutableData) *ethpb.ExecutionPayload {
	txs := make([]*ethpb.OpaqueTransaction, len(payload.Transactions))
	for i := range txs {
		// Double check this. It may not be right.
		txs[i] = &ethpb.OpaqueTransaction{Data: payload.Transactions[i]}
	}
	return &ethpb.ExecutionPayload{
		BlockHash:    bytesutil.PadTo(payload.BlockHash.Bytes(), 32),
		ParentHash:   bytesutil.PadTo(payload.ParentHash.Bytes(), 32),
		Coinbase:     bytesutil.PadTo(payload.Miner.Bytes(), 20),
		StateRoot:    bytesutil.PadTo(payload.StateRoot.Bytes(), 32),
		Number:       payload.Number,
		GasLimit:     payload.GasLimit,
		GasUsed:      payload.GasUsed,
		Timestamp:    payload.Timestamp,
		ReceiptRoot:  bytesutil.PadTo(payload.ReceiptRoot.Bytes(), 32),
		LogsBloom:    bytesutil.PadTo(payload.LogsBloom, 256),
		Transactions: txs,
	}
}

// ExecPayloadToJson converts eth1 execution payload from Prysm's protobuf format to JSON format.
func ExecPayloadToJson(payload *ethpb.ExecutionPayload) catalyst.ExecutableData {
	txs := make([][]byte, len(payload.Transactions))
	for i := range txs {
		// Double check this. It may not be right.
		txs[i] = payload.Transactions[i].Data
	}

	return catalyst.ExecutableData{
		BlockHash:    common.BytesToHash(payload.BlockHash),
		ParentHash:   common.BytesToHash(payload.ParentHash),
		Miner:        common.BytesToAddress(payload.Coinbase),
		StateRoot:    common.BytesToHash(payload.StateRoot),
		Number:       payload.Number,
		GasLimit:     payload.GasLimit,
		GasUsed:      payload.GasUsed,
		Timestamp:    payload.Timestamp,
		ReceiptRoot:  common.BytesToHash(payload.ReceiptRoot),
		LogsBloom:    payload.LogsBloom,
		Transactions: txs,
	}
}
