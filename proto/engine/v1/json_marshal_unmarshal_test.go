package enginev1

import (
	"encoding/json"
	"testing"

	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestJsonMarshalUnmarshal(t *testing.T) {
	foo := bytesutil.ToBytes32([]byte("foo"))
	bar := bytesutil.PadTo([]byte("bar"), 20)
	baz := bytesutil.PadTo([]byte("baz"), 256)
	t.Run("payload attributes", func(t *testing.T) {
		jsonPayload := map[string]interface{}{
			"timestamp":             1,
			"random":                foo[:],
			"suggestedFeeRecipient": bar,
		}
		enc, err := json.Marshal(jsonPayload)
		require.NoError(t, err)
		payloadPb := &PayloadAttributes{}
		require.NoError(t, json.Unmarshal(enc, payloadPb))
		require.DeepEqual(t, uint64(1), payloadPb.Timestamp)
		require.DeepEqual(t, foo[:], payloadPb.Random)
		require.DeepEqual(t, bar, payloadPb.SuggestedFeeRecipient)
	})
	t.Run("payload status", func(t *testing.T) {
		jsonPayload := map[string]interface{}{
			"status":          "INVALID",
			"latestValidHash": foo[:],
			"validationError": "failed validation",
		}
		enc, err := json.Marshal(jsonPayload)
		require.NoError(t, err)
		payloadPb := &PayloadStatus{}
		require.NoError(t, json.Unmarshal(enc, payloadPb))
		require.DeepEqual(t, "INVALID", payloadPb.Status.String())
		require.DeepEqual(t, foo[:], payloadPb.LatestValidHash)
		require.DeepEqual(t, "failed validation", payloadPb.ValidationError)
	})
	t.Run("forkchoice state", func(t *testing.T) {
		jsonPayload := map[string]interface{}{
			"headBlockHash":      foo[:],
			"safeBlockHash":      foo[:],
			"finalizedBlockHash": foo[:],
		}
		enc, err := json.Marshal(jsonPayload)
		require.NoError(t, err)
		payloadPb := &ForkchoiceState{}
		require.NoError(t, json.Unmarshal(enc, payloadPb))
		require.DeepEqual(t, foo[:], payloadPb.HeadBlockHash)
		require.DeepEqual(t, foo[:], payloadPb.SafeBlockHash)
		require.DeepEqual(t, foo[:], payloadPb.FinalizedBlockHash)
	})
	t.Run("execution payload", func(t *testing.T) {
		jsonPayload := map[string]interface{}{
			"parentHash":     foo[:],
			"feeRecipient":   bar,
			"stateRoot":      foo[:],
			"recipientsRoot": foo[:],
			"logsBloom":      baz,
			"random":         foo[:],
			"blockNumber":    1,
			"gasLimit":       1,
			"gasUsed":        1,
			"timestamp":      1,
			"extraData":      foo[:],
			"baseFeePerGas":  foo[:],
			"blockHash":      foo[:],
			"transactions":   [][]byte{foo[:]},
		}
		enc, err := json.Marshal(jsonPayload)
		require.NoError(t, err)
		payloadPb := &ExecutionPayload{}
		require.NoError(t, json.Unmarshal(enc, payloadPb))
		require.DeepEqual(t, foo[:], payloadPb.ParentHash)
		require.DeepEqual(t, bar, payloadPb.FeeRecipient)
		require.DeepEqual(t, foo[:], payloadPb.StateRoot)
		require.DeepEqual(t, foo[:], payloadPb.RecipientsRoot)
		require.DeepEqual(t, baz, payloadPb.LogsBloom)
		require.DeepEqual(t, foo[:], payloadPb.Random)
		require.DeepEqual(t, uint64(1), payloadPb.BlockNumber)
		require.DeepEqual(t, uint64(1), payloadPb.GasLimit)
		require.DeepEqual(t, uint64(1), payloadPb.GasUsed)
		require.DeepEqual(t, uint64(1), payloadPb.Timestamp)
		require.DeepEqual(t, foo[:], payloadPb.ExtraData)
		require.DeepEqual(t, foo[:], payloadPb.BaseFeePerGas)
		require.DeepEqual(t, foo[:], payloadPb.BlockHash)
		require.DeepEqual(t, [][]byte{foo[:]}, payloadPb.Transactions)
	})
}
