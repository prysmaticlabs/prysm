package phase0_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/consensus-types/forks/phase0"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestSignedBeaconBlock_Header(t *testing.T) {
	root := bytesutil.PadTo([]byte("root"), 32)
	signature := bytesutil.PadTo([]byte("sig"), 96)
	body := &ethpb.BeaconBlockBody{}
	body = util.HydrateBeaconBlockBody(body)
	bodyRoot, err := body.HashTreeRoot()
	require.NoError(t, err)
	block := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot:          1,
			ProposerIndex: 1,
			ParentRoot:    root,
			StateRoot:     root,
			Body:          body,
		},
		Signature: signature,
	}
	wrapped := phase0.WrappedSignedBeaconBlock(block)

	header, err := wrapped.Header()
	require.NoError(t, err)
	assert.Equal(t, types.ValidatorIndex(1), header.Header.ProposerIndex)
	assert.Equal(t, types.Slot(1), header.Header.Slot)
	assert.DeepEqual(t, bodyRoot[:], header.Header.BodyRoot)
	assert.DeepEqual(t, root, header.Header.StateRoot)
	assert.DeepEqual(t, root, header.Header.ParentRoot)
	assert.DeepEqual(t, signature, header.Signature)
}

func TestBeaconBlock_PbGenericBlock(t *testing.T) {
	abb := &ethpb.SignedBeaconBlock{
		Block: util.HydrateBeaconBlock(&ethpb.BeaconBlock{}),
	}
	wsb := phase0.WrappedSignedBeaconBlock(abb)

	got, err := wsb.PbGenericBlock()
	require.NoError(t, err)
	assert.Equal(t, abb, got.GetPhase0())
}

func TestBeaconBlock_AsSignRequestObject(t *testing.T) {
	abb := util.HydrateBeaconBlock(&ethpb.BeaconBlock{})
	wsb := phase0.WrappedBeaconBlock(abb)

	sro := wsb.AsSignRequestObject()
	got, ok := sro.(*validatorpb.SignRequest_Block)
	require.Equal(t, true, ok, "Not a SignRequest_Block")
	assert.Equal(t, abb, got.Block)
}

func TestBeaconBlock_PbBlindedBellatrixBlock(t *testing.T) {
	sb := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{Slot: 66},
	}
	wsb := phase0.WrappedSignedBeaconBlock(sb)
	_, err := wsb.PbBlindedBellatrixBlock()
	require.ErrorContains(t, "unsupported blinded bellatrix block", err)
}

func TestBeaconBlock_ExecutionPayloadHeader(t *testing.T) {
	sb := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{Slot: 66},
	}
	wsb := phase0.WrappedSignedBeaconBlock(sb)
	_, err := wsb.Block().Body().ExecutionPayloadHeader()
	require.ErrorContains(t, "unsupported field for block type", err)
}
