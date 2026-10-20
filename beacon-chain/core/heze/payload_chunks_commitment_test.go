package heze_test

import (
	"bytes"
	"testing"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/gloas"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/heze"
	"github.com/OffchainLabs/prysm/v7/consensus-types/blocks"
	"github.com/OffchainLabs/prysm/v7/encoding/bytesutil"
	enginev1 "github.com/OffchainLabs/prysm/v7/proto/engine/v1"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/testing/require"
	"github.com/OffchainLabs/prysm/v7/testing/util"
)

func commitmentTestEnvelope(t *testing.T) *ethpb.ExecutionPayloadEnvelopeHeze {
	t.Helper()
	txs, err := enginev1.NewProgressiveTransactionList([][]byte{bytes.Repeat([]byte{0xaa}, 100)})
	require.NoError(t, err)
	return &ethpb.ExecutionPayloadEnvelopeHeze{
		Payload: &enginev1.ExecutionPayloadGloas{
			ParentHash:    bytesutil.PadTo([]byte("parent"), 32),
			FeeRecipient:  bytesutil.PadTo([]byte("fee"), 20),
			StateRoot:     bytesutil.PadTo([]byte("stateroot"), 32),
			ReceiptsRoot:  bytesutil.PadTo([]byte("receipts"), 32),
			LogsBloom:     make([]byte, 256),
			PrevRandao:    bytesutil.PadTo([]byte("randao"), 32),
			BlockNumber:   100,
			GasLimit:      30000000,
			GasUsed:       21000,
			Timestamp:     1000,
			BaseFeePerGas: bytesutil.PadTo([]byte{1}, 32),
			BlockHash:     bytesutil.PadTo([]byte("blockhash"), 32),
			Transactions:  txs,
			SlotNumber:    99,
		},
		ExecutionRequests:     &enginev1.ExecutionRequestsGloas{},
		BuilderIndex:          42,
		ParentBeaconBlockRoot: make([]byte, 32),
	}
}

// A zero signature is only a marker for chunk-distributed envelopes; it must
// route through the chunk commitment check, never bypass authentication.
func TestVerifyEnvelopeChunkCommitment(t *testing.T) {
	envelope := commitmentTestEnvelope(t)
	zeroSigned := &ethpb.SignedExecutionPayloadEnvelopeHeze{
		Message:   envelope,
		Signature: make([]byte, 96),
	}
	wrapped, err := blocks.WrappedROSignedExecutionPayloadEnvelopeHeze(zeroSigned, [32]byte{})
	require.NoError(t, err)
	require.Equal(t, true, heze.IsChunkAuthenticatedEnvelope(wrapped))

	t.Run("mismatched commitment is rejected", func(t *testing.T) {
		// The fixture bid commits to a zero payload_chunks_root, which can never
		// equal a real envelope's commitment.
		st, err := util.NewBeaconStateHeze()
		require.NoError(t, err)

		err = heze.VerifyEnvelopeChunkCommitment(st, wrapped)
		require.ErrorContains(t, "does not match the committed bid payload_chunks_root", err)

		_, err = gloas.ExecutionPayloadEnvelopeSignatureBatch(st, wrapped)
		require.ErrorContains(t, "does not match the committed bid payload_chunks_root", err)
	})

	t.Run("matching commitment authenticates", func(t *testing.T) {
		chunksRoot, err := heze.PayloadChunksRootForEnvelope(envelope)
		require.NoError(t, err)
		st, err := util.NewBeaconStateHeze(func(s *ethpb.BeaconStateHeze) error {
			s.LatestExecutionPayloadBid.PayloadChunksRoot = chunksRoot[:]
			return nil
		})
		require.NoError(t, err)

		require.NoError(t, heze.VerifyEnvelopeChunkCommitment(st, wrapped))

		batch, err := gloas.ExecutionPayloadEnvelopeSignatureBatch(st, wrapped)
		require.NoError(t, err)
		require.Equal(t, 0, len(batch.Signatures))
	})

	t.Run("real signature is not chunk-authenticated", func(t *testing.T) {
		realSigned := &ethpb.SignedExecutionPayloadEnvelopeHeze{
			Message:   envelope,
			Signature: bytes.Repeat([]byte{0xFF}, 96),
		}
		w, err := blocks.WrappedROSignedExecutionPayloadEnvelopeHeze(realSigned, [32]byte{})
		require.NoError(t, err)
		require.Equal(t, false, heze.IsChunkAuthenticatedEnvelope(w))
	})

	t.Run("gloas envelope is not a valid chunk commitment target", func(t *testing.T) {
		st, err := util.NewBeaconStateHeze()
		require.NoError(t, err)
		gloasSigned := &ethpb.SignedExecutionPayloadEnvelope{
			Message: &ethpb.ExecutionPayloadEnvelope{
				Payload:               envelope.Payload,
				ExecutionRequests:     envelope.ExecutionRequests,
				BuilderIndex:          envelope.BuilderIndex,
				BeaconBlockRoot:       make([]byte, 32),
				ParentBeaconBlockRoot: make([]byte, 32),
			},
			Signature: make([]byte, 96),
		}
		w, err := blocks.WrappedROSignedExecutionPayloadEnvelope(gloasSigned)
		require.NoError(t, err)
		err = heze.VerifyEnvelopeChunkCommitment(st, w)
		require.ErrorContains(t, "not a Heze envelope", err)
	})
}
