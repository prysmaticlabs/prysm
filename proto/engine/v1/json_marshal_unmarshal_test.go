package enginev1_test

import (
	"encoding/json"
	"math/big"
	"testing"

	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestJsonMarshalUnmarshal(t *testing.T) {
	t.Run("payload attributes", func(t *testing.T) {
		random := bytesutil.PadTo([]byte("random"), fieldparams.RootLength)
		feeRecipient := bytesutil.PadTo([]byte("feeRecipient"), fieldparams.FeeRecipientLength)
		jsonPayload := &enginev1.PayloadAttributes{
			Timestamp:             1,
			Random:                random,
			SuggestedFeeRecipient: feeRecipient,
		}
		enc, err := json.Marshal(jsonPayload)
		require.NoError(t, err)
		payloadPb := &enginev1.PayloadAttributes{}
		require.NoError(t, json.Unmarshal(enc, payloadPb))
		require.DeepEqual(t, uint64(1), payloadPb.Timestamp)
		require.DeepEqual(t, random, payloadPb.Random)
		require.DeepEqual(t, feeRecipient, payloadPb.SuggestedFeeRecipient)
	})
	t.Run("payload status", func(t *testing.T) {
		hash := bytesutil.PadTo([]byte("hash"), fieldparams.RootLength)
		jsonPayload := &enginev1.PayloadStatus{
			Status:          enginev1.PayloadStatus_INVALID,
			LatestValidHash: hash,
			ValidationError: "failed validation",
		}
		enc, err := json.Marshal(jsonPayload)
		require.NoError(t, err)
		payloadPb := &enginev1.PayloadStatus{}
		require.NoError(t, json.Unmarshal(enc, payloadPb))
		require.DeepEqual(t, "INVALID", payloadPb.Status.String())
		require.DeepEqual(t, hash, payloadPb.LatestValidHash)
		require.DeepEqual(t, "failed validation", payloadPb.ValidationError)
	})
	t.Run("forkchoice state", func(t *testing.T) {
		head := bytesutil.PadTo([]byte("head"), fieldparams.RootLength)
		safe := bytesutil.PadTo([]byte("safe"), fieldparams.RootLength)
		finalized := bytesutil.PadTo([]byte("finalized"), fieldparams.RootLength)
		jsonPayload := &enginev1.ForkchoiceState{
			HeadBlockHash:      head,
			SafeBlockHash:      safe,
			FinalizedBlockHash: finalized,
		}
		enc, err := json.Marshal(jsonPayload)
		require.NoError(t, err)
		payloadPb := &enginev1.ForkchoiceState{}
		require.NoError(t, json.Unmarshal(enc, payloadPb))
		require.DeepEqual(t, head, payloadPb.HeadBlockHash)
		require.DeepEqual(t, safe, payloadPb.SafeBlockHash)
		require.DeepEqual(t, finalized, payloadPb.FinalizedBlockHash)
	})
	t.Run("transition configuration", func(t *testing.T) {
		blockHash := []byte("head")
		jsonPayload := &enginev1.TransitionConfiguration{
			TerminalBlockHash:       blockHash,
			TerminalTotalDifficulty: params.BeaconConfig().TerminalTotalDifficulty,
			TerminalBlockNumber:     big.NewInt(0).Bytes(),
		}
		enc, err := json.Marshal(jsonPayload)
		require.NoError(t, err)
		payloadPb := &enginev1.TransitionConfiguration{}
		require.NoError(t, json.Unmarshal(enc, payloadPb))
		require.DeepEqual(t, blockHash, payloadPb.TerminalBlockHash)

		require.DeepEqual(t, params.BeaconConfig().TerminalTotalDifficulty, payloadPb.TerminalTotalDifficulty)
		require.DeepEqual(t, big.NewInt(0).Bytes(), payloadPb.TerminalBlockNumber)
	})
	t.Run("execution payload", func(t *testing.T) {
		baseFeePerGas := big.NewInt(6)
		parentHash := bytesutil.PadTo([]byte("parent"), fieldparams.RootLength)
		feeRecipient := bytesutil.PadTo([]byte("feeRecipient"), fieldparams.FeeRecipientLength)
		stateRoot := bytesutil.PadTo([]byte("stateRoot"), fieldparams.RootLength)
		receiptsRoot := bytesutil.PadTo([]byte("receiptsRoot"), fieldparams.RootLength)
		logsBloom := bytesutil.PadTo([]byte("logs"), fieldparams.LogsBloomLength)
		random := bytesutil.PadTo([]byte("random"), fieldparams.RootLength)
		extra := bytesutil.PadTo([]byte("extraData"), fieldparams.RootLength)
		hash := bytesutil.PadTo([]byte("hash"), fieldparams.RootLength)
		jsonPayload := &enginev1.ExecutionPayload{
			ParentHash:    parentHash,
			FeeRecipient:  feeRecipient,
			StateRoot:     stateRoot,
			ReceiptsRoot:  receiptsRoot,
			LogsBloom:     logsBloom,
			Random:        random,
			BlockNumber:   1,
			GasLimit:      2,
			GasUsed:       3,
			Timestamp:     4,
			ExtraData:     extra,
			BaseFeePerGas: baseFeePerGas.Bytes(),
			BlockHash:     hash,
			Transactions:  [][]byte{[]byte("hi")},
		}
		enc, err := json.Marshal(jsonPayload)
		require.NoError(t, err)
		payloadPb := &enginev1.ExecutionPayload{}
		require.NoError(t, json.Unmarshal(enc, payloadPb))
		require.DeepEqual(t, parentHash, payloadPb.ParentHash)
		require.DeepEqual(t, feeRecipient, payloadPb.FeeRecipient)
		require.DeepEqual(t, stateRoot, payloadPb.StateRoot)
		require.DeepEqual(t, receiptsRoot, payloadPb.ReceiptsRoot)
		require.DeepEqual(t, logsBloom, payloadPb.LogsBloom)
		require.DeepEqual(t, random, payloadPb.Random)
		require.DeepEqual(t, uint64(1), payloadPb.BlockNumber)
		require.DeepEqual(t, uint64(2), payloadPb.GasLimit)
		require.DeepEqual(t, uint64(3), payloadPb.GasUsed)
		require.DeepEqual(t, uint64(4), payloadPb.Timestamp)
		require.DeepEqual(t, extra, payloadPb.ExtraData)
		require.DeepEqual(t, bytesutil.PadTo(baseFeePerGas.Bytes(), fieldparams.RootLength), payloadPb.BaseFeePerGas)
		require.DeepEqual(t, hash, payloadPb.BlockHash)
		require.DeepEqual(t, [][]byte{[]byte("hi")}, payloadPb.Transactions)
	})
	t.Run("execution block", func(t *testing.T) {
		jsonPayload := &enginev1.ExecutionBlock{
			Number:           []byte("100"),
			Hash:             []byte("hash"),
			ParentHash:       []byte("parent"),
			Sha3Uncles:       []byte("sha3Uncles"),
			Miner:            []byte("miner"),
			StateRoot:        []byte("stateRoot"),
			TransactionsRoot: []byte("txRoot"),
			ReceiptsRoot:     []byte("receiptsRoot"),
			LogsBloom:        []byte("logsBloom"),
			Difficulty:       []byte("1"),
			TotalDifficulty:  "2",
			GasLimit:         3,
			GasUsed:          4,
			Timestamp:        5,
			BaseFeePerGas:    []byte("6"),
			Size:             []byte("7"),
			ExtraData:        []byte("extraData"),
			MixHash:          []byte("mixHash"),
			Nonce:            []byte("nonce"),
			Transactions:     [][]byte{[]byte("hi")},
			Uncles:           [][]byte{[]byte("bye")},
		}
		enc, err := json.Marshal(jsonPayload)
		require.NoError(t, err)
		payloadPb := &enginev1.ExecutionBlock{}
		require.NoError(t, json.Unmarshal(enc, payloadPb))
		require.DeepEqual(t, []byte("100"), payloadPb.Number)
		require.DeepEqual(t, []byte("hash"), payloadPb.Hash)
		require.DeepEqual(t, []byte("parent"), payloadPb.ParentHash)
		require.DeepEqual(t, []byte("sha3Uncles"), payloadPb.Sha3Uncles)
		require.DeepEqual(t, []byte("miner"), payloadPb.Miner)
		require.DeepEqual(t, []byte("stateRoot"), payloadPb.StateRoot)
		require.DeepEqual(t, []byte("txRoot"), payloadPb.TransactionsRoot)
		require.DeepEqual(t, []byte("receiptsRoot"), payloadPb.ReceiptsRoot)
		require.DeepEqual(t, []byte("logsBloom"), payloadPb.LogsBloom)
		require.DeepEqual(t, []byte("1"), payloadPb.Difficulty)
		require.DeepEqual(t, []byte("2"), payloadPb.TotalDifficulty)
		require.DeepEqual(t, uint64(3), payloadPb.GasLimit)
		require.DeepEqual(t, uint64(4), payloadPb.GasUsed)
		require.DeepEqual(t, uint64(5), payloadPb.Timestamp)
		require.DeepEqual(t, []byte("6"), payloadPb.BaseFeePerGas)
		require.DeepEqual(t, []byte("7"), payloadPb.Size)
		require.DeepEqual(t, []byte("extraData"), payloadPb.ExtraData)
		require.DeepEqual(t, []byte("mixHash"), payloadPb.MixHash)
		require.DeepEqual(t, []byte("nonce"), payloadPb.Nonce)
		require.DeepEqual(t, [][]byte{[]byte("hi")}, payloadPb.Transactions)
		require.DeepEqual(t, [][]byte{[]byte("bye")}, payloadPb.Uncles)
	})
}

func TestPayloadIDBytes_MarshalUnmarshalJSON(t *testing.T) {
	item := [8]byte{1, 0, 0, 0, 0, 0, 0, 0}
	enc, err := json.Marshal(enginev1.PayloadIDBytes(item))
	require.NoError(t, err)
	require.DeepEqual(t, "\"0x0100000000000000\"", string(enc))
	res := &enginev1.PayloadIDBytes{}
	err = res.UnmarshalJSON(enc)
	require.NoError(t, err)
	require.Equal(t, true, item == *res)
}
