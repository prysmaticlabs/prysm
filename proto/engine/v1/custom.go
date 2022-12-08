package enginev1

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethrpc "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"strconv"
)

func NewExecutionPayloadHeaderFromJSON(header *ethrpc.ExecutionPayloadHeaderJson) (*ExecutionPayloadHeader,
	error) {
	blockNumber, err := strconv.ParseUint(header.BlockNumber, 10, 64)
	if err != nil {
		return nil, err
	}
	gasLimit, err := strconv.ParseUint(header.GasLimit, 10, 64)
	if err != nil {
		return nil, err
	}
	gasUsed, err := strconv.ParseUint(header.GasUsed, 10, 64)
	if err != nil {
		return nil, err
	}
	timestamp, err := strconv.ParseUint(header.TimeStamp, 10, 64)
	if err != nil {
		return nil, err
	}
	return &ExecutionPayloadHeader{
		ParentHash:       hexutil.MustDecode(header.ParentHash),
		FeeRecipient:     hexutil.MustDecode(header.FeeRecipient),
		StateRoot:        hexutil.MustDecode(header.StateRoot),
		ReceiptsRoot:     hexutil.MustDecode(header.ReceiptsRoot),
		LogsBloom:        hexutil.MustDecode(header.LogsBloom),
		PrevRandao:       hexutil.MustDecode(header.PrevRandao),
		BlockNumber:      blockNumber,
		GasLimit:         gasLimit,
		GasUsed:          gasUsed,
		Timestamp:        timestamp,
		ExtraData:        hexutil.MustDecode(header.ExtraData),
		BaseFeePerGas:    hexutil.MustDecode(header.BaseFeePerGas),
		BlockHash:        hexutil.MustDecode(header.BlockHash),
		TransactionsRoot: hexutil.MustDecode(header.TransactionsRoot),
	}, nil
}
