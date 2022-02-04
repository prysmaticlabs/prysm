package enginev1_test

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestJsonMarshalUnmarshal(t *testing.T) {
	foo := bytesutil.ToBytes32([]byte("foo"))
	bar := bytesutil.PadTo([]byte("bar"), 20)
	baz := bytesutil.PadTo([]byte("baz"), 256)
	t.Run("payload attributes", func(t *testing.T) {
		jsonPayload := &enginev1.PayloadAttributes{
			Timestamp:             1,
			Random:                enginev1.HexBytes(foo[:]),
			SuggestedFeeRecipient: enginev1.HexBytes(bar),
		}
		enc, err := json.Marshal(jsonPayload)
		require.NoError(t, err)
		payloadPb := &enginev1.PayloadAttributes{}
		require.NoError(t, json.Unmarshal(enc, payloadPb))
		require.DeepEqual(t, uint64(1), payloadPb.Timestamp)
		require.DeepEqual(t, foo[:], payloadPb.Random)
		require.DeepEqual(t, bar, payloadPb.SuggestedFeeRecipient)
	})
	t.Run("payload status", func(t *testing.T) {
		jsonPayload := &enginev1.PayloadStatus{
			Status:          enginev1.PayloadStatus_INVALID,
			LatestValidHash: foo[:],
			ValidationError: "failed validation",
		}
		enc, err := json.Marshal(jsonPayload)
		require.NoError(t, err)
		payloadPb := &enginev1.PayloadStatus{}
		require.NoError(t, json.Unmarshal(enc, payloadPb))
		require.DeepEqual(t, "INVALID", payloadPb.Status.String())
		require.DeepEqual(t, foo[:], payloadPb.LatestValidHash)
		require.DeepEqual(t, "failed validation", payloadPb.ValidationError)
	})
	t.Run("forkchoice state", func(t *testing.T) {
		jsonPayload := &enginev1.ForkchoiceState{
			HeadBlockHash:      enginev1.HexBytes(foo[:]),
			SafeBlockHash:      enginev1.HexBytes(foo[:]),
			FinalizedBlockHash: enginev1.HexBytes(foo[:]),
		}
		enc, err := json.Marshal(jsonPayload)
		require.NoError(t, err)
		payloadPb := &enginev1.ForkchoiceState{}
		require.NoError(t, json.Unmarshal(enc, payloadPb))
		require.DeepEqual(t, foo[:], payloadPb.HeadBlockHash)
		require.DeepEqual(t, foo[:], payloadPb.SafeBlockHash)
		require.DeepEqual(t, foo[:], payloadPb.FinalizedBlockHash)
	})
	t.Run("execution payload", func(t *testing.T) {
		jsonPayload := &enginev1.ExecutionPayload{
			ParentHash:    foo[:],
			FeeRecipient:  bar,
			StateRoot:     foo[:],
			ReceiptsRoot:  foo[:],
			LogsBloom:     baz,
			Random:        foo[:],
			BlockNumber:   1,
			GasLimit:      2,
			GasUsed:       3,
			Timestamp:     4,
			ExtraData:     foo[:],
			BaseFeePerGas: foo[:],
			BlockHash:     foo[:],
			Transactions:  [][]byte{foo[:]},
		}
		enc, err := json.Marshal(jsonPayload)
		require.NoError(t, err)
		payloadPb := &enginev1.ExecutionPayload{}
		require.NoError(t, json.Unmarshal(enc, payloadPb))
		require.DeepEqual(t, foo[:], payloadPb.ParentHash)
		require.DeepEqual(t, bar, payloadPb.FeeRecipient)
		require.DeepEqual(t, foo[:], payloadPb.StateRoot)
		require.DeepEqual(t, foo[:], payloadPb.ReceiptsRoot)
		require.DeepEqual(t, baz, payloadPb.LogsBloom)
		require.DeepEqual(t, foo[:], payloadPb.Random)
		require.DeepEqual(t, uint64(1), payloadPb.BlockNumber)
		require.DeepEqual(t, uint64(2), payloadPb.GasLimit)
		require.DeepEqual(t, uint64(3), payloadPb.GasUsed)
		require.DeepEqual(t, uint64(4), payloadPb.Timestamp)
		require.DeepEqual(t, foo[:], payloadPb.ExtraData)
		require.DeepEqual(t, foo[:], payloadPb.BaseFeePerGas)
		require.DeepEqual(t, foo[:], payloadPb.BlockHash)
		require.DeepEqual(t, [][]byte{foo[:]}, payloadPb.Transactions)
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
