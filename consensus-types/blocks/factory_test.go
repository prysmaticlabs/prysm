package blocks

import (
	"bytes"
	"errors"
	"testing"

	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func Test_NewSignedBeaconBlock(t *testing.T) {
	t.Run("GenericSignedBeaconBlock_Phase0", func(t *testing.T) {
		pb := &eth.GenericSignedBeaconBlock_Phase0{
			Phase0: &eth.SignedBeaconBlock{
				Block: &eth.BeaconBlock{
					Body: &eth.BeaconBlockBody{}}}}
		b, err := NewSignedBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Phase0, b.Version())
	})
	t.Run("SignedBeaconBlock", func(t *testing.T) {
		pb := &eth.SignedBeaconBlock{
			Block: &eth.BeaconBlock{
				Body: &eth.BeaconBlockBody{}}}
		b, err := NewSignedBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Phase0, b.Version())
	})
	t.Run("GenericSignedBeaconBlock_Altair", func(t *testing.T) {
		pb := &eth.GenericSignedBeaconBlock_Altair{
			Altair: &eth.SignedBeaconBlockAltair{
				Block: &eth.BeaconBlockAltair{
					Body: &eth.BeaconBlockBodyAltair{}}}}
		b, err := NewSignedBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Altair, b.Version())
	})
	t.Run("SignedBeaconBlockAltair", func(t *testing.T) {
		pb := &eth.SignedBeaconBlockAltair{
			Block: &eth.BeaconBlockAltair{
				Body: &eth.BeaconBlockBodyAltair{}}}
		b, err := NewSignedBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Altair, b.Version())
	})
	t.Run("GenericSignedBeaconBlock_Bellatrix", func(t *testing.T) {
		pb := &eth.GenericSignedBeaconBlock_Bellatrix{
			Bellatrix: &eth.SignedBeaconBlockBellatrix{
				Block: &eth.BeaconBlockBellatrix{
					Body: &eth.BeaconBlockBodyBellatrix{}}}}
		b, err := NewSignedBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Bellatrix, b.Version())
	})
	t.Run("SignedBeaconBlockBellatrix", func(t *testing.T) {
		pb := &eth.SignedBeaconBlockBellatrix{
			Block: &eth.BeaconBlockBellatrix{
				Body: &eth.BeaconBlockBodyBellatrix{}}}
		b, err := NewSignedBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Bellatrix, b.Version())
	})
	t.Run("GenericSignedBeaconBlock_BlindedBellatrix", func(t *testing.T) {
		pb := &eth.GenericSignedBeaconBlock_BlindedBellatrix{
			BlindedBellatrix: &eth.SignedBlindedBeaconBlockBellatrix{
				Block: &eth.BlindedBeaconBlockBellatrix{
					Body: &eth.BlindedBeaconBlockBodyBellatrix{}}}}
		b, err := NewSignedBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Bellatrix, b.Version())
		assert.Equal(t, true, b.IsBlinded())
	})
	t.Run("SignedBlindedBeaconBlockBellatrix", func(t *testing.T) {
		pb := &eth.SignedBlindedBeaconBlockBellatrix{
			Block: &eth.BlindedBeaconBlockBellatrix{
				Body: &eth.BlindedBeaconBlockBodyBellatrix{}}}
		b, err := NewSignedBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Bellatrix, b.Version())
		assert.Equal(t, true, b.IsBlinded())
	})
	t.Run("nil", func(t *testing.T) {
		_, err := NewSignedBeaconBlock(nil)
		assert.ErrorContains(t, "received nil object", err)
	})
	t.Run("unsupported type", func(t *testing.T) {
		_, err := NewSignedBeaconBlock(&bytes.Reader{})
		assert.ErrorContains(t, "unable to create block from type *bytes.Reader", err)
	})
}

