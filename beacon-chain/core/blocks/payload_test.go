package blocks_test

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
	"github.com/prysmaticlabs/prysm/time/slots"
)

func Test_MergeComplete(t *testing.T) {
	tests := []struct {
		name    string
		payload *ethpb.ExecutionPayloadHeader
		want    bool
	}{
		{
			name:    "empty payload header",
			payload: emptyPayloadHeader(),
			want:    false,
		},
		{
			name: "has parent hash",
			payload: func() *ethpb.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.ParentHash = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return h
			}(),
			want: true,
		},
		{
			name: "has fee recipient",
			payload: func() *ethpb.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.FeeRecipient = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return h
			}(),
			want: true,
		},
		{
			name: "has state root",
			payload: func() *ethpb.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.StateRoot = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return h
			}(),
			want: true,
		},
		{
			name: "has receipt root",
			payload: func() *ethpb.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.ReceiptRoot = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return h
			}(),
			want: true,
		},
		{
			name: "has logs bloom",
			payload: func() *ethpb.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.LogsBloom = bytesutil.PadTo([]byte{'a'}, fieldparams.LogsBloomLength)
				return h
			}(),
			want: true,
		},
		{
			name: "has random",
			payload: func() *ethpb.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.Random = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return h
			}(),
			want: true,
		},
		{
			name: "has base fee",
			payload: func() *ethpb.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.BaseFeePerGas = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return h
			}(),
			want: true,
		},
		{
			name: "has block hash",
			payload: func() *ethpb.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.BlockHash = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return h
			}(),
			want: true,
		},
		{
			name: "has tx root",
			payload: func() *ethpb.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.TransactionsRoot = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return h
			}(),
			want: true,
		},
		{
			name: "has extra data",
			payload: func() *ethpb.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.ExtraData = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return h
			}(),
			want: true,
		},
		{
			name: "has block number",
			payload: func() *ethpb.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.BlockNumber = 1
				return h
			}(),
			want: true,
		},
		{
			name: "has gas limit",
			payload: func() *ethpb.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.GasLimit = 1
				return h
			}(),
			want: true,
		},
		{
			name: "has gas used",
			payload: func() *ethpb.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.GasUsed = 1
				return h
			}(),
			want: true,
		},
		{
			name: "has time stamp",
			payload: func() *ethpb.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.Timestamp = 1
				return h
			}(),
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, _ := util.DeterministicGenesisStateMerge(t, 1)
			require.NoError(t, st.SetLatestExecutionPayloadHeader(tt.payload))
			got, err := blocks.MergeComplete(st)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("mergeComplete() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ValidatePayloadWhenMergeCompletes(t *testing.T) {
	tests := []struct {
		name    string
		payload *ethpb.ExecutionPayload
		header  *ethpb.ExecutionPayloadHeader
		err     error
	}{
		{
			name:    "merge incomplete",
			payload: emptyPayload(),
			header:  emptyPayloadHeader(),
			err:     nil,
		},
		{
			name:    "validate passes",
			payload: emptyPayload(),
			header: func() *ethpb.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.ParentHash = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return h
			}(),
			err: nil,
		},
		{
			name: "incorrect blockhash",
			payload: func() *ethpb.ExecutionPayload {
				h := emptyPayload()
				h.ParentHash = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return h
			}(),
			header: func() *ethpb.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.ParentHash = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return h
			}(),
			err: errors.New("incorrect block hash"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, _ := util.DeterministicGenesisStateMerge(t, 1)
			require.NoError(t, st.SetLatestExecutionPayloadHeader(tt.header))
			err := blocks.ValidatePayloadWhenMergeCompletes(st, tt.payload)
			if err != nil {
				require.Equal(t, tt.err.Error(), err.Error())
			} else {
				require.Equal(t, tt.err, err)
			}
		})
	}
}

func Test_ValidatePayload(t *testing.T) {
	st, _ := util.DeterministicGenesisStateMerge(t, 1)
	random, err := helpers.RandaoMix(st, time.CurrentEpoch(st))
	require.NoError(t, err)
	ts, err := slots.ToTime(st.GenesisTime(), st.Slot())
	require.NoError(t, err)
	tests := []struct {
		name    string
		payload *ethpb.ExecutionPayload
		err     error
	}{
		{
			name: "validate passes",
			payload: func() *ethpb.ExecutionPayload {
				h := emptyPayload()
				h.Random = random
				h.Timestamp = uint64(ts.Unix())
				return h
			}(), err: nil,
		},
		{
			name:    "incorrect random",
			payload: emptyPayload(),
			err:     errors.New("incorrect random"),
		},
		{
			name: "incorrect timestamp",
			payload: func() *ethpb.ExecutionPayload {
				h := emptyPayload()
				h.Random = random
				h.Timestamp = 1
				return h
			}(),
			err: errors.New("incorrect timestamp"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := blocks.ValidatePayload(st, tt.payload)
			if err != nil {
				require.Equal(t, tt.err.Error(), err.Error())
			} else {
				require.Equal(t, tt.err, err)
			}
		})
	}
}

func BenchmarkMergeComplete(b *testing.B) {
	st, _ := util.DeterministicGenesisStateMerge(b, 1)
	require.NoError(b, st.SetLatestExecutionPayloadHeader(emptyPayloadHeader()))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := blocks.MergeComplete(st)
		require.NoError(b, err)
	}
}

func emptyPayloadHeader() *ethpb.ExecutionPayloadHeader {
	return &ethpb.ExecutionPayloadHeader{
		ParentHash:       make([]byte, fieldparams.RootLength),
		FeeRecipient:     make([]byte, fieldparams.FeeRecipientLength),
		StateRoot:        make([]byte, fieldparams.RootLength),
		ReceiptRoot:      make([]byte, fieldparams.RootLength),
		LogsBloom:        make([]byte, fieldparams.LogsBloomLength),
		Random:           make([]byte, fieldparams.RootLength),
		BaseFeePerGas:    make([]byte, fieldparams.RootLength),
		BlockHash:        make([]byte, fieldparams.RootLength),
		TransactionsRoot: make([]byte, fieldparams.RootLength),
	}
}

func emptyPayload() *ethpb.ExecutionPayload {
	return &ethpb.ExecutionPayload{
		ParentHash:    make([]byte, fieldparams.RootLength),
		FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
		StateRoot:     make([]byte, fieldparams.RootLength),
		ReceiptRoot:   make([]byte, fieldparams.RootLength),
		LogsBloom:     make([]byte, fieldparams.LogsBloomLength),
		Random:        make([]byte, fieldparams.RootLength),
		BaseFeePerGas: make([]byte, fieldparams.RootLength),
		BlockHash:     make([]byte, fieldparams.RootLength),
	}
}
