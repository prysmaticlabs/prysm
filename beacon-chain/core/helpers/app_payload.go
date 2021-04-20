package helpers

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ExecutionPayloadProtobuf converts eth1 application payload from Geth's JSON format
// to Prysm's protobuf format.
func ExecutionPayloadProtobuf(payload *eth.ApplicationPayload) *ethpb.ApplicationPayload {
	txs := make([]*ethpb.OpaqueTransaction, len(payload.Transactions))
	for i := range txs {
		txs[i] = &ethpb.OpaqueTransaction{Data: payload.Transactions[i].Data()}
	}
	return &ethpb.ApplicationPayload{
		BlockHash:    bytesutil.PadTo(payload.BlockHash.Bytes(), 32),
		Coinbase:     bytesutil.PadTo(payload.Coinbase.Bytes(), 20),
		StateRoot:    bytesutil.PadTo(payload.StateRoot.Bytes(), 32),
		GasLimit:     payload.GasLimit,
		GasUsed:      payload.GasUsed,
		ReceiptRoot:  bytesutil.PadTo(payload.ReceiptRoot.Bytes(), 32),
		LogsBloom:    bytesutil.PadTo(payload.LogsBloom, 256),
		Transactions: txs,
	}
}

// AppPayloadJson converts eth1 application payload from Prysm's protobuf format to Geth's JSON format.
func AppPayloadJson(payload *ethpb.ApplicationPayload, parentHash []byte) eth.ApplicationPayload {
	return eth.ApplicationPayload{
		Coinbase:     common.BytesToAddress(payload.Coinbase),
		StateRoot:    common.BytesToHash(payload.StateRoot),
		GasLimit:     payload.GasLimit,
		GasUsed:      payload.GasUsed,
		Transactions: []*types.Transaction{},
		ReceiptRoot:  common.BytesToHash(payload.ReceiptRoot),
		LogsBloom:    payload.LogsBloom,
		BlockHash:    common.BytesToHash(payload.BlockHash),
		Difficulty:   big.NewInt(131072),
		ParentHash:   common.BytesToHash(parentHash),
	}
}

// ComputeRandaoMixWithReveal computes and returns the randao mix using input reveal.
func ComputeRandaoMixWithReveal(beaconState iface.ReadOnlyBeaconState, reveal []byte) ([]byte, error) {
	epoch := SlotToEpoch(beaconState.Slot())
	mix, err := beaconState.RandaoMixAtIndex(uint64(epoch % params.BeaconConfig().EpochsPerHistoricalVector))
	if err != nil {
		return nil, err
	}
	hashedReveal := hashutil.Hash(reveal)
	for i, x := range hashedReveal {
		mix[i] ^= x
	}
	return mix, nil
}
