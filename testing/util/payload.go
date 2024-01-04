package util

import (
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/encoding/ssz"
	enginev1 "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
)

func DefaultPayload() *enginev1.ExecutionPayload {
	return &enginev1.ExecutionPayload{
		ParentHash:    params.BeaconConfig().ZeroHash[:],
		FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
		StateRoot:     make([]byte, fieldparams.RootLength),
		ReceiptsRoot:  make([]byte, fieldparams.RootLength),
		LogsBloom:     make([]byte, fieldparams.LogsBloomLength),
		PrevRandao:    make([]byte, fieldparams.RootLength),
		BaseFeePerGas: make([]byte, fieldparams.RootLength),
		BlockHash:     params.BeaconConfig().ZeroHash[:],
		Transactions:  make([][]byte, 0),
	}
}

func DefaultPayloadHeader() (*enginev1.ExecutionPayloadHeader, error) {
	txRoot, err := ssz.TransactionsRoot([][]byte{{1}})
	if err != nil {
		return nil, err
	}
	return &enginev1.ExecutionPayloadHeader{
		ParentHash:       params.BeaconConfig().ZeroHash[:],
		FeeRecipient:     make([]byte, fieldparams.FeeRecipientLength),
		StateRoot:        make([]byte, fieldparams.RootLength),
		ReceiptsRoot:     make([]byte, fieldparams.RootLength),
		LogsBloom:        make([]byte, fieldparams.LogsBloomLength),
		PrevRandao:       make([]byte, fieldparams.RootLength),
		BaseFeePerGas:    make([]byte, fieldparams.RootLength),
		BlockHash:        params.BeaconConfig().ZeroHash[:],
		TransactionsRoot: txRoot[:],
	}, nil
}

func DefaultPayloadCapella() *enginev1.ExecutionPayloadCapella {
	return &enginev1.ExecutionPayloadCapella{
		ParentHash:    params.BeaconConfig().ZeroHash[:],
		FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
		StateRoot:     make([]byte, fieldparams.RootLength),
		ReceiptsRoot:  make([]byte, fieldparams.RootLength),
		LogsBloom:     make([]byte, fieldparams.LogsBloomLength),
		PrevRandao:    make([]byte, fieldparams.RootLength),
		BaseFeePerGas: make([]byte, fieldparams.RootLength),
		BlockHash:     params.BeaconConfig().ZeroHash[:],
		Transactions:  make([][]byte, 0),
		Withdrawals:   make([]*enginev1.Withdrawal, 0),
	}
}

func DefaultPayloadHeaderCapella() (*enginev1.ExecutionPayloadHeaderCapella, error) {
	txRoot, err := ssz.TransactionsRoot([][]byte{{1}})
	if err != nil {
		return nil, err
	}
	wRoot, err := ssz.WithdrawalSliceRoot([]*enginev1.Withdrawal{}, fieldparams.MaxWithdrawalsPerPayload)
	if err != nil {
		return nil, err
	}
	return &enginev1.ExecutionPayloadHeaderCapella{
		ParentHash:       params.BeaconConfig().ZeroHash[:],
		FeeRecipient:     make([]byte, fieldparams.FeeRecipientLength),
		StateRoot:        make([]byte, fieldparams.RootLength),
		ReceiptsRoot:     make([]byte, fieldparams.RootLength),
		LogsBloom:        make([]byte, fieldparams.LogsBloomLength),
		PrevRandao:       make([]byte, fieldparams.RootLength),
		BaseFeePerGas:    make([]byte, fieldparams.RootLength),
		BlockHash:        params.BeaconConfig().ZeroHash[:],
		TransactionsRoot: txRoot[:],
		WithdrawalsRoot:  wRoot[:],
	}, nil
}

func DefaultPayloadDeneb() *enginev1.ExecutionPayloadDeneb {
	return &enginev1.ExecutionPayloadDeneb{
		ParentHash:    params.BeaconConfig().ZeroHash[:],
		FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
		StateRoot:     make([]byte, fieldparams.RootLength),
		ReceiptsRoot:  make([]byte, fieldparams.RootLength),
		LogsBloom:     make([]byte, fieldparams.LogsBloomLength),
		PrevRandao:    make([]byte, fieldparams.RootLength),
		BaseFeePerGas: make([]byte, fieldparams.RootLength),
		BlockHash:     params.BeaconConfig().ZeroHash[:],
		Transactions:  make([][]byte, 0),
		Withdrawals:   make([]*enginev1.Withdrawal, 0),
	}
}

func DefaultPayloadHeaderDeneb() (*enginev1.ExecutionPayloadHeaderDeneb, error) {
	txRoot, err := ssz.TransactionsRoot([][]byte{{1}})
	if err != nil {
		return nil, err
	}
	wRoot, err := ssz.WithdrawalSliceRoot([]*enginev1.Withdrawal{}, fieldparams.MaxWithdrawalsPerPayload)
	if err != nil {
		return nil, err
	}
	return &enginev1.ExecutionPayloadHeaderDeneb{
		ParentHash:       params.BeaconConfig().ZeroHash[:],
		FeeRecipient:     make([]byte, fieldparams.FeeRecipientLength),
		StateRoot:        make([]byte, fieldparams.RootLength),
		ReceiptsRoot:     make([]byte, fieldparams.RootLength),
		LogsBloom:        make([]byte, fieldparams.LogsBloomLength),
		PrevRandao:       make([]byte, fieldparams.RootLength),
		BaseFeePerGas:    make([]byte, fieldparams.RootLength),
		BlockHash:        params.BeaconConfig().ZeroHash[:],
		TransactionsRoot: txRoot[:],
		WithdrawalsRoot:  wRoot[:],
	}, nil
}