func Test_NewBeaconBlock(t *testing.T) {
	t.Run("GenericBeaconBlock_Phase0", func(t *testing.T) {
		pb := &eth.GenericBeaconBlock_Phase0{Phase0: &eth.BeaconBlock{Body: &eth.BeaconBlockBody{}}}
		b, err := NewBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Phase0, b.Version())
	})
	t.Run("BeaconBlock", func(t *testing.T) {
		pb := &eth.BeaconBlock{Body: &eth.BeaconBlockBody{}}
		b, err := NewBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Phase0, b.Version())
	})
	t.Run("GenericBeaconBlock_Altair", func(t *testing.T) {
		pb := &eth.GenericBeaconBlock_Altair{Altair: &eth.BeaconBlockAltair{Body: &eth.BeaconBlockBodyAltair{}}}
		b, err := NewBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Altair, b.Version())
	})
	t.Run("BeaconBlockAltair", func(t *testing.T) {
		pb := &eth.BeaconBlockAltair{Body: &eth.BeaconBlockBodyAltair{}}
		b, err := NewBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Altair, b.Version())
	})
	t.Run("GenericBeaconBlock_Bellatrix", func(t *testing.T) {
		pb := &eth.GenericBeaconBlock_Bellatrix{Bellatrix: &eth.BeaconBlockBellatrix{Body: &eth.BeaconBlockBodyBellatrix{}}}
		b, err := NewBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Bellatrix, b.Version())
	})
	t.Run("BeaconBlockBellatrix", func(t *testing.T) {
		pb := &eth.BeaconBlockBellatrix{Body: &eth.BeaconBlockBodyBellatrix{}}
		b, err := NewBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Bellatrix, b.Version())
	})
	t.Run("GenericBeaconBlock_BlindedBellatrix", func(t *testing.T) {
		pb := &eth.GenericBeaconBlock_BlindedBellatrix{BlindedBellatrix: &eth.BlindedBeaconBlockBellatrix{Body: &eth.BlindedBeaconBlockBodyBellatrix{}}}
		b, err := NewBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Bellatrix, b.Version())
		assert.Equal(t, true, b.IsBlinded())
	})
	t.Run("BlindedBeaconBlockBellatrix", func(t *testing.T) {
		pb := &eth.BlindedBeaconBlockBellatrix{Body: &eth.BlindedBeaconBlockBodyBellatrix{}}
		b, err := NewBeaconBlock(pb)
		require.NoError(t, err)
		assert.Equal(t, version.Bellatrix, b.Version())
		assert.Equal(t, true, b.IsBlinded())
	})
	t.Run("nil", func(t *testing.T) {
		_, err := NewBeaconBlock(nil)
		assert.ErrorContains(t, "received nil object", err)
	})
	t.Run("unsupported type", func(t *testing.T) {
		_, err := NewBeaconBlock(&bytes.Reader{})
		assert.ErrorContains(t, "unable to create block from type *bytes.Reader", err)
	})
}

func Test_NewBeaconBlockBody(t *testing.T) {
	t.Run("BeaconBlockBody", func(t *testing.T) {
		pb := &eth.BeaconBlockBody{}
		i, err := NewBeaconBlockBody(pb)
		require.NoError(t, err)
		b, ok := i.(*BeaconBlockBody)
		require.Equal(t, true, ok)
		assert.Equal(t, version.Phase0, b.version)
	})
	t.Run("BeaconBlockBodyAltair", func(t *testing.T) {
		pb := &eth.BeaconBlockBodyAltair{}
		i, err := NewBeaconBlockBody(pb)
		require.NoError(t, err)
		b, ok := i.(*BeaconBlockBody)
		require.Equal(t, true, ok)
		assert.Equal(t, version.Altair, b.version)
	})
	t.Run("BeaconBlockBodyBellatrix", func(t *testing.T) {
		pb := &eth.BeaconBlockBodyBellatrix{}
		i, err := NewBeaconBlockBody(pb)
		require.NoError(t, err)
		b, ok := i.(*BeaconBlockBody)
		require.Equal(t, true, ok)
		assert.Equal(t, version.Bellatrix, b.version)
	})
	t.Run("BlindedBeaconBlockBodyBellatrix", func(t *testing.T) {
		pb := &eth.BlindedBeaconBlockBodyBellatrix{}
		i, err := NewBeaconBlockBody(pb)
		require.NoError(t, err)
		b, ok := i.(*BeaconBlockBody)
		require.Equal(t, true, ok)
		assert.Equal(t, version.Bellatrix, b.version)
		assert.Equal(t, true, b.isBlinded)
	})
	t.Run("nil", func(t *testing.T) {
		_, err := NewBeaconBlockBody(nil)
		assert.ErrorContains(t, "received nil object", err)
	})
	t.Run("unsupported type", func(t *testing.T) {
		_, err := NewBeaconBlockBody(&bytes.Reader{})
		assert.ErrorContains(t, "unable to create block body from type *bytes.Reader", err)
	})
}

