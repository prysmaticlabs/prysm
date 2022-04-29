package bellatrix

import (
	"bytes"
	"testing"

	typeerrors "github.com/prysmaticlabs/prysm/consensus-types/errors"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestSignedBlindedBeaconBlock_Header(t *testing.T) {
	root := bytesutil.PadTo([]byte("root"), 32)
	signature := bytesutil.PadTo([]byte("sig"), 96)
	body := &ethpb.BlindedBeaconBlockBodyBellatrix{}
	body = util.HydrateBlindedBeaconBlockBodyBellatrix(body)
	bodyRoot, err := body.HashTreeRoot()
	require.NoError(t, err)
	block := &ethpb.SignedBlindedBeaconBlockBellatrix{
		Block: &ethpb.BlindedBeaconBlockBellatrix{
			Slot:          1,
			ProposerIndex: 1,
			ParentRoot:    root,
			StateRoot:     root,
			Body:          body,
		},
		Signature: signature,
	}
	wrapped, err := WrappedSignedBlindedBeaconBlock(block)
	require.NoError(t, err)

	header, err := wrapped.Header()
	require.NoError(t, err)
	assert.Equal(t, types.ValidatorIndex(1), header.Header.ProposerIndex)
	assert.Equal(t, types.Slot(1), header.Header.Slot)
	assert.DeepEqual(t, bodyRoot[:], header.Header.BodyRoot)
	assert.DeepEqual(t, root, header.Header.StateRoot)
	assert.DeepEqual(t, root, header.Header.ParentRoot)
	assert.DeepEqual(t, signature, header.Signature)
}

func TestSignedBlindedBeaconBlock_Signature(t *testing.T) {
	sig := []byte{0x11, 0x22}
	wsb, err := WrappedSignedBlindedBeaconBlock(&ethpb.SignedBlindedBeaconBlockBellatrix{Block: &ethpb.BlindedBeaconBlockBellatrix{}, Signature: sig})
	require.NoError(t, err)

	if !bytes.Equal(sig, wsb.Signature()) {
		t.Error("Wrong signature returned")
	}
}

func TestSignedBlindedBeaconBlock_Block(t *testing.T) {
	blk := &ethpb.BlindedBeaconBlockBellatrix{Slot: 54}
	wsb, err := WrappedSignedBlindedBeaconBlock(&ethpb.SignedBlindedBeaconBlockBellatrix{Block: blk})
	require.NoError(t, err)

	assert.DeepEqual(t, blk, wsb.Block().Proto())
}

func TestSignedBlindedBeaconBlock_IsNil(t *testing.T) {
	_, err := WrappedSignedBlindedBeaconBlock(nil)
	require.Equal(t, typeerrors.ErrNilObjectWrapped, err)

	wsb, err := WrappedSignedBlindedBeaconBlock(&ethpb.SignedBlindedBeaconBlockBellatrix{Block: &ethpb.BlindedBeaconBlockBellatrix{}})
	require.NoError(t, err)

	assert.Equal(t, false, wsb.IsNil())
}

func TestSignedBlindedBeaconBlock_Copy(t *testing.T) {
	t.Skip("TODO: Missing mutation evaluation helpers")
}

func TestSignedBlindedBeaconBlock_Proto(t *testing.T) {
	sb := &ethpb.SignedBlindedBeaconBlockBellatrix{
		Block:     &ethpb.BlindedBeaconBlockBellatrix{Slot: 66},
		Signature: []byte{0x11, 0x22},
	}
	wsb, err := WrappedSignedBlindedBeaconBlock(sb)
	require.NoError(t, err)

	assert.Equal(t, sb, wsb.Proto())
}

func TestSignedBlindedBeaconBlock_PbPhase0Block(t *testing.T) {
	wsb, err := WrappedSignedBlindedBeaconBlock(&ethpb.SignedBlindedBeaconBlockBellatrix{Block: &ethpb.BlindedBeaconBlockBellatrix{}})
	require.NoError(t, err)

	if _, err := wsb.PbPhase0Block(); err != typeerrors.ErrUnsupportedPhase0Block {
		t.Errorf("Wrong error returned. Want %v got %v", typeerrors.ErrUnsupportedPhase0Block, err)
	}
}

func TestSignedBlindedBeaconBlock_PbBellatrixBlock(t *testing.T) {
	sb := &ethpb.SignedBlindedBeaconBlockBellatrix{
		Block:     &ethpb.BlindedBeaconBlockBellatrix{Slot: 66},
		Signature: []byte{0x11, 0x22},
	}
	wsb, err := WrappedSignedBlindedBeaconBlock(sb)
	require.NoError(t, err)

	_, err = wsb.PbBellatrixBlock()
	require.ErrorContains(t, "unsupported bellatrix block", err)
}

func TestSignedBlindedBeaconBlock_PbBlindedBellatrixBlock(t *testing.T) {
	sb := &ethpb.SignedBlindedBeaconBlockBellatrix{
		Block:     &ethpb.BlindedBeaconBlockBellatrix{Slot: 66},
		Signature: []byte{0x11, 0x22},
	}
	wsb, err := WrappedSignedBlindedBeaconBlock(sb)
	require.NoError(t, err)

	got, err := wsb.PbBlindedBellatrixBlock()
	assert.NoError(t, err)
	assert.Equal(t, sb, got)
}

func TestSignedBlindedBeaconBlock_MarshalSSZTo(t *testing.T) {
	wsb, err := WrappedSignedBlindedBeaconBlock(util.HydrateSignedBlindedBeaconBlockBellatrix(&ethpb.SignedBlindedBeaconBlockBellatrix{}))
	assert.NoError(t, err)

	var b []byte
	b, err = wsb.MarshalSSZTo(b)
	assert.NoError(t, err)
	assert.NotEqual(t, 0, len(b))
}

func TestSignedBlindedBeaconBlock_SSZ(t *testing.T) {
	wsb, err := WrappedSignedBlindedBeaconBlock(util.HydrateSignedBlindedBeaconBlockBellatrix(&ethpb.SignedBlindedBeaconBlockBellatrix{}))
	assert.NoError(t, err)

	b, err := wsb.MarshalSSZ()
	assert.NoError(t, err)
	assert.NotEqual(t, 0, len(b))

	assert.NotEqual(t, 0, wsb.SizeSSZ())

	assert.NoError(t, wsb.UnmarshalSSZ(b))
}

func TestSignedBlindedBeaconBlock_Version(t *testing.T) {
	wsb, err := WrappedSignedBlindedBeaconBlock(&ethpb.SignedBlindedBeaconBlockBellatrix{Block: &ethpb.BlindedBeaconBlockBellatrix{}})
	require.NoError(t, err)

	assert.Equal(t, version.Bellatrix, wsb.Version())
}

func TestBlindedBeaconBlock_Slot(t *testing.T) {
	slot := types.Slot(546)
	wb, err := WrappedBlindedBeaconBlock(&ethpb.BlindedBeaconBlockBellatrix{Slot: slot})
	require.NoError(t, err)

	assert.Equal(t, slot, wb.Slot())
}

func TestBlindedBeaconBlock_ProposerIndex(t *testing.T) {
	pi := types.ValidatorIndex(555)
	wb, err := WrappedBlindedBeaconBlock(&ethpb.BlindedBeaconBlockBellatrix{ProposerIndex: pi})
	require.NoError(t, err)

	assert.Equal(t, pi, wb.ProposerIndex())
}

func TestBlindedBeaconBlock_ParentRoot(t *testing.T) {
	root := []byte{0xAA, 0xBF, 0x33, 0x01}
	wb, err := WrappedBlindedBeaconBlock(&ethpb.BlindedBeaconBlockBellatrix{ParentRoot: root})
	require.NoError(t, err)

	assert.DeepEqual(t, root, wb.ParentRoot())
}

func TestBlindedBeaconBlock_StateRoot(t *testing.T) {
	root := []byte{0xAA, 0xBF, 0x33, 0x01}
	wb, err := WrappedBlindedBeaconBlock(&ethpb.BlindedBeaconBlockBellatrix{StateRoot: root})
	require.NoError(t, err)

	assert.DeepEqual(t, root, wb.StateRoot())
}

func TestBlindedBeaconBlock_Body(t *testing.T) {
	body := &ethpb.BlindedBeaconBlockBodyBellatrix{Graffiti: []byte{0x44}}
	wb, err := WrappedBlindedBeaconBlock(&ethpb.BlindedBeaconBlockBellatrix{Body: body})
	require.NoError(t, err)

	assert.Equal(t, body, wb.Body().Proto())
}

func TestBlindedBeaconBlock_IsNil(t *testing.T) {
	_, err := WrappedBlindedBeaconBlockBody(nil)
	require.Equal(t, typeerrors.ErrNilObjectWrapped, err)

	wb, err := WrappedBlindedBeaconBlock(&ethpb.BlindedBeaconBlockBellatrix{})
	require.NoError(t, err)

	assert.Equal(t, false, wb.IsNil())
}

func TestBlindedBeaconBlock_IsBlinded(t *testing.T) {
	wb, err := WrappedBlindedBeaconBlock(&ethpb.BlindedBeaconBlockBellatrix{})
	require.NoError(t, err)

	assert.Equal(t, true, wb.IsBlinded())
}

func TestBlindedBeaconBlock_HashTreeRoot(t *testing.T) {
	wb, err := WrappedBlindedBeaconBlock(util.HydrateBlindedBeaconBlockBellatrix(&ethpb.BlindedBeaconBlockBellatrix{}))
	require.NoError(t, err)

	rt, err := wb.HashTreeRoot()
	assert.NoError(t, err)
	assert.NotEmpty(t, rt)
}

func TestBlindedBeaconBlock_Proto(t *testing.T) {
	blk := &ethpb.BlindedBeaconBlockBellatrix{ProposerIndex: 234}
	wb, err := WrappedBlindedBeaconBlock(blk)
	require.NoError(t, err)

	assert.Equal(t, blk, wb.Proto())
}

func TestBlindedBeaconBlock_SSZ(t *testing.T) {
	wb, err := WrappedBlindedBeaconBlock(util.HydrateBlindedBeaconBlockBellatrix(&ethpb.BlindedBeaconBlockBellatrix{}))
	assert.NoError(t, err)

	b, err := wb.MarshalSSZ()
	assert.NoError(t, err)
	assert.NotEqual(t, 0, len(b))

	assert.NotEqual(t, 0, wb.SizeSSZ())

	assert.NoError(t, wb.UnmarshalSSZ(b))
}

func TestBlindedBeaconBlock_Version(t *testing.T) {
	wb, err := WrappedBlindedBeaconBlock(&ethpb.BlindedBeaconBlockBellatrix{})
	require.NoError(t, err)

	assert.Equal(t, version.Bellatrix, wb.Version())
}

func TestBlindedBeaconBlockBody_RandaoReveal(t *testing.T) {
	root := []byte{0xAA, 0xBF, 0x33, 0x01}
	wbb, err := WrappedBlindedBeaconBlockBody(&ethpb.BlindedBeaconBlockBodyBellatrix{RandaoReveal: root})
	require.NoError(t, err)

	assert.DeepEqual(t, root, wbb.RandaoReveal())
}

func TestBlindedBeaconBlockBody_Eth1Data(t *testing.T) {
	data := &ethpb.Eth1Data{}
	body := &ethpb.BlindedBeaconBlockBodyBellatrix{
		Eth1Data: data,
	}
	wbb, err := WrappedBlindedBeaconBlockBody(body)
	require.NoError(t, err)
	assert.Equal(t, data, wbb.Eth1Data())
}

func TestBlindedBeaconBlockBody_Graffiti(t *testing.T) {
	graffiti := []byte{0x66, 0xAA}
	body := &ethpb.BlindedBeaconBlockBodyBellatrix{Graffiti: graffiti}
	wbb, err := WrappedBlindedBeaconBlockBody(body)
	require.NoError(t, err)

	assert.DeepEqual(t, graffiti, wbb.Graffiti())
}

func TestBlindedBeaconBlockBody_ProposerSlashings(t *testing.T) {
	ps := []*ethpb.ProposerSlashing{
		{Header_1: &ethpb.SignedBeaconBlockHeader{
			Signature: []byte{0x11, 0x20},
		}},
	}
	body := &ethpb.BlindedBeaconBlockBodyBellatrix{ProposerSlashings: ps}
	wbb, err := WrappedBlindedBeaconBlockBody(body)
	require.NoError(t, err)

	assert.DeepEqual(t, ps, wbb.ProposerSlashings())
}

func TestBlindedBeaconBlockBody_AttesterSlashings(t *testing.T) {
	as := []*ethpb.AttesterSlashing{
		{Attestation_1: &ethpb.IndexedAttestation{Signature: []byte{0x11}}},
	}
	body := &ethpb.BlindedBeaconBlockBodyBellatrix{AttesterSlashings: as}
	wbb, err := WrappedBlindedBeaconBlockBody(body)
	require.NoError(t, err)

	assert.DeepEqual(t, as, wbb.AttesterSlashings())
}

func TestBlindedBeaconBlockBody_Attestations(t *testing.T) {
	atts := []*ethpb.Attestation{{Signature: []byte{0x88}}}

	body := &ethpb.BlindedBeaconBlockBodyBellatrix{Attestations: atts}
	wbb, err := WrappedBlindedBeaconBlockBody(body)
	require.NoError(t, err)

	assert.DeepEqual(t, atts, wbb.Attestations())
}

func TestBlindedBeaconBlockBody_Deposits(t *testing.T) {
	deposits := []*ethpb.Deposit{
		{Proof: [][]byte{{0x54, 0x10}}},
	}
	body := &ethpb.BlindedBeaconBlockBodyBellatrix{Deposits: deposits}
	wbb, err := WrappedBlindedBeaconBlockBody(body)
	require.NoError(t, err)

	assert.DeepEqual(t, deposits, wbb.Deposits())
}

func TestBlindedBeaconBlockBody_VoluntaryExits(t *testing.T) {
	exits := []*ethpb.SignedVoluntaryExit{
		{Exit: &ethpb.VoluntaryExit{Epoch: 54}},
	}
	body := &ethpb.BlindedBeaconBlockBodyBellatrix{VoluntaryExits: exits}
	wbb, err := WrappedBlindedBeaconBlockBody(body)
	require.NoError(t, err)

	assert.DeepEqual(t, exits, wbb.VoluntaryExits())
}

func TestBlindedBeaconBlockBody_IsNil(t *testing.T) {
	_, err := WrappedBlindedBeaconBlockBody(nil)
	require.Equal(t, typeerrors.ErrNilObjectWrapped, err)

	wbb, err := WrappedBlindedBeaconBlockBody(&ethpb.BlindedBeaconBlockBodyBellatrix{})
	require.NoError(t, err)
	assert.Equal(t, false, wbb.IsNil())

}

func TestBlindedBeaconBlockBody_HashTreeRoot(t *testing.T) {
	wb, err := WrappedBlindedBeaconBlockBody(util.HydrateBlindedBeaconBlockBodyBellatrix(&ethpb.BlindedBeaconBlockBodyBellatrix{}))
	assert.NoError(t, err)

	rt, err := wb.HashTreeRoot()
	assert.NoError(t, err)
	assert.NotEmpty(t, rt)
}

func TestBlindedBeaconBlockBody_Proto(t *testing.T) {
	body := &ethpb.BlindedBeaconBlockBodyBellatrix{Graffiti: []byte{0x66, 0xAA}}
	wbb, err := WrappedBlindedBeaconBlockBody(body)
	require.NoError(t, err)

	assert.Equal(t, body, wbb.Proto())
}

func TestBlindedBeaconBlockBody_ExecutionPayloadHeader(t *testing.T) {
	payloads := &ethpb.ExecutionPayloadHeader{
		BlockNumber: 100,
	}
	body := &ethpb.BlindedBeaconBlockBodyBellatrix{ExecutionPayloadHeader: payloads}
	wbb, err := WrappedBlindedBeaconBlockBody(body)
	require.NoError(t, err)

	_, err = wbb.ExecutionPayload()
	require.ErrorContains(t, typeerrors.ErrUnsupportedField.Error(), err)
}

func TestBlindedBeaconBlock_PbGenericBlock(t *testing.T) {
	abb := &ethpb.SignedBlindedBeaconBlockBellatrix{
		Block: util.HydrateBlindedBeaconBlockBellatrix(&ethpb.BlindedBeaconBlockBellatrix{}),
	}
	wsb, err := WrappedSignedBlindedBeaconBlock(abb)
	require.NoError(t, err)

	got, err := wsb.PbGenericBlock()
	require.NoError(t, err)
	assert.Equal(t, abb, got.GetBlindedBellatrix())
}

func TestBlindedBeaconBlock_AsSignRequestObject(t *testing.T) {
	abb := util.HydrateBlindedBeaconBlockBellatrix(&ethpb.BlindedBeaconBlockBellatrix{})
	wsb, err := WrappedBlindedBeaconBlock(abb)
	require.NoError(t, err)

	sro := wsb.AsSignRequestObject()
	got, ok := sro.(*validatorpb.SignRequest_BlindedBlockV3)
	require.Equal(t, true, ok, "Not a SignRequest_BlockV3")
	assert.Equal(t, abb, got.BlindedBlockV3)
}
