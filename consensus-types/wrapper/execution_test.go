package wrapper_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/consensus-types/wrapper"
	"github.com/prysmaticlabs/prysm/encoding/ssz"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestPayloadToHeader(t *testing.T) {
	t.Run("no transactions should lead to a zero hash transactions root", func(t *testing.T) {
		payload := createWrappedPayload(t)
		wrapped, err := wrapper.PayloadToHeader(payload)
		require.NoError(t, err)
		require.DeepEqual(t, params.BeaconConfig().ZeroHash[:], wrapped.TransactionsRoot)
	})
	t.Run("transactions should lead to non-zero transactions root", func(t *testing.T) {
		payload := createEmptyPayload()
		tx := gethtypes.NewTransaction(
			1,
			common.BytesToAddress([]byte("hi")),
			big.NewInt(0),
			21000,
			big.NewInt(1e6),
			[]byte{},
		)
		enc, err := tx.MarshalBinary()
		require.NoError(t, err)
		payload.Transactions = [][]byte{enc}
		txsRoot, err := ssz.TransactionsRoot(payload.Transactions)
		require.NoError(t, err)
		wrappedPayload, err := wrapper.WrappedExecutionPayload(payload)
		require.NoError(t, err)
		wrappedHeader, err := wrapper.PayloadToHeader(wrappedPayload)
		require.NoError(t, err)
		require.DeepNotEqual(t, params.BeaconConfig().ZeroHash, wrappedHeader.TransactionsRoot)
		require.DeepEqual(t, txsRoot[:], wrappedHeader.TransactionsRoot)
	})
}

func TestWrapExecutionPayload(t *testing.T) {
	data := &enginev1.ExecutionPayload{GasUsed: 54}
	wsb, err := wrapper.WrappedExecutionPayload(data)
	require.NoError(t, err)

	assert.DeepEqual(t, data, wsb.Proto())
}

func TestWrapExecutionPayloadHeader(t *testing.T) {
	data := &enginev1.ExecutionPayloadHeader{GasUsed: 54}
	wsb, err := wrapper.WrappedExecutionPayloadHeader(data)
	require.NoError(t, err)

	assert.DeepEqual(t, data, wsb.Proto())
}

func TestWrapExecutionPayload_IsNil(t *testing.T) {
	_, err := wrapper.WrappedExecutionPayload(nil)
	require.Equal(t, wrapper.ErrNilObjectWrapped, err)

	data := &enginev1.ExecutionPayload{GasUsed: 54}
	wsb, err := wrapper.WrappedExecutionPayload(data)
	require.NoError(t, err)

	assert.Equal(t, false, wsb.IsNil())
}

func TestWrapExecutionPayloadHeader_IsNil(t *testing.T) {
	_, err := wrapper.WrappedExecutionPayloadHeader(nil)
	require.Equal(t, wrapper.ErrNilObjectWrapped, err)

	data := &enginev1.ExecutionPayloadHeader{GasUsed: 54}
	wsb, err := wrapper.WrappedExecutionPayloadHeader(data)
	require.NoError(t, err)

	assert.Equal(t, false, wsb.IsNil())
}

func TestWrapExecutionPayload_SSZ(t *testing.T) {
	wsb := createWrappedPayload(t)
	rt, err := wsb.HashTreeRoot()
	assert.NoError(t, err)
	assert.NotEmpty(t, rt)

	var b []byte
	b, err = wsb.MarshalSSZTo(b)
	assert.NoError(t, err)
	assert.NotEqual(t, 0, len(b))
	encoded, err := wsb.MarshalSSZ()
	require.NoError(t, err)
	assert.NotEqual(t, 0, wsb.SizeSSZ())
	assert.NoError(t, wsb.UnmarshalSSZ(encoded))
}

func TestWrapExecutionPayloadHeader_SSZ(t *testing.T) {
	wsb := createWrappedPayloadHeader(t)
	rt, err := wsb.HashTreeRoot()
	assert.NoError(t, err)
	assert.NotEmpty(t, rt)

	var b []byte
	b, err = wsb.MarshalSSZTo(b)
	assert.NoError(t, err)
	assert.NotEqual(t, 0, len(b))
	encoded, err := wsb.MarshalSSZ()
	require.NoError(t, err)
	assert.NotEqual(t, 0, wsb.SizeSSZ())
	assert.NoError(t, wsb.UnmarshalSSZ(encoded))
}

func createEmptyPayload() *enginev1.ExecutionPayload {
	return &enginev1.ExecutionPayload{
		ParentHash:    make([]byte, fieldparams.RootLength),
		FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
		StateRoot:     make([]byte, fieldparams.RootLength),
		ReceiptsRoot:  make([]byte, fieldparams.RootLength),
		LogsBloom:     make([]byte, fieldparams.LogsBloomLength),
		PrevRandao:    make([]byte, fieldparams.RootLength),
		BlockNumber:   0,
		GasLimit:      0,
		GasUsed:       0,
		Timestamp:     0,
		ExtraData:     make([]byte, 0),
		BaseFeePerGas: make([]byte, fieldparams.RootLength),
		BlockHash:     make([]byte, fieldparams.RootLength),
		Transactions:  make([][]byte, 0),
	}
}

func createWrappedPayload(t testing.TB) interfaces.ExecutionData {
	wsb, err := wrapper.WrappedExecutionPayload(createEmptyPayload())
	require.NoError(t, err)
	return wsb
}

func createWrappedPayloadHeader(t testing.TB) interfaces.ExecutionData {
	wsb, err := wrapper.WrappedExecutionPayloadHeader(&enginev1.ExecutionPayloadHeader{
		ParentHash:       make([]byte, fieldparams.RootLength),
		FeeRecipient:     make([]byte, fieldparams.FeeRecipientLength),
		StateRoot:        make([]byte, fieldparams.RootLength),
		ReceiptsRoot:     make([]byte, fieldparams.RootLength),
		LogsBloom:        make([]byte, fieldparams.LogsBloomLength),
		PrevRandao:       make([]byte, fieldparams.RootLength),
		BlockNumber:      0,
		GasLimit:         0,
		GasUsed:          0,
		Timestamp:        0,
		ExtraData:        make([]byte, 0),
		BaseFeePerGas:    make([]byte, fieldparams.RootLength),
		BlockHash:        make([]byte, fieldparams.RootLength),
		TransactionsRoot: make([]byte, fieldparams.RootLength),
	})
	require.NoError(t, err)
	return wsb
}