func Test_BuildSignedBeaconBlock(t *testing.T) {
	sig := bytesutil.ToBytes96([]byte("signature"))
	t.Run("Phase0", func(t *testing.T) {
		b := &BeaconBlock{version: version.Phase0, body: &BeaconBlockBody{version: version.Phase0}}
		sb, err := BuildSignedBeaconBlock(b, sig[:])
		require.NoError(t, err)
		assert.DeepEqual(t, sig, sb.Signature())
		assert.Equal(t, version.Phase0, sb.Version())
	})
	t.Run("Altair", func(t *testing.T) {
		b := &BeaconBlock{version: version.Altair, body: &BeaconBlockBody{version: version.Altair}}
		sb, err := BuildSignedBeaconBlock(b, sig[:])
		require.NoError(t, err)
		assert.DeepEqual(t, sig, sb.Signature())
		assert.Equal(t, version.Altair, sb.Version())
	})
	t.Run("Bellatrix", func(t *testing.T) {
		b := &BeaconBlock{version: version.Bellatrix, body: &BeaconBlockBody{version: version.Bellatrix}}
		sb, err := BuildSignedBeaconBlock(b, sig[:])
		require.NoError(t, err)
		assert.DeepEqual(t, sig, sb.Signature())
		assert.Equal(t, version.Bellatrix, sb.Version())
	})
	t.Run("BellatrixBlind", func(t *testing.T) {
		b := &BeaconBlock{version: version.Bellatrix, body: &BeaconBlockBody{version: version.Bellatrix, isBlinded: true}}
		sb, err := BuildSignedBeaconBlock(b, sig[:])
		require.NoError(t, err)
		assert.DeepEqual(t, sig, sb.Signature())
		assert.Equal(t, version.Bellatrix, sb.Version())
		assert.Equal(t, true, sb.IsBlinded())
	})
}

func TestBuildSignedBeaconBlockFromExecutionPayload(t *testing.T) {
	t.Run("nil block check", func(t *testing.T) {
		_, err := BuildSignedBeaconBlockFromExecutionPayload(nil, nil)
		require.ErrorIs(t, ErrNilSignedBeaconBlock, err)
	})
	t.Run("unsupported field payload header", func(t *testing.T) {
		altairBlock := &eth.SignedBeaconBlockAltair{
			Block: &eth.BeaconBlockAltair{
				Body: &eth.BeaconBlockBodyAltair{}}}
		blk, err := NewSignedBeaconBlock(altairBlock)
		require.NoError(t, err)
		_, err = BuildSignedBeaconBlockFromExecutionPayload(blk, nil)
		require.Equal(t, true, errors.Is(err, ErrUnsupportedGetter))
	})
	t.Run("payload header root and payload root mismatch", func(t *testing.T) {
		blockHash := bytesutil.Bytes32(1)
		payload := &enginev1.ExecutionPayload{
			ParentHash:    make([]byte, fieldparams.RootLength),
			FeeRecipient:  make([]byte, 20),
			StateRoot:     make([]byte, fieldparams.RootLength),
			ReceiptsRoot:  make([]byte, fieldparams.RootLength),
			LogsBloom:     make([]byte, 256),
			PrevRandao:    make([]byte, fieldparams.RootLength),
			BaseFeePerGas: make([]byte, fieldparams.RootLength),
			BlockHash:     blockHash,
			Transactions:  make([][]byte, 0),
		}
		wrapped, err := WrappedExecutionPayload(payload)
		require.NoError(t, err)
		header, err := PayloadToHeader(wrapped)
		require.NoError(t, err)
		blindedBlock := &eth.SignedBlindedBeaconBlockBellatrix{
			Block: &eth.BlindedBeaconBlockBellatrix{
				Body: &eth.BlindedBeaconBlockBodyBellatrix{}}}

		// Modify the header.
		header.GasUsed += 1
		blindedBlock.Block.Body.ExecutionPayloadHeader = header

		blk, err := NewSignedBeaconBlock(blindedBlock)
		require.NoError(t, err)
		_, err = BuildSignedBeaconBlockFromExecutionPayload(blk, payload)
		require.ErrorContains(t, "roots do not match", err)
	})
	t.Run("ok", func(t *testing.T) {
		payload := &enginev1.ExecutionPayload{
			ParentHash:    make([]byte, fieldparams.RootLength),
			FeeRecipient:  make([]byte, 20),
			StateRoot:     make([]byte, fieldparams.RootLength),
			ReceiptsRoot:  make([]byte, fieldparams.RootLength),
			LogsBloom:     make([]byte, 256),
			PrevRandao:    make([]byte, fieldparams.RootLength),
			BaseFeePerGas: make([]byte, fieldparams.RootLength),
			BlockHash:     make([]byte, fieldparams.RootLength),
			Transactions:  make([][]byte, 0),
		}
		wrapped, err := WrappedExecutionPayload(payload)
		require.NoError(t, err)
		header, err := PayloadToHeader(wrapped)
		require.NoError(t, err)
		blindedBlock := &eth.SignedBlindedBeaconBlockBellatrix{
			Block: &eth.BlindedBeaconBlockBellatrix{
				Body: &eth.BlindedBeaconBlockBodyBellatrix{}}}
		blindedBlock.Block.Body.ExecutionPayloadHeader = header

		blk, err := NewSignedBeaconBlock(blindedBlock)
		require.NoError(t, err)
		builtBlock, err := BuildSignedBeaconBlockFromExecutionPayload(blk, payload)
		require.NoError(t, err)

		got, err := builtBlock.Block().Body().Execution()
		require.NoError(t, err)
		require.DeepEqual(t, payload, got.Proto())
	})
}
