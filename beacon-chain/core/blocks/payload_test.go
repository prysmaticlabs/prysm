package blocks_test

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/encoding/ssz"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
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
			st, _ := util.DeterministicGenesisStateBellatrix(t, 1)
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
		payload *enginev1.ExecutionPayload
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
			payload: func() *enginev1.ExecutionPayload {
				p := emptyPayload()
				p.ParentHash = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return p
			}(),
			header: emptyPayloadHeader(),
			want:   true,
		},
		{
			name: "empty header, payload has fee recipient",
			payload: func() *enginev1.ExecutionPayload {
				p := emptyPayload()
				p.FeeRecipient = bytesutil.PadTo([]byte{'a'}, fieldparams.FeeRecipientLength)
				return p
			}(),
			header: emptyPayloadHeader(),
			want:   true,
		},
		{
			name: "empty header, payload has state root",
			payload: func() *enginev1.ExecutionPayload {
				p := emptyPayload()
				p.StateRoot = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return p
			}(),
			header: emptyPayloadHeader(),
			want:   true,
		},
		{
			name: "empty header, payload has receipt root",
			payload: func() *enginev1.ExecutionPayload {
				p := emptyPayload()
				p.ReceiptsRoot = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return p
			}(),
			header: emptyPayloadHeader(),
			want:   true,
		},
		{
			name: "empty header, payload has logs bloom",
			payload: func() *enginev1.ExecutionPayload {
				p := emptyPayload()
				p.LogsBloom = bytesutil.PadTo([]byte{'a'}, fieldparams.LogsBloomLength)
				return p
			}(),
			header: emptyPayloadHeader(),
			want:   true,
		},
		{
			name: "empty header, payload has random",
			payload: func() *enginev1.ExecutionPayload {
				p := emptyPayload()
				p.Random = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return p
			}(),
			header: emptyPayloadHeader(),
			want:   true,
		},
		{
			name: "empty header, payload has base fee",
			payload: func() *enginev1.ExecutionPayload {
				p := emptyPayload()
				p.BaseFeePerGas = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return p
			}(),
			header: emptyPayloadHeader(),
			want:   true,
		},
		{
			name: "empty header, payload has block hash",
			payload: func() *enginev1.ExecutionPayload {
				p := emptyPayload()
				p.BlockHash = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return p
			}(),
			header: emptyPayloadHeader(),
			want:   true,
		},
		{
			name: "empty header, payload has tx",
			payload: func() *enginev1.ExecutionPayload {
				p := emptyPayload()
				p.Transactions = [][]byte{{'a'}}
				return p
			}(),
			header: emptyPayloadHeader(),
			want:   true,
		},
		{
			name: "empty header, payload has extra data",
			payload: func() *enginev1.ExecutionPayload {
				p := emptyPayload()
				p.ExtraData = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return p
			}(),
			header: emptyPayloadHeader(),
			want:   true,
		},
		{
			name: "empty header, payload has block number",
			payload: func() *enginev1.ExecutionPayload {
				p := emptyPayload()
				p.BlockNumber = 1
				return p
			}(),
			header: emptyPayloadHeader(),
			want:   true,
		},
		{
			name: "empty header, payload has gas limit",
			payload: func() *enginev1.ExecutionPayload {
				p := emptyPayload()
				p.GasLimit = 1
				return p
			}(),
			header: emptyPayloadHeader(),
			want:   true,
		},
		{
			name: "empty header, payload has gas used",
			payload: func() *enginev1.ExecutionPayload {
				p := emptyPayload()
				p.GasUsed = 1
				return p
			}(),
			header: emptyPayloadHeader(),
			want:   true,
		},
		{
			name: "empty header, payload has timestamp",
			payload: func() *enginev1.ExecutionPayload {
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
			st, _ := util.DeterministicGenesisStateBellatrix(t, 1)
			require.NoError(t, st.SetLatestExecutionPayloadHeader(tt.header))
			blk := util.NewBeaconBlockBellatrix()
			blk.Block.Body.ExecutionPayload = tt.payload
			body, err := wrapper.WrappedBellatrixBeaconBlockBody(blk.Block.Body)
			require.NoError(t, err)
			got, err := blocks.IsMergeBlock(st, body)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("IsMergeBlock() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_IsExecutionBlock(t *testing.T) {
	tests := []struct {
		name    string
		payload *enginev1.ExecutionPayload
		want    bool
	}{
		{
			name:    "empty payload",
			payload: emptyPayload(),
			want:    false,
		},
		{
			name: "non-empty payload",
			payload: func() *enginev1.ExecutionPayload {
				p := emptyPayload()
				p.ParentHash = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return p
			}(),
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blk := util.NewBeaconBlockBellatrix()
			blk.Block.Body.ExecutionPayload = tt.payload
			wrappedBlock, err := wrapper.WrappedBellatrixBeaconBlock(blk.Block)
			require.NoError(t, err)
			got, err := blocks.IsExecutionBlock(wrappedBlock)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_ExecutionEnabled(t *testing.T) {
	tests := []struct {
		name    string
		payload *enginev1.ExecutionPayload
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
			want: true,
		},
		{
			name:   "empty header, non-empty payload",
			header: emptyPayloadHeader(),
			payload: func() *enginev1.ExecutionPayload {
				p := emptyPayload()
				p.Timestamp = 1
				return p
			}(),
			want: true,
		},
		{
			name: "non-empty header, non-empty payload",
			header: func() *ethpb.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.ParentHash = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return h
			}(),
			payload: func() *enginev1.ExecutionPayload {
				p := emptyPayload()
				p.Timestamp = 1
				return p
			}(),
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, _ := util.DeterministicGenesisStateBellatrix(t, 1)
			require.NoError(t, st.SetLatestExecutionPayloadHeader(tt.header))
			blk := util.NewBeaconBlockBellatrix()
			blk.Block.Body.ExecutionPayload = tt.payload
			body, err := wrapper.WrappedBellatrixBeaconBlockBody(blk.Block.Body)
			require.NoError(t, err)
			got, err := blocks.ExecutionEnabled(st, body)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("ExecutionEnabled() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ValidatePayloadWhenMergeCompletes(t *testing.T) {
	tests := []struct {
		name    string
		payload *enginev1.ExecutionPayload
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
			name: "validate passes",
			payload: func() *enginev1.ExecutionPayload {
				p := emptyPayload()
				p.ParentHash = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return p
			}(),
			header: func() *ethpb.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.BlockHash = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return h
			}(),
			err: nil,
		},
		{
			name: "incorrect blockhash",
			payload: func() *enginev1.ExecutionPayload {
				p := emptyPayload()
				p.ParentHash = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return p
			}(),
			header: func() *ethpb.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.BlockHash = bytesutil.PadTo([]byte{'b'}, fieldparams.RootLength)
				return h
			}(),
			err: errors.New("incorrect block hash"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, _ := util.DeterministicGenesisStateBellatrix(t, 1)
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
	st, _ := util.DeterministicGenesisStateBellatrix(t, 1)
	random, err := helpers.RandaoMix(st, time.CurrentEpoch(st))
	require.NoError(t, err)
	ts, err := slots.ToTime(st.GenesisTime(), st.Slot())
	require.NoError(t, err)
	tests := []struct {
		name    string
		payload *enginev1.ExecutionPayload
		err     error
	}{
		{
			name: "validate passes",
			payload: func() *enginev1.ExecutionPayload {
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
			payload: func() *enginev1.ExecutionPayload {
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

func Test_ProcessPayload(t *testing.T) {
	st, _ := util.DeterministicGenesisStateBellatrix(t, 1)
	random, err := helpers.RandaoMix(st, time.CurrentEpoch(st))
	require.NoError(t, err)
	ts, err := slots.ToTime(st.GenesisTime(), st.Slot())
	require.NoError(t, err)
	tests := []struct {
		name    string
		payload *enginev1.ExecutionPayload
		err     error
	}{
		{
			name: "process passes",
			payload: func() *enginev1.ExecutionPayload {
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
			payload: func() *enginev1.ExecutionPayload {
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
			st, err := blocks.ProcessPayload(st, tt.payload)
			if err != nil {
				require.Equal(t, tt.err.Error(), err.Error())
			} else {
				require.Equal(t, tt.err, err)
				want, err := blocks.PayloadToHeader(tt.payload)
				require.Equal(t, tt.err, err)
				got, err := st.LatestExecutionPayloadHeader()
				require.NoError(t, err)
				require.DeepSSZEqual(t, want, got)
			}
		})
	}
}

func Test_PayloadToHeader(t *testing.T) {
	p := emptyPayload()
	h, err := blocks.PayloadToHeader(p)
	require.NoError(t, err)
	txRoot, err := ssz.TransactionsRoot(p.Transactions)
	require.NoError(t, err)
	require.DeepSSZEqual(t, txRoot, bytesutil.ToBytes32(h.TransactionsRoot))

	// Verify copy works
	b := []byte{'a'}
	p.ParentHash = b
	p.FeeRecipient = b
	p.StateRoot = b
	p.ReceiptsRoot = b
	p.LogsBloom = b
	p.Random = b
	p.ExtraData = b
	p.BaseFeePerGas = b
	p.BlockHash = b
	p.BlockNumber = 1
	p.GasUsed = 1
	p.GasLimit = 1
	p.Timestamp = 1

	require.DeepSSZEqual(t, h.ParentHash, make([]byte, fieldparams.RootLength))
	require.DeepSSZEqual(t, h.FeeRecipient, make([]byte, fieldparams.FeeRecipientLength))
	require.DeepSSZEqual(t, h.StateRoot, make([]byte, fieldparams.RootLength))
	require.DeepSSZEqual(t, h.ReceiptRoot, make([]byte, fieldparams.RootLength))
	require.DeepSSZEqual(t, h.LogsBloom, make([]byte, fieldparams.LogsBloomLength))
	require.DeepSSZEqual(t, h.Random, make([]byte, fieldparams.RootLength))
	require.DeepSSZEqual(t, h.ExtraData, make([]byte, 0))
	require.DeepSSZEqual(t, h.BaseFeePerGas, make([]byte, fieldparams.RootLength))
	require.DeepSSZEqual(t, h.BlockHash, make([]byte, fieldparams.RootLength))
	require.Equal(t, h.BlockNumber, uint64(0))
	require.Equal(t, h.GasUsed, uint64(0))
	require.Equal(t, h.GasLimit, uint64(0))
	require.Equal(t, h.Timestamp, uint64(0))
}

func BenchmarkBellatrixComplete(b *testing.B) {
	st, _ := util.DeterministicGenesisStateBellatrix(b, 1)
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
		ExtraData:        make([]byte, 0),
	}
}

func emptyPayload() *enginev1.ExecutionPayload {
	return &enginev1.ExecutionPayload{
		ParentHash:    make([]byte, fieldparams.RootLength),
		FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
		StateRoot:     make([]byte, fieldparams.RootLength),
		ReceiptsRoot:  make([]byte, fieldparams.RootLength),
		LogsBloom:     make([]byte, fieldparams.LogsBloomLength),
		Random:        make([]byte, fieldparams.RootLength),
		BaseFeePerGas: make([]byte, fieldparams.RootLength),
		BlockHash:     make([]byte, fieldparams.RootLength),
		Transactions:  make([][]byte, 0),
		ExtraData:     make([]byte, 0),
	}
}
