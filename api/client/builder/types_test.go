package builder

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common/hexutil"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
	"fmt"
	"strconv"
	"testing"
)

func TestSignedValidatorRegistration_MarshalJSON(t *testing.T) {
	svr := &eth.SignedValidatorRegistrationV1{
		Message:   &eth.ValidatorRegistrationV1{
			FeeRecipient: make([]byte, 20),
			GasLimit:     0,
			Timestamp:    0,
			Pubkey:       make([]byte, 48),
		},
		Signature: make([]byte, 96),
	}
	je, err := json.Marshal(&SignedValidatorRegistration{SignedValidatorRegistrationV1: svr})
	require.NoError(t, err)
	// decode with a struct w/ plain strings so we can check the string encoding of the hex fields
	un := struct{
		Message struct {
			FeeRecipient string `json:"fee_recipient"`
			Pubkey string `json:"pubkey"`
		} `json:"message"`
		Signature string `json:"signature"`
	}{}
	require.NoError(t, json.Unmarshal(je, &un))
	require.Equal(t, "0x000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", un.Signature)
	require.Equal(t, "0x0000000000000000000000000000000000000000", un.Message.FeeRecipient)
	require.Equal(t, "0x000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", un.Message.Pubkey)
}

var testExampleHeaderResponse = `{
  "version": "bellatrix",
  "data": {
    "message": {
      "header": {
        "parent_hash": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
        "fee_recipient": "0xabcf8e0d4e9587369b2301d0790347320302cc09",
        "state_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
        "receipts_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
        "logs_bloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
        "prev_randao": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
        "block_number": "1",
        "gas_limit": "1",
        "gas_used": "1",
        "timestamp": "1",
        "extra_data": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
        "base_fee_per_gas": "1",
        "block_hash": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
        "transactions_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
      },
      "value": "1",
      "pubkey": "0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a"
    },
    "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
  }
}`

func TestExecutionHeaderResponseUnmarshal(t *testing.T) {
	hr := &ExecHeaderResponse{}
	require.NoError(t, json.Unmarshal([]byte(testExampleHeaderResponse), hr))
	cases := []struct{
		expected string
		actual string
		name string
	}{
		{
			expected: "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505",
			actual: fmt.Sprintf("%#x", hr.Data.Signature),
			name: "Signature",
		},
		{
			expected: "0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a",
			actual: fmt.Sprintf("%#x", hr.Data.Message.Pubkey),
			name: "ExecHeaderResponse.Pubkey",
		},
		{
			expected: "1",
			actual: fmt.Sprintf("%d", hr.Data.Message.Value),
			name: "ExecHeaderResponse.Value",
		},
		{
			expected: "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
			actual: fmt.Sprintf("%#x", hr.Data.Message.Header.ParentHash),
			name: "ExecHeaderResponse.ExecutionPayloadHeader.ParentHash",
		},
		{
			expected: "0xabcf8e0d4e9587369b2301d0790347320302cc09",
			actual: fmt.Sprintf("%#x", hr.Data.Message.Header.FeeRecipient),
			name: "ExecHeaderResponse.ExecutionPayloadHeader.FeeRecipient",
		},
		{
			expected: "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
			actual: fmt.Sprintf("%#x", hr.Data.Message.Header.StateRoot),
			name: "ExecHeaderResponse.ExecutionPayloadHeader.StateRoot",
		},
		{
			expected: "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
			actual: fmt.Sprintf("%#x", hr.Data.Message.Header.ReceiptsRoot),
			name: "ExecHeaderResponse.ExecutionPayloadHeader.ReceiptsRoot",
		},
		{
			expected: "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
			actual: fmt.Sprintf("%#x", hr.Data.Message.Header.LogsBloom),
			name: "ExecHeaderResponse.ExecutionPayloadHeader.LogsBloom",
		},
		{
			expected: "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
			actual: fmt.Sprintf("%#x", hr.Data.Message.Header.PrevRandao),
			name: "ExecHeaderResponse.ExecutionPayloadHeader.PrevRandao",
		},
		{
			expected: "1",
			actual: fmt.Sprintf("%d", hr.Data.Message.Header.BlockNumber),
			name: "ExecHeaderResponse.ExecutionPayloadHeader.BlockNumber",
		},
		{
			expected: "1",
			actual: fmt.Sprintf("%d", hr.Data.Message.Header.GasLimit),
			name: "ExecHeaderResponse.ExecutionPayloadHeader.GasLimit",
		},
		{
			expected: "1",
			actual: fmt.Sprintf("%d", hr.Data.Message.Header.GasUsed),
			name: "ExecHeaderResponse.ExecutionPayloadHeader.GasUsed",
		},
		{
			expected: "1",
			actual: fmt.Sprintf("%d", hr.Data.Message.Header.Timestamp),
			name: "ExecHeaderResponse.ExecutionPayloadHeader.Timestamp",
		},
		{
			expected: "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
			actual: fmt.Sprintf("%#x", hr.Data.Message.Header.ExtraData),
			name: "ExecHeaderResponse.ExecutionPayloadHeader.ExtraData",
		},
		{
			expected: "1",
			actual: fmt.Sprintf("%d", hr.Data.Message.Header.BaseFeePerGas),
			name: "ExecHeaderResponse.ExecutionPayloadHeader.BaseFeePerGas",
		},
		{
			expected: "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
			actual: fmt.Sprintf("%#x", hr.Data.Message.Header.BlockHash),
			name: "ExecHeaderResponse.ExecutionPayloadHeader.BlockHash",
		},
		{
			expected: "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
			actual: fmt.Sprintf("%#x", hr.Data.Message.Header.TransactionsRoot),
			name: "ExecHeaderResponse.ExecutionPayloadHeader.TransactionsRoot",
		},
	}
	for _, c := range cases {
		require.Equal(t, c.expected, c.actual, fmt.Sprintf("unexpected value for field %s", c.name))
	}
}

