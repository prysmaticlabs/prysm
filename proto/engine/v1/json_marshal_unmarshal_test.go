package enginev1

import (
	"encoding/json"
	"testing"

	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestJsonMarshalUnmarshal(t *testing.T) {
	t.Run("execution payload", func(t *testing.T) {
		foo := bytesutil.ToBytes32([]byte("foo"))
		bar := bytesutil.PadTo([]byte("bar"), 20)
		baz := bytesutil.PadTo([]byte("baz"), 256)
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
	})
}
