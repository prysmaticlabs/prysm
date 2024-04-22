package blocks_test

import (
	"math/big"
	"testing"

	ssz "github.com/prysmaticlabs/fastssz"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	consensus_types "github.com/prysmaticlabs/prysm/v5/consensus-types"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestWrapExecutionPayload(t *testing.T) {
	data := &enginev1.ExecutionPayload{GasUsed: 54}
	wsb, err := blocks.WrappedExecutionPayload(data)
	require.NoError(t, err)

	assert.DeepEqual(t, data, wsb.Proto())
}

func TestWrapExecutionPayloadHeader(t *testing.T) {
	data := &enginev1.ExecutionPayloadHeader{GasUsed: 54}
	wsb, err := blocks.WrappedExecutionPayloadHeader(data)
	require.NoError(t, err)

	assert.DeepEqual(t, data, wsb.Proto())
}

func TestWrapExecutionPayload_IsNil(t *testing.T) {
	_, err := blocks.WrappedExecutionPayload(nil)
	require.Equal(t, consensus_types.ErrNilObjectWrapped, err)

	data := &enginev1.ExecutionPayload{GasUsed: 54}
	wsb, err := blocks.WrappedExecutionPayload(data)
	require.NoError(t, err)

	assert.Equal(t, false, wsb.IsNil())
}

func TestWrapExecutionPayloadHeader_IsNil(t *testing.T) {
	_, err := blocks.WrappedExecutionPayloadHeader(nil)
	require.Equal(t, consensus_types.ErrNilObjectWrapped, err)

	data := &enginev1.ExecutionPayloadHeader{GasUsed: 54}
	wsb, err := blocks.WrappedExecutionPayloadHeader(data)
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

func TestWrapExecutionPayloadCapella(t *testing.T) {
	data := &enginev1.ExecutionPayloadCapella{
		ParentHash:    []byte("parenthash"),
		FeeRecipient:  []byte("feerecipient"),
		StateRoot:     []byte("stateroot"),
		ReceiptsRoot:  []byte("receiptsroot"),
		LogsBloom:     []byte("logsbloom"),
		PrevRandao:    []byte("prevrandao"),
		BlockNumber:   11,
		GasLimit:      22,
		GasUsed:       33,
		Timestamp:     44,
		ExtraData:     []byte("extradata"),
		BaseFeePerGas: []byte("basefeepergas"),
		BlockHash:     []byte("blockhash"),
		Transactions:  [][]byte{[]byte("transaction")},
		Withdrawals: []*enginev1.Withdrawal{{
			Index:          55,
			ValidatorIndex: 66,
			Address:        []byte("executionaddress"),
			Amount:         77,
		}},
	}
	payload, err := blocks.WrappedExecutionPayloadCapella(data, big.NewInt(10*1e9))
	require.NoError(t, err)
	wei, err := payload.ValueInWei()
	require.NoError(t, err)
	assert.Equal(t, 0, big.NewInt(10*1e9).Cmp(wei))
	gwei, err := payload.ValueInGwei()
	require.NoError(t, err)
	assert.Equal(t, uint64(10), gwei)

	assert.DeepEqual(t, data, payload.Proto())
}

func TestWrapExecutionPayloadHeaderCapella(t *testing.T) {
	data := &enginev1.ExecutionPayloadHeaderCapella{
		ParentHash:       []byte("parenthash"),
		FeeRecipient:     []byte("feerecipient"),
		StateRoot:        []byte("stateroot"),
		ReceiptsRoot:     []byte("receiptsroot"),
		LogsBloom:        []byte("logsbloom"),
		PrevRandao:       []byte("prevrandao"),
		BlockNumber:      11,
		GasLimit:         22,
		GasUsed:          33,
		Timestamp:        44,
		ExtraData:        []byte("extradata"),
		BaseFeePerGas:    []byte("basefeepergas"),
		BlockHash:        []byte("blockhash"),
		TransactionsRoot: []byte("transactionsroot"),
		WithdrawalsRoot:  []byte("withdrawalsroot"),
	}
	payload, err := blocks.WrappedExecutionPayloadHeaderCapella(data, big.NewInt(10*1e9))
	require.NoError(t, err)

	wei, err := payload.ValueInWei()
	require.NoError(t, err)
	assert.Equal(t, 0, big.NewInt(10*1e9).Cmp(wei))
	gwei, err := payload.ValueInGwei()
	require.NoError(t, err)
	assert.Equal(t, uint64(10), gwei)

	assert.DeepEqual(t, data, payload.Proto())

	txRoot, err := payload.TransactionsRoot()
	require.NoError(t, err)
	require.DeepEqual(t, txRoot, data.TransactionsRoot)

	wrRoot, err := payload.WithdrawalsRoot()
	require.NoError(t, err)
	require.DeepEqual(t, wrRoot, data.WithdrawalsRoot)
}

func TestWrapExecutionPayloadCapella_IsNil(t *testing.T) {
	_, err := blocks.WrappedExecutionPayloadCapella(nil, big.NewInt(0))
	require.Equal(t, consensus_types.ErrNilObjectWrapped, err)

	data := &enginev1.ExecutionPayloadCapella{GasUsed: 54}
	payload, err := blocks.WrappedExecutionPayloadCapella(data, big.NewInt(0))
	require.NoError(t, err)

	assert.Equal(t, false, payload.IsNil())
}

func TestWrapExecutionPayloadHeaderCapella_IsNil(t *testing.T) {
	_, err := blocks.WrappedExecutionPayloadHeaderCapella(nil, big.NewInt(0))
	require.Equal(t, consensus_types.ErrNilObjectWrapped, err)

	data := &enginev1.ExecutionPayloadHeaderCapella{GasUsed: 54}
	payload, err := blocks.WrappedExecutionPayloadHeaderCapella(data, big.NewInt(0))
	require.NoError(t, err)

	assert.Equal(t, false, payload.IsNil())
}

func TestWrapExecutionPayloadCapella_SSZ(t *testing.T) {
	payload := createWrappedPayloadCapella(t)
	rt, err := payload.HashTreeRoot()
	assert.NoError(t, err)
	assert.NotEmpty(t, rt)

	var b []byte
	b, err = payload.MarshalSSZTo(b)
	assert.NoError(t, err)
	assert.NotEqual(t, 0, len(b))
	encoded, err := payload.MarshalSSZ()
	require.NoError(t, err)
	assert.NotEqual(t, 0, payload.SizeSSZ())
	assert.NoError(t, payload.UnmarshalSSZ(encoded))
}

func TestWrapExecutionPayloadHeaderCapella_SSZ(t *testing.T) {
	payload := createWrappedPayloadHeaderCapella(t)
	rt, err := payload.HashTreeRoot()
	assert.NoError(t, err)
	assert.NotEmpty(t, rt)

	var b []byte
	b, err = payload.MarshalSSZTo(b)
	assert.NoError(t, err)
	assert.NotEqual(t, 0, len(b))
	encoded, err := payload.MarshalSSZ()
	require.NoError(t, err)
	assert.NotEqual(t, 0, payload.SizeSSZ())
	assert.NoError(t, payload.UnmarshalSSZ(encoded))
}

func Test_executionPayload_Pb(t *testing.T) {
	payload := createWrappedPayload(t)
	pb, err := payload.PbBellatrix()
	require.NoError(t, err)
	assert.DeepEqual(t, payload.Proto(), pb)

	_, err = payload.PbCapella()
	require.ErrorIs(t, err, consensus_types.ErrUnsupportedField)
}

func Test_executionPayloadHeader_Pb(t *testing.T) {
	payload := createWrappedPayloadHeader(t)
	_, err := payload.PbBellatrix()
	require.ErrorIs(t, err, consensus_types.ErrUnsupportedField)

	_, err = payload.PbCapella()
	require.ErrorIs(t, err, consensus_types.ErrUnsupportedField)
}

func Test_executionPayloadCapella_Pb(t *testing.T) {
	payload := createWrappedPayloadCapella(t)
	pb, err := payload.PbCapella()
	require.NoError(t, err)
	assert.DeepEqual(t, payload.Proto(), pb)

	_, err = payload.PbBellatrix()
	require.ErrorIs(t, err, consensus_types.ErrUnsupportedField)
}

func Test_executionPayloadHeaderCapella_Pb(t *testing.T) {
	payload := createWrappedPayloadHeaderCapella(t)
	_, err := payload.PbBellatrix()
	require.ErrorIs(t, err, consensus_types.ErrUnsupportedField)

	_, err = payload.PbCapella()
	require.ErrorIs(t, err, consensus_types.ErrUnsupportedField)
}

func TestWrapExecutionPayloadDeneb(t *testing.T) {
	data := &enginev1.ExecutionPayloadDeneb{
		ParentHash:    []byte("parenthash"),
		FeeRecipient:  []byte("feerecipient"),
		StateRoot:     []byte("stateroot"),
		ReceiptsRoot:  []byte("receiptsroot"),
		LogsBloom:     []byte("logsbloom"),
		PrevRandao:    []byte("prevrandao"),
		BlockNumber:   11,
		GasLimit:      22,
		GasUsed:       33,
		Timestamp:     44,
		ExtraData:     []byte("extradata"),
		BaseFeePerGas: []byte("basefeepergas"),
		BlockHash:     []byte("blockhash"),
		Transactions:  [][]byte{[]byte("transaction")},
		Withdrawals: []*enginev1.Withdrawal{{
			Index:          55,
			ValidatorIndex: 66,
			Address:        []byte("executionaddress"),
			Amount:         77,
		}},
		BlobGasUsed:   88,
		ExcessBlobGas: 99,
	}
	payload, err := blocks.WrappedExecutionPayloadDeneb(data, big.NewInt(420*1e9))
	require.NoError(t, err)
	wei, err := payload.ValueInWei()
	require.NoError(t, err)
	assert.Equal(t, 0, big.NewInt(420*1e9).Cmp(wei))
	gwei, err := payload.ValueInGwei()
	require.NoError(t, err)
	assert.Equal(t, uint64(420), gwei)

	g, err := payload.BlobGasUsed()
	require.NoError(t, err)
	require.DeepEqual(t, uint64(88), g)

	g, err = payload.ExcessBlobGas()
	require.NoError(t, err)
	require.DeepEqual(t, uint64(99), g)
}

func TestWrapExecutionPayloadHeaderDeneb(t *testing.T) {
	data := &enginev1.ExecutionPayloadHeaderDeneb{
		ParentHash:       []byte("parenthash"),
		FeeRecipient:     []byte("feerecipient"),
		StateRoot:        []byte("stateroot"),
		ReceiptsRoot:     []byte("receiptsroot"),
		LogsBloom:        []byte("logsbloom"),
		PrevRandao:       []byte("prevrandao"),
		BlockNumber:      11,
		GasLimit:         22,
		GasUsed:          33,
		Timestamp:        44,
		ExtraData:        []byte("extradata"),
		BaseFeePerGas:    []byte("basefeepergas"),
		BlockHash:        []byte("blockhash"),
		TransactionsRoot: []byte("transactionsroot"),
		WithdrawalsRoot:  []byte("withdrawalsroot"),
		BlobGasUsed:      88,
		ExcessBlobGas:    99,
	}
	payload, err := blocks.WrappedExecutionPayloadHeaderDeneb(data, big.NewInt(10*1e9))
	require.NoError(t, err)

	wei, err := payload.ValueInWei()
	require.NoError(t, err)
	assert.Equal(t, 0, big.NewInt(10*1e9).Cmp(wei))
	gwei, err := payload.ValueInGwei()
	require.NoError(t, err)
	assert.Equal(t, uint64(10), gwei)

	g, err := payload.BlobGasUsed()
	require.NoError(t, err)
	require.DeepEqual(t, uint64(88), g)

	g, err = payload.ExcessBlobGas()
	require.NoError(t, err)
	require.DeepEqual(t, uint64(99), g)
}

func TestWrapExecutionPayloadDeneb_SSZ(t *testing.T) {
	payload := createWrappedPayloadDeneb(t)
	rt, err := payload.HashTreeRoot()
	assert.NoError(t, err)
	assert.NotEmpty(t, rt)

	var b []byte
	b, err = payload.MarshalSSZTo(b)
	assert.NoError(t, err)
	assert.NotEqual(t, 0, len(b))
	encoded, err := payload.MarshalSSZ()
	require.NoError(t, err)
	assert.NotEqual(t, 0, payload.SizeSSZ())
	assert.NoError(t, payload.UnmarshalSSZ(encoded))
}

func TestWrapExecutionPayloadHeaderDeneb_SSZ(t *testing.T) {
	payload := createWrappedPayloadHeaderDeneb(t)
	rt, err := payload.HashTreeRoot()
	assert.NoError(t, err)
	assert.NotEmpty(t, rt)

	var b []byte
	b, err = payload.MarshalSSZTo(b)
	assert.NoError(t, err)
	assert.NotEqual(t, 0, len(b))
	encoded, err := payload.MarshalSSZ()
	require.NoError(t, err)
	assert.NotEqual(t, 0, payload.SizeSSZ())
	assert.NoError(t, payload.UnmarshalSSZ(encoded))
}

func createWrappedSignedPayloadHeader(t testing.TB) interfaces.ExecutionData {
	header, err := blocks.WrappedSignedExecutionPayloadHeader(&enginev1.SignedExecutionPayloadHeader{
		Message: &enginev1.ExecutionPayloadHeaderEPBS{
			ParentBlockHash:        bytesutil.PadTo([]byte("parentblockhash"), fieldparams.RootLength),
			ParentBlockRoot:        make([]byte, fieldparams.RootLength),
			BlockHash:              bytesutil.PadTo([]byte("blockhash"), fieldparams.RootLength),
			BuilderIndex:           0,
			Slot:                   0,
			Value:                  100, // Gwei
			BlobKzgCommitmentsRoot: make([]byte, fieldparams.RootLength),
		},
		Signature: make([]byte, fieldparams.BLSSignatureLength),
	})
	require.NoError(t, err)
	return header
}

func TestWrappedSignedExecutionPayloadHeader(t *testing.T) {
	h := createWrappedSignedPayloadHeader(t)
	assert.Equal(t, false, h.IsNil())
	m, err := h.MarshalSSZ()
	require.NoError(t, err)
	_, err = h.MarshalSSZTo(nil)
	require.NoError(t, err)
	n := h.SizeSSZ()
	require.NotEqual(t, 0, n)
	_, err = h.HashTreeRoot()
	require.NoError(t, err)
	err = h.HashTreeRootWith(ssz.DefaultHasherPool.Get())
	require.NoError(t, err)
	p := h.Proto()
	require.NotNil(t, p)
	parentHash := h.ParentHash()
	require.DeepEqual(t, parentHash, bytesutil.PadTo([]byte("parentblockhash"), fieldparams.RootLength))
	require.DeepEqual(t, []byte{}, h.FeeRecipient())
	require.DeepEqual(t, []byte{}, h.StateRoot())
	require.DeepEqual(t, []byte{}, h.ReceiptsRoot())
	require.DeepEqual(t, []byte{}, h.LogsBloom())
	require.DeepEqual(t, []byte{}, h.PrevRandao())
	require.DeepEqual(t, []byte{}, h.ExtraData())
	require.DeepEqual(t, []byte{}, h.BaseFeePerGas())
	require.Equal(t, uint64(0), h.BlockNumber())
	require.Equal(t, uint64(0), h.GasLimit())
	require.Equal(t, uint64(0), h.GasUsed())
	require.Equal(t, uint64(0), h.Timestamp())
	require.DeepEqual(t, h.BlockHash(), bytesutil.PadTo([]byte("blockhash"), fieldparams.RootLength))
	_, err = h.Transactions()
	require.ErrorIs(t, err, consensus_types.ErrUnsupportedField)
	_, err = h.TransactionsRoot()
	require.ErrorIs(t, err, consensus_types.ErrUnsupportedField)
	_, err = h.Withdrawals()
	require.ErrorIs(t, err, consensus_types.ErrUnsupportedField)
	_, err = h.WithdrawalsRoot()
	require.ErrorIs(t, err, consensus_types.ErrUnsupportedField)
	_, err = h.BlobGasUsed()
	require.ErrorIs(t, err, consensus_types.ErrUnsupportedField)
	_, err = h.ExcessBlobGas()
	require.ErrorIs(t, err, consensus_types.ErrUnsupportedField)
	_, err = h.PbBellatrix()
	require.ErrorIs(t, err, consensus_types.ErrUnsupportedField)
	_, err = h.PbCapella()
	require.ErrorIs(t, err, consensus_types.ErrUnsupportedField)
	_, err = h.PbDeneb()
	require.ErrorIs(t, err, consensus_types.ErrUnsupportedField)
	_, err = h.PbSignedExecutionPayloadEnvelope()
	require.ErrorIs(t, err, consensus_types.ErrUnsupportedField)
	gwei, err := h.ValueInGwei()
	require.NoError(t, err)
	require.Equal(t, uint64(100), gwei)
	wei, err := h.ValueInWei()
	require.NoError(t, err)
	require.Equal(t, 0, big.NewInt(100*1e9).Cmp(wei))
	require.Equal(t, true, h.IsBlinded())
	_, err = h.PbSignedExecutionPayloadHeader()
	require.NoError(t, err)
	require.NoError(t, h.UnmarshalSSZ(m)) // Testing this last because it modifies the object.
}

func createWrappedSignedPayloadEnvelope(t testing.TB) interfaces.ExecutionData {
	payload, err := blocks.WrappedSignedExecutionPayloadEnvelope(&enginev1.SignedExecutionPayloadEnvelope{
		Message: &enginev1.ExecutionPayloadEnvelope{
			Payload: &enginev1.ExecutionPayloadEPBS{
				ParentHash:    bytesutil.PadTo([]byte("parentblockhash"), fieldparams.RootLength),
				FeeRecipient:  bytesutil.PadTo([]byte("feerecipient"), fieldparams.FeeRecipientLength),
				StateRoot:     bytesutil.PadTo([]byte("stateroot"), fieldparams.RootLength),
				ReceiptsRoot:  bytesutil.PadTo([]byte("receiptsroot"), fieldparams.RootLength),
				LogsBloom:     bytesutil.PadTo([]byte("logsbloom"), fieldparams.LogsBloomLength),
				PrevRandao:    bytesutil.PadTo([]byte("prevrandao"), fieldparams.RootLength),
				BlockNumber:   1,
				GasLimit:      2,
				GasUsed:       3,
				Timestamp:     4,
				ExtraData:     bytesutil.PadTo([]byte("extradata"), fieldparams.RootLength),
				BaseFeePerGas: bytesutil.PadTo([]byte("basefeepergas"), fieldparams.RootLength),
				BlockHash:     bytesutil.PadTo([]byte("blockhash"), fieldparams.RootLength),
				Transactions:  [][]byte{{0xa}, {0xb}, {0xc}},
				Withdrawals:   make([]*enginev1.Withdrawal, 0),
				BlobGasUsed:   5,
				ExcessBlobGas: 6,
				InclusionListSummary: [][]byte{
					bytesutil.PadTo([]byte("alice"), fieldparams.FeeRecipientLength),
					bytesutil.PadTo([]byte("blob"), fieldparams.FeeRecipientLength),
					bytesutil.PadTo([]byte("charlie"), fieldparams.FeeRecipientLength),
				},
			},
			BuilderIndex:               0,
			BeaconBlockRoot:            bytesutil.PadTo([]byte("beaconblockroot"), fieldparams.RootLength),
			BlobKzgCommitments:         make([][]byte, 0),
			InclusionListProposerIndex: 0,
			InclusionListSlot:          0,
			InclusionListSignature:     make([]byte, fieldparams.BLSSignatureLength),
			PayloadWithheld:            false,
			StateRoot:                  make([]byte, fieldparams.RootLength),
		},
		Signature: make([]byte, fieldparams.BLSSignatureLength),
	})
	require.NoError(t, err)
	return payload
}

func TestWrappedSignedExecutionPayloadEnvelope(t *testing.T) {
	p := createWrappedSignedPayloadEnvelope(t)
	assert.Equal(t, false, p.IsNil())
	m, err := p.MarshalSSZ()
	require.NoError(t, err)
	_, err = p.MarshalSSZTo(nil)
	require.NoError(t, err)
	n := p.SizeSSZ()
	require.NotEqual(t, 0, n)
	_, err = p.HashTreeRoot()
	require.NoError(t, err)
	err = p.HashTreeRootWith(ssz.DefaultHasherPool.Get())
	require.NoError(t, err)
	proto := p.Proto()
	require.NotNil(t, proto)
	parentHash := p.ParentHash()
	require.DeepEqual(t, parentHash, bytesutil.PadTo([]byte("parentblockhash"), fieldparams.RootLength))
	feeRecipient := p.FeeRecipient()
	require.DeepEqual(t, feeRecipient, bytesutil.PadTo([]byte("feerecipient"), fieldparams.FeeRecipientLength))
	stateRoot := p.StateRoot()
	require.DeepEqual(t, stateRoot, bytesutil.PadTo([]byte("stateroot"), fieldparams.RootLength))
	receiptsRoot := p.ReceiptsRoot()
	require.DeepEqual(t, receiptsRoot, bytesutil.PadTo([]byte("receiptsroot"), fieldparams.RootLength))
	logsBloom := p.LogsBloom()
	require.DeepEqual(t, logsBloom, bytesutil.PadTo([]byte("logsbloom"), fieldparams.LogsBloomLength))
	prevRandao := p.PrevRandao()
	require.DeepEqual(t, prevRandao, bytesutil.PadTo([]byte("prevrandao"), fieldparams.RootLength))
	extraData := p.ExtraData()
	require.DeepEqual(t, extraData, bytesutil.PadTo([]byte("extradata"), fieldparams.RootLength))
	baseFeePerGas := p.BaseFeePerGas()
	require.DeepEqual(t, baseFeePerGas, bytesutil.PadTo([]byte("basefeepergas"), fieldparams.RootLength))
	require.Equal(t, uint64(1), p.BlockNumber())
	require.Equal(t, uint64(2), p.GasLimit())
	require.Equal(t, uint64(3), p.GasUsed())
	require.Equal(t, uint64(4), p.Timestamp())
	require.DeepEqual(t, p.BlockHash(), bytesutil.PadTo([]byte("blockhash"), fieldparams.RootLength))
	txs, err := p.Transactions()
	require.NoError(t, err)
	require.DeepEqual(t, txs, [][]byte{{0xa}, {0xb}, {0xc}})
	_, err = p.TransactionsRoot()
	require.ErrorIs(t, err, consensus_types.ErrUnsupportedField)
	wd, err := p.Withdrawals()
	require.NoError(t, err)
	require.DeepEqual(t, wd, make([]*enginev1.Withdrawal, 0))
	_, err = p.WithdrawalsRoot()
	require.ErrorIs(t, err, consensus_types.ErrUnsupportedField)
	b, err := p.BlobGasUsed()
	require.NoError(t, err)
	require.DeepEqual(t, uint64(5), b)
	e, err := p.ExcessBlobGas()
	require.NoError(t, err)
	require.DeepEqual(t, uint64(6), e)
	_, err = p.PbBellatrix()
	require.ErrorIs(t, err, consensus_types.ErrUnsupportedField)
	_, err = p.PbCapella()
	require.ErrorIs(t, err, consensus_types.ErrUnsupportedField)
	_, err = p.PbDeneb()
	require.ErrorIs(t, err, consensus_types.ErrUnsupportedField)
	_, err = p.PbSignedExecutionPayloadHeader()
	require.ErrorIs(t, err, consensus_types.ErrUnsupportedField)
	_, err = p.ValueInGwei()
	require.ErrorIs(t, err, consensus_types.ErrUnsupportedField)
	_, err = p.ValueInWei()
	require.ErrorIs(t, err, consensus_types.ErrUnsupportedField)
	require.Equal(t, false, p.IsBlinded())
	_, err = p.PbSignedExecutionPayloadEnvelope()
	require.NoError(t, err)
	require.NoError(t, p.UnmarshalSSZ(m)) // Testing this last because it modifies the object.
}

func createWrappedPayload(t testing.TB) interfaces.ExecutionData {
	wsb, err := blocks.WrappedExecutionPayload(&enginev1.ExecutionPayload{
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
	})
	require.NoError(t, err)
	return wsb
}

func createWrappedPayloadHeader(t testing.TB) interfaces.ExecutionData {
	wsb, err := blocks.WrappedExecutionPayloadHeader(&enginev1.ExecutionPayloadHeader{
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

func createWrappedPayloadCapella(t testing.TB) interfaces.ExecutionData {
	payload, err := blocks.WrappedExecutionPayloadCapella(&enginev1.ExecutionPayloadCapella{
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
		Withdrawals:   make([]*enginev1.Withdrawal, 0),
	}, big.NewInt(0))
	require.NoError(t, err)
	return payload
}

func createWrappedPayloadHeaderCapella(t testing.TB) interfaces.ExecutionData {
	payload, err := blocks.WrappedExecutionPayloadHeaderCapella(&enginev1.ExecutionPayloadHeaderCapella{
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
		WithdrawalsRoot:  make([]byte, fieldparams.RootLength),
	}, big.NewInt(0))
	require.NoError(t, err)
	return payload
}

func createWrappedPayloadDeneb(t testing.TB) interfaces.ExecutionData {
	payload, err := blocks.WrappedExecutionPayloadDeneb(&enginev1.ExecutionPayloadDeneb{
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
		Withdrawals:   make([]*enginev1.Withdrawal, 0),
		BlobGasUsed:   0,
		ExcessBlobGas: 0,
	}, big.NewInt(0))
	require.NoError(t, err)
	return payload
}

func createWrappedPayloadHeaderDeneb(t testing.TB) interfaces.ExecutionData {
	payload, err := blocks.WrappedExecutionPayloadHeaderDeneb(&enginev1.ExecutionPayloadHeaderDeneb{
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
		WithdrawalsRoot:  make([]byte, fieldparams.RootLength),
		BlobGasUsed:      0,
		ExcessBlobGas:    0,
	}, big.NewInt(0))
	require.NoError(t, err)
	return payload
}
