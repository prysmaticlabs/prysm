package blocks_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	consensusblocks "github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v3/encoding/ssz"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

func Test_IsMergeComplete(t *testing.T) {
	tests := []struct {
		name    string
		payload *enginev1.ExecutionPayloadHeader
		want    bool
	}{
		{
			name:    "empty payload header",
			payload: emptyPayloadHeader(),
			want:    false,
		},
		{
			name: "has parent hash",
			payload: func() *enginev1.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.ParentHash = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return h
			}(),
			want: true,
		},
		{
			name: "has fee recipient",
			payload: func() *enginev1.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.FeeRecipient = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return h
			}(),
			want: true,
		},
		{
			name: "has state root",
			payload: func() *enginev1.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.StateRoot = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return h
			}(),
			want: true,
		},
		{
			name: "has receipt root",
			payload: func() *enginev1.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.ReceiptsRoot = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return h
			}(),
			want: true,
		},
		{
			name: "has logs bloom",
			payload: func() *enginev1.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.LogsBloom = bytesutil.PadTo([]byte{'a'}, fieldparams.LogsBloomLength)
				return h
			}(),
			want: true,
		},
		{
			name: "has random",
			payload: func() *enginev1.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.PrevRandao = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return h
			}(),
			want: true,
		},
		{
			name: "has base fee",
			payload: func() *enginev1.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.BaseFeePerGas = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return h
			}(),
			want: true,
		},
		{
			name: "has block hash",
			payload: func() *enginev1.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.BlockHash = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return h
			}(),
			want: true,
		},
		{
			name: "has extra data",
			payload: func() *enginev1.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.ExtraData = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				return h
			}(),
			want: true,
		},
		{
			name: "has block number",
			payload: func() *enginev1.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.BlockNumber = 1
				return h
			}(),
			want: true,
		},
		{
			name: "has gas limit",
			payload: func() *enginev1.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.GasLimit = 1
				return h
			}(),
			want: true,
		},
		{
			name: "has gas used",
			payload: func() *enginev1.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.GasUsed = 1
				return h
			}(),
			want: true,
		},
		{
			name: "has time stamp",
			payload: func() *enginev1.ExecutionPayloadHeader {
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
			wrappedHeader, err := consensusblocks.WrappedExecutionPayloadHeader(tt.payload)
			require.NoError(t, err)
			require.NoError(t, st.SetLatestExecutionPayloadHeader(wrappedHeader))
			got, err := blocks.IsMergeTransitionComplete(st)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("mergeComplete() got = %v, want %v", got, tt.want)
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
			wrappedBlock, err := consensusblocks.NewBeaconBlock(blk.Block)
			require.NoError(t, err)
			got, err := blocks.IsExecutionBlock(wrappedBlock.Body())
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_IsExecutionEnabled(t *testing.T) {
	tests := []struct {
		name        string
		payload     *enginev1.ExecutionPayload
		header      *enginev1.ExecutionPayloadHeader
		useAltairSt bool
		want        bool
	}{
		{
			name:        "use older than bellatrix state",
			payload:     emptyPayload(),
			header:      emptyPayloadHeader(),
			useAltairSt: true,
			want:        false,
		},
		{
			name:    "empty header, empty payload",
			payload: emptyPayload(),
			header:  emptyPayloadHeader(),
			want:    false,
		},
		{
			name:    "non-empty header, empty payload",
			payload: emptyPayload(),
			header: func() *enginev1.ExecutionPayloadHeader {
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
			header: func() *enginev1.ExecutionPayloadHeader {
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
			wrappedHeader, err := consensusblocks.WrappedExecutionPayloadHeader(tt.header)
			require.NoError(t, err)
			require.NoError(t, st.SetLatestExecutionPayloadHeader(wrappedHeader))
			blk := util.NewBeaconBlockBellatrix()
			blk.Block.Body.ExecutionPayload = tt.payload
			body, err := consensusblocks.NewBeaconBlockBody(blk.Block.Body)
			require.NoError(t, err)
			if tt.useAltairSt {
				st, _ = util.DeterministicGenesisStateAltair(t, 1)
			}
			got, err := blocks.IsExecutionEnabled(st, body)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("IsExecutionEnabled() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_IsExecutionEnabledUsingHeader(t *testing.T) {
	tests := []struct {
		name    string
		payload *enginev1.ExecutionPayload
		header  *enginev1.ExecutionPayloadHeader
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
			header: func() *enginev1.ExecutionPayloadHeader {
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
			header: func() *enginev1.ExecutionPayloadHeader {
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
			blk := util.NewBeaconBlockBellatrix()
			blk.Block.Body.ExecutionPayload = tt.payload
			body, err := consensusblocks.NewBeaconBlockBody(blk.Block.Body)
			require.NoError(t, err)
			got, err := blocks.IsExecutionEnabledUsingHeader(tt.header, body)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("IsExecutionEnabled() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ValidatePayloadWhenMergeCompletes(t *testing.T) {
	tests := []struct {
		name    string
		payload *enginev1.ExecutionPayload
		header  *enginev1.ExecutionPayloadHeader
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
			header: func() *enginev1.ExecutionPayloadHeader {
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
			header: func() *enginev1.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.BlockHash = bytesutil.PadTo([]byte{'b'}, fieldparams.RootLength)
				return h
			}(),
			err: blocks.ErrInvalidPayloadBlockHash,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, _ := util.DeterministicGenesisStateBellatrix(t, 1)
			wrappedHeader, err := consensusblocks.WrappedExecutionPayloadHeader(tt.header)
			require.NoError(t, err)
			require.NoError(t, st.SetLatestExecutionPayloadHeader(wrappedHeader))
			wrappedPayload, err := consensusblocks.WrappedExecutionPayload(tt.payload)
			require.NoError(t, err)
			err = blocks.ValidatePayloadWhenMergeCompletes(st, wrappedPayload)
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
				h.PrevRandao = random
				h.Timestamp = uint64(ts.Unix())
				return h
			}(), err: nil,
		},
		{
			name:    "incorrect prev randao",
			payload: emptyPayload(),
			err:     blocks.ErrInvalidPayloadPrevRandao,
		},
		{
			name: "incorrect timestamp",
			payload: func() *enginev1.ExecutionPayload {
				h := emptyPayload()
				h.PrevRandao = random
				h.Timestamp = 1
				return h
			}(),
			err: blocks.ErrInvalidPayloadTimeStamp,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrappedPayload, err := consensusblocks.WrappedExecutionPayload(tt.payload)
			require.NoError(t, err)
			err = blocks.ValidatePayload(st, wrappedPayload)
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
				h.PrevRandao = random
				h.Timestamp = uint64(ts.Unix())
				return h
			}(), err: nil,
		},
		{
			name:    "incorrect prev randao",
			payload: emptyPayload(),
			err:     blocks.ErrInvalidPayloadPrevRandao,
		},
		{
			name: "incorrect timestamp",
			payload: func() *enginev1.ExecutionPayload {
				h := emptyPayload()
				h.PrevRandao = random
				h.Timestamp = 1
				return h
			}(),
			err: blocks.ErrInvalidPayloadTimeStamp,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrappedPayload, err := consensusblocks.WrappedExecutionPayload(tt.payload)
			require.NoError(t, err)
			st, err := blocks.ProcessPayload(st, wrappedPayload)
			if err != nil {
				require.Equal(t, tt.err.Error(), err.Error())
			} else {
				require.Equal(t, tt.err, err)
				want, err := consensusblocks.PayloadToHeader(wrappedPayload)
				require.Equal(t, tt.err, err)
				got, err := st.LatestExecutionPayloadHeader()
				require.NoError(t, err)
				require.DeepSSZEqual(t, want, got)
			}
		})
	}
}

func Test_ProcessPayloadHeader(t *testing.T) {
	st, _ := util.DeterministicGenesisStateBellatrix(t, 1)
	random, err := helpers.RandaoMix(st, time.CurrentEpoch(st))
	require.NoError(t, err)
	ts, err := slots.ToTime(st.GenesisTime(), st.Slot())
	require.NoError(t, err)
	tests := []struct {
		name   string
		header *enginev1.ExecutionPayloadHeader
		err    error
	}{
		{
			name: "process passes",
			header: func() *enginev1.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.PrevRandao = random
				h.Timestamp = uint64(ts.Unix())
				return h
			}(), err: nil,
		},
		{
			name:   "incorrect prev randao",
			header: emptyPayloadHeader(),
			err:    blocks.ErrInvalidPayloadPrevRandao,
		},
		{
			name: "incorrect timestamp",
			header: func() *enginev1.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.PrevRandao = random
				h.Timestamp = 1
				return h
			}(),
			err: blocks.ErrInvalidPayloadTimeStamp,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrappedHeader, err := consensusblocks.WrappedExecutionPayloadHeader(tt.header)
			require.NoError(t, err)
			st, err := blocks.ProcessPayloadHeader(st, wrappedHeader)
			if err != nil {
				require.Equal(t, tt.err.Error(), err.Error())
			} else {
				require.Equal(t, tt.err, err)
				got, err := st.LatestExecutionPayloadHeader()
				require.NoError(t, err)
				require.DeepSSZEqual(t, tt.header, got)
			}
		})
	}
}

func Test_ValidatePayloadHeader(t *testing.T) {
	st, _ := util.DeterministicGenesisStateBellatrix(t, 1)
	random, err := helpers.RandaoMix(st, time.CurrentEpoch(st))
	require.NoError(t, err)
	ts, err := slots.ToTime(st.GenesisTime(), st.Slot())
	require.NoError(t, err)
	tests := []struct {
		name   string
		header *enginev1.ExecutionPayloadHeader
		err    error
	}{
		{
			name: "process passes",
			header: func() *enginev1.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.PrevRandao = random
				h.Timestamp = uint64(ts.Unix())
				return h
			}(), err: nil,
		},
		{
			name:   "incorrect prev randao",
			header: emptyPayloadHeader(),
			err:    blocks.ErrInvalidPayloadPrevRandao,
		},
		{
			name: "incorrect timestamp",
			header: func() *enginev1.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.PrevRandao = random
				h.Timestamp = 1
				return h
			}(),
			err: blocks.ErrInvalidPayloadTimeStamp,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrappedHeader, err := consensusblocks.WrappedExecutionPayloadHeader(tt.header)
			require.NoError(t, err)
			err = blocks.ValidatePayloadHeader(st, wrappedHeader)
			require.Equal(t, tt.err, err)
		})
	}
}

func Test_ValidatePayloadHeaderWhenMergeCompletes(t *testing.T) {
	st, _ := util.DeterministicGenesisStateBellatrix(t, 1)
	emptySt := st.Copy()
	wrappedHeader, err := consensusblocks.WrappedExecutionPayloadHeader(&enginev1.ExecutionPayloadHeader{BlockHash: []byte{'a'}})
	require.NoError(t, err)
	require.NoError(t, st.SetLatestExecutionPayloadHeader(wrappedHeader))
	tests := []struct {
		name   string
		state  state.BeaconState
		header *enginev1.ExecutionPayloadHeader
		err    error
	}{
		{
			name: "no merge",
			header: func() *enginev1.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				return h
			}(),
			state: emptySt,
			err:   nil,
		},
		{
			name: "process passes",
			header: func() *enginev1.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.ParentHash = []byte{'a'}
				return h
			}(),
			state: st,
			err:   nil,
		},
		{
			name: "invalid block hash",
			header: func() *enginev1.ExecutionPayloadHeader {
				h := emptyPayloadHeader()
				h.ParentHash = []byte{'b'}
				return h
			}(),
			state: st,
			err:   blocks.ErrInvalidPayloadBlockHash,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrappedHeader, err := consensusblocks.WrappedExecutionPayloadHeader(tt.header)
			require.NoError(t, err)
			err = blocks.ValidatePayloadHeaderWhenMergeCompletes(tt.state, wrappedHeader)
			require.Equal(t, tt.err, err)
		})
	}
}

