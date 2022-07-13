package wrapper_test

import (
	"bytes"
	"testing"

	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/consensus-types/wrapper"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestBellatrixSignedBeaconBlock_Header(t *testing.T) {
	root := bytesutil.PadTo([]byte("root"), 32)
	signature := bytesutil.PadTo([]byte("sig"), 96)
	body := &ethpb.BeaconBlockBodyBellatrix{}
	body = util.HydrateBeaconBlockBodyBellatrix(body)
	bodyRoot, err := body.HashTreeRoot()
	require.NoError(t, err)
	block := &ethpb.SignedBeaconBlockBellatrix{
		Block: &ethpb.BeaconBlockBellatrix{
			Slot:          1,
			ProposerIndex: 1,
			ParentRoot:    root,
			StateRoot:     root,
			Body:          body,
		},
		Signature: signature,
	}
	wrapped, err := wrapper.WrappedSignedBeaconBlock(block)
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

func TestBellatrixSignedBeaconBlock_Signature(t *testing.T) {
	sig := []byte{0x11, 0x22}
	wsb, err := wrapper.WrappedSignedBeaconBlock(&ethpb.SignedBeaconBlockBellatrix{Block: &ethpb.BeaconBlockBellatrix{}, Signature: sig})
	require.NoError(t, err)

	if !bytes.Equal(sig, wsb.Signature()) {
		t.Error("Wrong signature returned")
	}
}

func TestBellatrixSignedBeaconBlock_Block(t *testing.T) {
	blk := &ethpb.BeaconBlockBellatrix{Slot: 54}
	wsb, err := wrapper.WrappedSignedBeaconBlock(&ethpb.SignedBeaconBlockBellatrix{Block: blk})
	require.NoError(t, err)

	assert.DeepEqual(t, blk, wsb.Block().Proto())
}

func TestBellatrixSignedBeaconBlock_IsNil(t *testing.T) {
	_, err := wrapper.WrappedSignedBeaconBlock(nil)
	require.Equal(t, wrapper.ErrNilObjectWrapped, err)

	wsb, err := wrapper.WrappedSignedBeaconBlock(&ethpb.SignedBeaconBlockBellatrix{Block: &ethpb.BeaconBlockBellatrix{}})
	require.NoError(t, err)

	assert.Equal(t, false, wsb.IsNil())
}

func TestBellatrixSignedBeaconBlock_Copy(t *testing.T) {
	t.Skip("TODO: Missing mutation evaluation helpers")
}

func TestBellatrixSignedBeaconBlock_Proto(t *testing.T) {
	sb := &ethpb.SignedBeaconBlockBellatrix{
		Block:     &ethpb.BeaconBlockBellatrix{Slot: 66},
		Signature: []byte{0x11, 0x22},
	}
	wsb, err := wrapper.WrappedSignedBeaconBlock(sb)
	require.NoError(t, err)

	assert.Equal(t, sb, wsb.Proto())
}

func TestBellatrixSignedBeaconBlock_PbPhase0Block(t *testing.T) {
	wsb, err := wrapper.WrappedSignedBeaconBlock(&ethpb.SignedBeaconBlockBellatrix{Block: &ethpb.BeaconBlockBellatrix{}})
	require.NoError(t, err)

	if _, err := wsb.PbPhase0Block(); err != wrapper.ErrUnsupportedPhase0Block {
		t.Errorf("Wrong error returned. Want %v got %v", wrapper.ErrUnsupportedPhase0Block, err)
	}
}

func TestBellatrixSignedBeaconBlock_PbBellatrixBlock(t *testing.T) {
	sb := &ethpb.SignedBeaconBlockBellatrix{
		Block:     &ethpb.BeaconBlockBellatrix{Slot: 66},
		Signature: []byte{0x11, 0x22},
	}
	wsb, err := wrapper.WrappedSignedBeaconBlock(sb)
	require.NoError(t, err)

	got, err := wsb.PbBellatrixBlock()
	assert.NoError(t, err)
	assert.Equal(t, sb, got)
}

func TestBellatrixSignedBeaconBlock_MarshalSSZTo(t *testing.T) {
	wsb, err := wrapper.WrappedSignedBeaconBlock(util.HydrateSignedBeaconBlockBellatrix(&ethpb.SignedBeaconBlockBellatrix{}))
	assert.NoError(t, err)

	var b []byte
	b, err = wsb.MarshalSSZTo(b)
	assert.NoError(t, err)
	assert.NotEqual(t, 0, len(b))
}

func TestBellatrixSignedBeaconBlock_SSZ(t *testing.T) {
	wsb, err := wrapper.WrappedSignedBeaconBlock(util.HydrateSignedBeaconBlockBellatrix(&ethpb.SignedBeaconBlockBellatrix{}))
	assert.NoError(t, err)

	b, err := wsb.MarshalSSZ()
	assert.NoError(t, err)
	assert.NotEqual(t, 0, len(b))

	assert.NotEqual(t, 0, wsb.SizeSSZ())

	assert.NoError(t, wsb.UnmarshalSSZ(b))
}

func TestBellatrixSignedBeaconBlock_Version(t *testing.T) {
	wsb, err := wrapper.WrappedSignedBeaconBlock(&ethpb.SignedBeaconBlockBellatrix{Block: &ethpb.BeaconBlockBellatrix{}})
	require.NoError(t, err)

	assert.Equal(t, version.Bellatrix, wsb.Version())
}

func TestBellatrixBeaconBlock_Slot(t *testing.T) {
	slot := types.Slot(546)
	wb, err := wrapper.WrappedBeaconBlock(&ethpb.BeaconBlockBellatrix{Slot: slot})
	require.NoError(t, err)

	assert.Equal(t, slot, wb.Slot())
}

func TestBellatrixBeaconBlock_ProposerIndex(t *testing.T) {
	pi := types.ValidatorIndex(555)
	wb, err := wrapper.WrappedBeaconBlock(&ethpb.BeaconBlockBellatrix{ProposerIndex: pi})
	require.NoError(t, err)

	assert.Equal(t, pi, wb.ProposerIndex())
}

func TestBellatrixBeaconBlock_ParentRoot(t *testing.T) {
	root := []byte{0xAA, 0xBF, 0x33, 0x01}
	wb, err := wrapper.WrappedBeaconBlock(&ethpb.BeaconBlockBellatrix{ParentRoot: root})
	require.NoError(t, err)

	assert.DeepEqual(t, root, wb.ParentRoot())
}

func TestBellatrixBeaconBlock_StateRoot(t *testing.T) {
	root := []byte{0xAA, 0xBF, 0x33, 0x01}
	wb, err := wrapper.WrappedBeaconBlock(&ethpb.BeaconBlockBellatrix{StateRoot: root})
	require.NoError(t, err)

	assert.DeepEqual(t, root, wb.StateRoot())
}

func TestBellatrixBeaconBlock_Body(t *testing.T) {
	body := &ethpb.BeaconBlockBodyBellatrix{Graffiti: []byte{0x44}}
	wb, err := wrapper.WrappedBeaconBlock(&ethpb.BeaconBlockBellatrix{Body: body})
	require.NoError(t, err)

	assert.Equal(t, body, wb.Body().Proto())
}

func TestBellatrixBeaconBlock_IsNil(t *testing.T) {
	_, err := wrapper.WrappedBeaconBlock(nil)
	require.Equal(t, wrapper.ErrNilObjectWrapped, err)

	wb, err := wrapper.WrappedBeaconBlock(&ethpb.BeaconBlockBellatrix{})
	require.NoError(t, err)

	assert.Equal(t, false, wb.IsNil())
}

func TestBellatrixBeaconBlock_IsBlinded(t *testing.T) {
	wsb, err := wrapper.WrappedBeaconBlock(&ethpb.BeaconBlockBellatrix{})
	require.NoError(t, err)
	require.Equal(t, false, wsb.IsNil())
}

func TestBellatrixBeaconBlock_HashTreeRoot(t *testing.T) {
	wb, err := wrapper.WrappedBeaconBlock(util.HydrateBeaconBlockBellatrix(&ethpb.BeaconBlockBellatrix{}))
	require.NoError(t, err)

	rt, err := wb.HashTreeRoot()
	assert.NoError(t, err)
	assert.NotEmpty(t, rt)
}

func TestBellatrixBeaconBlock_Proto(t *testing.T) {
	blk := &ethpb.BeaconBlockBellatrix{ProposerIndex: 234}
	wb, err := wrapper.WrappedBeaconBlock(blk)
	require.NoError(t, err)

	assert.Equal(t, blk, wb.Proto())
}

func TestBellatrixBeaconBlock_SSZ(t *testing.T) {
	wb, err := wrapper.WrappedBeaconBlock(util.HydrateBeaconBlockBellatrix(&ethpb.BeaconBlockBellatrix{}))
	assert.NoError(t, err)

	b, err := wb.MarshalSSZ()
	assert.NoError(t, err)
	assert.NotEqual(t, 0, len(b))

	assert.NotEqual(t, 0, wb.SizeSSZ())

	assert.NoError(t, wb.UnmarshalSSZ(b))
}

func TestBellatrixBeaconBlock_Version(t *testing.T) {
	wb, err := wrapper.WrappedBeaconBlock(&ethpb.BeaconBlockBellatrix{})
	require.NoError(t, err)

	assert.Equal(t, version.Bellatrix, wb.Version())
}

func TestBellatrixBeaconBlockBody_RandaoReveal(t *testing.T) {
	root := []byte{0xAA, 0xBF, 0x33, 0x01}
	wbb, err := wrapper.WrappedBeaconBlockBody(&ethpb.BeaconBlockBodyBellatrix{RandaoReveal: root})
	require.NoError(t, err)

	assert.DeepEqual(t, root, wbb.RandaoReveal())
}

func TestBellatrixBeaconBlockBody_Eth1Data(t *testing.T) {
	data := &ethpb.Eth1Data{}
	body := &ethpb.BeaconBlockBodyBellatrix{
		Eth1Data: data,
	}
	wbb, err := wrapper.WrappedBeaconBlockBody(body)
	require.NoError(t, err)
	assert.Equal(t, data, wbb.Eth1Data())
}

func TestBellatrixBeaconBlockBody_Graffiti(t *testing.T) {
	graffiti := []byte{0x66, 0xAA}
	body := &ethpb.BeaconBlockBodyBellatrix{Graffiti: graffiti}
	wbb, err := wrapper.WrappedBeaconBlockBody(body)
	require.NoError(t, err)

	assert.DeepEqual(t, graffiti, wbb.Graffiti())
}

func TestBellatrixBeaconBlockBody_ProposerSlashings(t *testing.T) {
	ps := []*ethpb.ProposerSlashing{
		{Header_1: &ethpb.SignedBeaconBlockHeader{
			Signature: []byte{0x11, 0x20},
		}},
	}
	body := &ethpb.BeaconBlockBodyBellatrix{ProposerSlashings: ps}
	wbb, err := wrapper.WrappedBeaconBlockBody(body)
	require.NoError(t, err)

	assert.DeepEqual(t, ps, wbb.ProposerSlashings())
}

func TestBellatrixBeaconBlockBody_AttesterSlashings(t *testing.T) {
	as := []*ethpb.AttesterSlashing{
		{Attestation_1: &ethpb.IndexedAttestation{Signature: []byte{0x11}}},
	}
	body := &ethpb.BeaconBlockBodyBellatrix{AttesterSlashings: as}
	wbb, err := wrapper.WrappedBeaconBlockBody(body)
	require.NoError(t, err)

	assert.DeepEqual(t, as, wbb.AttesterSlashings())
}

func TestBellatrixBeaconBlockBody_Attestations(t *testing.T) {
	atts := []*ethpb.Attestation{{Signature: []byte{0x88}}}

	body := &ethpb.BeaconBlockBodyBellatrix{Attestations: atts}
	wbb, err := wrapper.WrappedBeaconBlockBody(body)
	require.NoError(t, err)

	assert.DeepEqual(t, atts, wbb.Attestations())
}

func TestBellatrixBeaconBlockBody_Deposits(t *testing.T) {
	deposits := []*ethpb.Deposit{
		{Proof: [][]byte{{0x54, 0x10}}},
	}
	body := &ethpb.BeaconBlockBodyBellatrix{Deposits: deposits}
	wbb, err := wrapper.WrappedBeaconBlockBody(body)
	require.NoError(t, err)

	assert.DeepEqual(t, deposits, wbb.Deposits())
}

func TestBellatrixBeaconBlockBody_VoluntaryExits(t *testing.T) {
	exits := []*ethpb.SignedVoluntaryExit{
		{Exit: &ethpb.VoluntaryExit{Epoch: 54}},
	}
	body := &ethpb.BeaconBlockBodyBellatrix{VoluntaryExits: exits}
	wbb, err := wrapper.WrappedBeaconBlockBody(body)
	require.NoError(t, err)

	assert.DeepEqual(t, exits, wbb.VoluntaryExits())
}

func TestBellatrixBeaconBlockBody_IsNil(t *testing.T) {
	_, err := wrapper.WrappedBeaconBlockBody(nil)
	require.Equal(t, wrapper.ErrNilObjectWrapped, err)

	wbb, err := wrapper.WrappedBeaconBlockBody(&ethpb.BeaconBlockBodyBellatrix{})
	require.NoError(t, err)
	assert.Equal(t, false, wbb.IsNil())

}

func TestBellatrixBeaconBlockBody_HashTreeRoot(t *testing.T) {
	wb, err := wrapper.WrappedBeaconBlockBody(util.HydrateBeaconBlockBodyBellatrix(&ethpb.BeaconBlockBodyBellatrix{}))
	assert.NoError(t, err)

	rt, err := wb.HashTreeRoot()
	assert.NoError(t, err)
	assert.NotEmpty(t, rt)
}

func TestBellatrixBeaconBlockBody_Proto(t *testing.T) {
	body := &ethpb.BeaconBlockBodyBellatrix{Graffiti: []byte{0x66, 0xAA}}
	wbb, err := wrapper.WrappedBeaconBlockBody(body)
	require.NoError(t, err)

	assert.Equal(t, body, wbb.Proto())
}

func TestBellatrixBeaconBlockBody_ExecutionPayload(t *testing.T) {
	payloads := &enginev1.ExecutionPayload{
		BlockNumber: 100,
	}
	body := &ethpb.BeaconBlockBodyBellatrix{ExecutionPayload: payloads}
	wbb, err := wrapper.WrappedBeaconBlockBody(body)
	require.NoError(t, err)

	got, err := wbb.Execution()
	require.NoError(t, err)
	assert.DeepEqual(t, payloads, got.Proto())
}

func TestBellatrixBeaconBlock_PbGenericBlock(t *testing.T) {
	abb := &ethpb.SignedBeaconBlockBellatrix{
		Block: util.HydrateBeaconBlockBellatrix(&ethpb.BeaconBlockBellatrix{}),
	}
	wsb, err := wrapper.WrappedSignedBeaconBlock(abb)
	require.NoError(t, err)

	got, err := wsb.PbGenericBlock()
	require.NoError(t, err)
	assert.Equal(t, abb, got.GetBellatrix())
}

func TestBellatrixBeaconBlock_AsSignRequestObject(t *testing.T) {
	abb := util.HydrateBeaconBlockBellatrix(&ethpb.BeaconBlockBellatrix{})
	wsb, err := wrapper.WrappedBeaconBlock(abb)
	require.NoError(t, err)

	sro := wsb.AsSignRequestObject()
	got, ok := sro.(*validatorpb.SignRequest_BlockV3)
	require.Equal(t, true, ok, "Not a SignRequest_BlockV3")
	assert.Equal(t, abb, got.BlockV3)
}

func TestBellatrixBeaconBlock_PbBlindedBellatrixBlock(t *testing.T) {
	sb := &ethpb.SignedBeaconBlockBellatrix{
		Block: &ethpb.BeaconBlockBellatrix{Slot: 66},
	}
	wsb, err := wrapper.WrappedSignedBeaconBlock(sb)
	require.NoError(t, err)
	_, err = wsb.PbBlindedBellatrixBlock()
	require.ErrorContains(t, "unsupported blinded bellatrix block", err)
}

func TestBellatrixBeaconBlock_ExecutionPayloadHeader(t *testing.T) {
	sb := &ethpb.SignedBeaconBlockBellatrix{
		Block: &ethpb.BeaconBlockBellatrix{Slot: 66, Body: &ethpb.BeaconBlockBodyBellatrix{
			ExecutionPayload: &enginev1.ExecutionPayload{},
		}},
	}
	wsb, err := wrapper.WrappedSignedBeaconBlock(sb)
	require.NoError(t, err)
	exec, err := wsb.Block().Body().Execution()
	require.NoError(t, err)
	_, err = exec.TransactionsRoot()
	require.ErrorContains(t, "unsupported field for block type", err)
}
