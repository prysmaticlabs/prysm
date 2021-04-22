package helpers

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/htrutils"
)

// ExecutionPayloadProtobuf converts eth1 execution payload from JSON format
// to Prysm's protobuf format.
func ExecutionPayloadToProtobuf(payload *catalyst.ExecutableData) *ethpb.ExecutionPayload {
	txs := make([][]byte, len(payload.Transactions))
	for i := range txs {
		// Double check this. It may not be right.
		txs[i] = payload.Transactions[i]
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
		txs[i] = payload.Transactions[i]
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

// ExecPayloadToHeader converts execution payload to header format.
func ExecPayloadToHeader(payload *ethpb.ExecutionPayload) (*pb.ExecutionPayloadHeader, error) {
	txRoot, err := htrutils.TransactionsRoot(payload.Transactions)
	if err != nil {
		return nil, err
	}

	return &pb.ExecutionPayloadHeader{
		BlockHash:        bytesutil.SafeCopyBytes(payload.BlockHash),
		ParentHash:       bytesutil.SafeCopyBytes(payload.ParentHash),
		Coinbase:         bytesutil.SafeCopyBytes(payload.Coinbase),
		StateRoot:        bytesutil.SafeCopyBytes(payload.StateRoot),
		Number:           payload.Number,
		GasLimit:         payload.GasLimit,
		GasUsed:          payload.GasUsed,
		Timestamp:        payload.Timestamp,
		ReceiptRoot:      bytesutil.SafeCopyBytes(payload.ReceiptRoot),
		LogsBloom:        bytesutil.SafeCopyBytes(payload.LogsBloom),
		TransactionsRoot: txRoot[:],
	}, nil
}