func Test_PayloadToHeader(t *testing.T) {
	p := emptyPayload()
	wrappedPayload, err := consensusblocks.WrappedExecutionPayload(p)
	require.NoError(t, err)
	h, err := consensusblocks.PayloadToHeader(wrappedPayload)
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
	p.PrevRandao = b
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
	require.DeepSSZEqual(t, h.ReceiptsRoot, make([]byte, fieldparams.RootLength))
	require.DeepSSZEqual(t, h.LogsBloom, make([]byte, fieldparams.LogsBloomLength))
	require.DeepSSZEqual(t, h.PrevRandao, make([]byte, fieldparams.RootLength))
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
	wrappedHeader, err := consensusblocks.WrappedExecutionPayloadHeader(emptyPayloadHeader())
	require.NoError(b, err)
	require.NoError(b, st.SetLatestExecutionPayloadHeader(wrappedHeader))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := blocks.IsMergeTransitionComplete(st)
		require.NoError(b, err)
	}
}

func emptyPayloadHeader() *enginev1.ExecutionPayloadHeader {
	return &enginev1.ExecutionPayloadHeader{
		ParentHash:       make([]byte, fieldparams.RootLength),
		FeeRecipient:     make([]byte, fieldparams.FeeRecipientLength),
		StateRoot:        make([]byte, fieldparams.RootLength),
		ReceiptsRoot:     make([]byte, fieldparams.RootLength),
		LogsBloom:        make([]byte, fieldparams.LogsBloomLength),
		PrevRandao:       make([]byte, fieldparams.RootLength),
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
		PrevRandao:    make([]byte, fieldparams.RootLength),
		BaseFeePerGas: make([]byte, fieldparams.RootLength),
		BlockHash:     make([]byte, fieldparams.RootLength),
		Transactions:  make([][]byte, 0),
		ExtraData:     make([]byte, 0),
	}
}
