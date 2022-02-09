package enginev1_test

import (
	"encoding/json"
	"math"
	"testing"

	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestJsonMarshalUnmarshal(t *testing.T) {
	t.Run("payload attributes", func(t *testing.T) {
		jsonPayload := &enginev1.PayloadAttributes{
			Timestamp:             1,
			Random:                []byte("random"),
			SuggestedFeeRecipient: []byte("suggestedFeeRecipient"),
		}
		enc, err := json.Marshal(jsonPayload)
		require.NoError(t, err)
		payloadPb := &enginev1.PayloadAttributes{}
		require.NoError(t, json.Unmarshal(enc, payloadPb))
		require.DeepEqual(t, uint64(1), payloadPb.Timestamp)
		require.DeepEqual(t, []byte("random"), payloadPb.Random)
		require.DeepEqual(t, []byte("suggestedFeeRecipient"), payloadPb.SuggestedFeeRecipient)
	})
	t.Run("payload status", func(t *testing.T) {
		jsonPayload := &enginev1.PayloadStatus{
			Status:          enginev1.PayloadStatus_INVALID,
			LatestValidHash: []byte("latestValidHash"),
			ValidationError: "failed validation",
		}
		enc, err := json.Marshal(jsonPayload)
		require.NoError(t, err)
		payloadPb := &enginev1.PayloadStatus{}
		require.NoError(t, json.Unmarshal(enc, payloadPb))
		require.DeepEqual(t, "INVALID", payloadPb.Status.String())
		require.DeepEqual(t, []byte("latestValidHash"), payloadPb.LatestValidHash)
		require.DeepEqual(t, "failed validation", payloadPb.ValidationError)
	})
	t.Run("forkchoice state", func(t *testing.T) {
		jsonPayload := &enginev1.ForkchoiceState{
			HeadBlockHash:      []byte("head"),
			SafeBlockHash:      []byte("safe"),
			FinalizedBlockHash: []byte("finalized"),
		}
		enc, err := json.Marshal(jsonPayload)
		require.NoError(t, err)
		payloadPb := &enginev1.ForkchoiceState{}
		require.NoError(t, json.Unmarshal(enc, payloadPb))
		require.DeepEqual(t, []byte("head"), payloadPb.HeadBlockHash)
		require.DeepEqual(t, []byte("safe"), payloadPb.SafeBlockHash)
		require.DeepEqual(t, []byte("finalized"), payloadPb.FinalizedBlockHash)
	})
	t.Run("execution payload", func(t *testing.T) {
		jsonPayload := &enginev1.ExecutionPayload{
			ParentHash:    []byte("parent"),
			FeeRecipient:  []byte("feeRecipient"),
			StateRoot:     []byte("stateRoot"),
			ReceiptsRoot:  []byte("receiptsRoot"),
			LogsBloom:     []byte("logsBloom"),
			Random:        []byte("random"),
			BlockNumber:   1,
			GasLimit:      2,
			GasUsed:       3,
			Timestamp:     4,
			ExtraData:     []byte("extraData"),
			BaseFeePerGas: []byte("baseFeePerGas"),
			BlockHash:     []byte("blockHash"),
			Transactions:  [][]byte{[]byte("hi")},
		}
		enc, err := json.Marshal(jsonPayload)
		require.NoError(t, err)
		payloadPb := &enginev1.ExecutionPayload{}
		require.NoError(t, json.Unmarshal(enc, payloadPb))
		require.DeepEqual(t, []byte("parent"), payloadPb.ParentHash)
		require.DeepEqual(t, []byte("feeRecipient"), payloadPb.FeeRecipient)
		require.DeepEqual(t, []byte("stateRoot"), payloadPb.StateRoot)
		require.DeepEqual(t, []byte("receiptsRoot"), payloadPb.ReceiptsRoot)
		require.DeepEqual(t, []byte("logsBloom"), payloadPb.LogsBloom)
		require.DeepEqual(t, []byte("random"), payloadPb.Random)
		require.DeepEqual(t, uint64(1), payloadPb.BlockNumber)
		require.DeepEqual(t, uint64(2), payloadPb.GasLimit)
		require.DeepEqual(t, uint64(3), payloadPb.GasUsed)
		require.DeepEqual(t, uint64(4), payloadPb.Timestamp)
		require.DeepEqual(t, []byte("extraData"), payloadPb.ExtraData)
		require.DeepEqual(t, []byte("baseFeePerGas"), payloadPb.BaseFeePerGas)
		require.DeepEqual(t, []byte("blockHash"), payloadPb.BlockHash)
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
			TotalDifficulty:  []byte("2"),
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

func TestHexBytes_MarshalUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		b    enginev1.HexBytes
	}{
		{
			name: "empty",
			b:    []byte{},
		},
		{
			name: "foo",
			b:    []byte("foo"),
		},
		{
			name: "bytes",
			b:    []byte{1, 2, 3, 4},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.b.MarshalJSON()
			require.NoError(t, err)
			var dec enginev1.HexBytes
			err = dec.UnmarshalJSON(got)
			require.NoError(t, err)
			require.DeepEqual(t, tt.b, dec)
		})
	}
}

func TestQuantity_MarshalUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		b    enginev1.Quantity
	}{
		{
			name: "zero",
			b:    0,
		},
		{
			name: "num",
			b:    5,
		},
		{
			name: "max",
			b:    math.MaxUint64,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.b.MarshalJSON()
			require.NoError(t, err)
			var dec enginev1.Quantity
			err = dec.UnmarshalJSON(got)
			require.NoError(t, err)
			require.DeepEqual(t, tt.b, dec)
		})
	}
}
