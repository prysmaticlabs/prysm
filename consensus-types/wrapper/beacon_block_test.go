package wrapper_test

import (
	"testing"

	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/consensus-types/forks/bellatrix"
	"github.com/prysmaticlabs/prysm/consensus-types/wrapper"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestBuildSignedBeaconBlockFromExecutionPayload(t *testing.T) {
	t.Run("nil block check", func(t *testing.T) {
		_, err := wrapper.BuildSignedBeaconBlockFromExecutionPayload(nil, nil)
		require.ErrorIs(t, wrapper.ErrNilSignedBeaconBlock, err)
	})
	t.Run("unsupported field payload header", func(t *testing.T) {
		altairBlock := util.NewBeaconBlockAltair()
		blk, err := wrapper.WrappedSignedBeaconBlock(altairBlock)
		require.NoError(t, err)
		_, err = wrapper.BuildSignedBeaconBlockFromExecutionPayload(blk, nil)
		require.Equal(t, true, errors.Is(err, wrapper.ErrUnsupportedField))
	})
	t.Run("payload header root and payload root mismatch", func(t *testing.T) {
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
		header, err := bellatrix.PayloadToHeader(payload)
		require.NoError(t, err)
		blindedBlock := util.NewBlindedBeaconBlockBellatrix()

		// Modify the header.
		header.GasUsed += 1
		blindedBlock.Block.Body.ExecutionPayloadHeader = header

		blk, err := wrapper.WrappedSignedBeaconBlock(blindedBlock)
		require.NoError(t, err)
		_, err = wrapper.BuildSignedBeaconBlockFromExecutionPayload(blk, payload)
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
		header, err := bellatrix.PayloadToHeader(payload)
		require.NoError(t, err)
		blindedBlock := util.NewBlindedBeaconBlockBellatrix()
		blindedBlock.Block.Body.ExecutionPayloadHeader = header

		blk, err := wrapper.WrappedSignedBeaconBlock(blindedBlock)
		require.NoError(t, err)
		builtBlock, err := wrapper.BuildSignedBeaconBlockFromExecutionPayload(blk, payload)
		require.NoError(t, err)

		got, err := builtBlock.Block().Body().ExecutionPayload()
		require.NoError(t, err)
		require.DeepEqual(t, payload, got)
	})
}

func TestWrapSignedBlindedBeaconBlock(t *testing.T) {
	t.Run("nil block check", func(t *testing.T) {
		_, err := wrapper.BuildSignedBeaconBlockFromExecutionPayload(nil, nil)
		require.ErrorIs(t, wrapper.ErrNilSignedBeaconBlock, err)
	})
	t.Run("unsupported field execution payload", func(t *testing.T) {
		altairBlock := util.NewBeaconBlockAltair()
		blk, err := wrapper.WrappedSignedBeaconBlock(altairBlock)
		require.NoError(t, err)
		_, err = wrapper.BuildSignedBeaconBlockFromExecutionPayload(blk, nil)
		require.Equal(t, true, errors.Is(err, wrapper.ErrUnsupportedField))
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
		bellatrixBlk := util.NewBeaconBlockBellatrix()
		bellatrixBlk.Block.Body.ExecutionPayload = payload

		want, err := bellatrix.PayloadToHeader(payload)
		require.NoError(t, err)

		blk, err := wrapper.WrappedSignedBeaconBlock(bellatrixBlk)
		require.NoError(t, err)
		builtBlock, err := wrapper.WrapSignedBlindedBeaconBlock(blk)
		require.NoError(t, err)

		got, err := builtBlock.Block().Body().ExecutionPayloadHeader()
		require.NoError(t, err)
		require.DeepEqual(t, want, got)
	})
}

func TestWrappedSignedBeaconBlock(t *testing.T) {
	tests := []struct {
		name    string
		blk     interface{}
		wantErr bool
	}{
		{
			name:    "unsupported type",
			blk:     "not a beacon block",
			wantErr: true,
		},
		{
			name: "phase0",
			blk:  util.NewBeaconBlock(),
		},
		{
			name: "altair",
			blk:  util.NewBeaconBlockAltair(),
		},
		{
			name: "bellatrix",
			blk:  util.NewBeaconBlockBellatrix(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := wrapper.WrappedSignedBeaconBlock(tt.blk)
			if tt.wantErr {
				require.ErrorIs(t, err, wrapper.ErrUnsupportedSignedBeaconBlock)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