func TestExecutionHeaderResponseToProto(t *testing.T) {
	hr := &ExecHeaderResponse{}
	require.NoError(t, json.Unmarshal([]byte(testExampleHeaderResponse), hr))
	p, err := hr.ToProto()
	require.NoError(t, err)
	signature, err := hexutil.Decode("0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505")
	require.NoError(t, err)
	pubkey, err := hexutil.Decode("0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a")
	require.NoError(t, err)
	parentHash, err := hexutil.Decode("0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2")
	require.NoError(t, err)
	feeRecipient, err := hexutil.Decode("0xabcf8e0d4e9587369b2301d0790347320302cc09")
	require.NoError(t, err)
	stateRoot, err := hexutil.Decode("0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2")
	require.NoError(t, err)
	receiptsRoot, err := hexutil.Decode("0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2")
	require.NoError(t, err)
	logsBloom, err := hexutil.Decode("0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	require.NoError(t, err)
	prevRandao, err := hexutil.Decode("0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2")
	require.NoError(t, err)
	extraData, err := hexutil.Decode("0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2")
	require.NoError(t, err)
	blockHash, err := hexutil.Decode("0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2")
	require.NoError(t, err)
	txRoot, err := hexutil.Decode("0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2")
	require.NoError(t, err)
	expected := &eth.SignedBuilderBid{
		Message:   &eth.BuilderBid{
			Header: &eth.ExecutionPayloadHeader{
				ParentHash:       parentHash,
				FeeRecipient:     feeRecipient,
				StateRoot:        stateRoot,
				ReceiptsRoot:     receiptsRoot,
				LogsBloom:        logsBloom,
				PrevRandao:       prevRandao,
				BlockNumber:      1,
				GasLimit:         1,
				GasUsed:          1,
				Timestamp:        1,
				ExtraData:        extraData,
				// TODO assumes weird byte slice field
				BaseFeePerGas:    []byte(strconv.FormatUint(uint64(1), 10)),
				BlockHash:        blockHash,
				TransactionsRoot: txRoot,
			},
			// TODO assumes weird byte slice field
			Value:  []byte(strconv.FormatUint(uint64(1), 10)),
			Pubkey: pubkey,
		},
		Signature: signature,
	}
	require.DeepEqual(t, expected, p)
}
