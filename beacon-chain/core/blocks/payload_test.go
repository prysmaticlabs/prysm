package blocks_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func Test_MergeComplete(t *testing.T) {
	tests := []struct {
		name    string
		payload *ethpb.ExecutionPayloadHeader
		want    bool
	}{
		{
			name:    "empty header header",
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

func Test_MergeBlock(t *testing.T) {
	tests := []struct {
		name    string
		payload *ethpb.ExecutionPayload
		header  *ethpb.ExecutionPayloadHeader
		want    bool
	}{
		{
			name:    "empty header, empty payload",
			payload: emptyPayload(),
			header:  emptyPayloadHeader(),
			want:    false,
		},
		{
			name:    "non-empty header, empty payload",
			payload: emptyPayload(),
			header: func() *ethpb.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.ParentHash = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return h
			}(),
			want: false,
		},
		{
			name: "empty header, payload has parent hash",
			payload: func() *ethpb.ExecutionPayload {
				p := emptyPayload()
				p.ParentHash = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return p
			}(),
			header: emptyPayloadHeader(),
			want:   true,
		},
		{
			name: "empty header, payload has fee recipient",
			payload: func() *ethpb.ExecutionPayload {
				p := emptyPayload()
				p.FeeRecipient = bytesutil.PadTo([]byte{'a'}, fieldparams.FeeRecipientLength)
				return p
			}(),
			header: emptyPayloadHeader(),
			want:   true,
		},
		{
			name: "empty header, payload has state root",
			payload: func() *ethpb.ExecutionPayload {
				p := emptyPayload()
				p.StateRoot = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return p
			}(),
			header: emptyPayloadHeader(),
			want:   true,
		},
		{
			name: "empty header, payload has receipt root",
			payload: func() *ethpb.ExecutionPayload {
				p := emptyPayload()
				p.ReceiptRoot = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return p
			}(),
			header: emptyPayloadHeader(),
			want:   true,
		},
		{
			name: "empty header, payload has logs bloom",
			payload: func() *ethpb.ExecutionPayload {
				p := emptyPayload()
				p.LogsBloom = bytesutil.PadTo([]byte{'a'}, fieldparams.LogsBloomLength)
				return p
			}(),
			header: emptyPayloadHeader(),
			want:   true,
		},
		{
			name: "empty header, payload has random",
			payload: func() *ethpb.ExecutionPayload {
				p := emptyPayload()
				p.Random = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return p
			}(),
			header: emptyPayloadHeader(),
			want:   true,
		},
		{
			name: "empty header, payload has base fee",
			payload: func() *ethpb.ExecutionPayload {
				p := emptyPayload()
				p.BaseFeePerGas = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return p
			}(),
			header: emptyPayloadHeader(),
			want:   true,
		},
		{
			name: "empty header, payload has block hash",
			payload: func() *ethpb.ExecutionPayload {
				p := emptyPayload()
				p.BlockHash = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return p
			}(),
			header: emptyPayloadHeader(),
			want:   true,
		},
		{
			name: "empty header, payload has tx",
			payload: func() *ethpb.ExecutionPayload {
				p := emptyPayload()
				p.Transactions = [][]byte{{'a'}}
				return p
			}(),
			header: emptyPayloadHeader(),
			want:   true,
		},
		{
			name: "empty header, payload has extra data",
			payload: func() *ethpb.ExecutionPayload {
				p := emptyPayload()
				p.ExtraData = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return p
			}(),
			header: emptyPayloadHeader(),
			want:   true,
		},
		{
			name: "empty header, payload has block number",
			payload: func() *ethpb.ExecutionPayload {
				p := emptyPayload()
				p.BlockNumber = 1
				return p
			}(),
			header: emptyPayloadHeader(),
			want:   true,
		},
		{
			name: "empty header, payload has gas limit",
			payload: func() *ethpb.ExecutionPayload {
				p := emptyPayload()
				p.GasLimit = 1
				return p
			}(),
			header: emptyPayloadHeader(),
			want:   true,
		},
		{
			name: "empty header, payload has gas used",
			payload: func() *ethpb.ExecutionPayload {
				p := emptyPayload()
				p.GasUsed = 1
				return p
			}(),
			header: emptyPayloadHeader(),
			want:   true,
		},
		{
			name: "empty header, payload has timestamp",
			payload: func() *ethpb.ExecutionPayload {
				p := emptyPayload()
				p.Timestamp = 1
				return p
			}(),
			header: emptyPayloadHeader(),
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, _ := util.DeterministicGenesisStateMerge(t, 1)
			require.NoError(t, st.SetLatestExecutionPayloadHeader(tt.header))
			blk := util.NewBeaconBlockMerge()
			blk.Block.Body.ExecutionPayload = tt.payload
			body, err := wrapper.WrappedMergeBeaconBlockBody(blk.Block.Body)
			require.NoError(t, err)
			got, err := blocks.MergeBlock(st, body)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("MergeBlock() got = %v, want %v", got, tt.want)
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
