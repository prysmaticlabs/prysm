package altair_test

import (
	"bytes"
	"testing"

	typeerrors "github.com/prysmaticlabs/prysm/consensus-types/errors"
	"github.com/prysmaticlabs/prysm/consensus-types/forks/altair"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestSignedBeaconBlock_Signature(t *testing.T) {
	sig := []byte{0x11, 0x22}
	wsb, err := altair.WrappedSignedBeaconBlock(&ethpb.SignedBeaconBlockAltair{Block: &ethpb.BeaconBlockAltair{}, Signature: sig})
	require.NoError(t, err)

	if !bytes.Equal(sig, wsb.Signature()) {
		t.Error("Wrong signature returned")
	}
}

func TestSignedBeaconBlock_Block(t *testing.T) {
	blk := &ethpb.BeaconBlockAltair{Slot: 54}
	wsb, err := altair.WrappedSignedBeaconBlock(&ethpb.SignedBeaconBlockAltair{Block: blk})
	require.NoError(t, err)

	assert.DeepEqual(t, blk, wsb.Block().Proto())
}

func TestSignedBeaconBlock_IsNil(t *testing.T) {
	_, err := altair.WrappedSignedBeaconBlock(nil)
	require.Equal(t, typeerrors.ErrNilObjectWrapped, err)

	wsb, err := altair.WrappedSignedBeaconBlock(&ethpb.SignedBeaconBlockAltair{Block: &ethpb.BeaconBlockAltair{}})
	require.NoError(t, err)

	assert.Equal(t, false, wsb.IsNil())
}

func TestSignedBeaconBlock_Copy(t *testing.T) {
	t.Skip("TODO: Missing mutation evaluation helpers")
}

func TestSignedBeaconBlock_Proto(t *testing.T) {
	sb := &ethpb.SignedBeaconBlockAltair{
		Block:     &ethpb.BeaconBlockAltair{Slot: 66},
		Signature: []byte{0x11, 0x22},
	}
	wsb, err := altair.WrappedSignedBeaconBlock(sb)
	require.NoError(t, err)

	assert.Equal(t, sb, wsb.Proto())
}

func TestSignedBeaconBlock_PbPhase0Block(t *testing.T) {
	wsb, err := altair.WrappedSignedBeaconBlock(&ethpb.SignedBeaconBlockAltair{Block: &ethpb.BeaconBlockAltair{}})
	require.NoError(t, err)

	if _, err := wsb.PbPhase0Block(); err != typeerrors.ErrUnsupportedPhase0Block {
		t.Errorf("Wrong error returned. Want %v got %v", typeerrors.ErrUnsupportedPhase0Block, err)
	}
}

func TestSignedBeaconBlock_PbAltairBlock(t *testing.T) {
	sb := &ethpb.SignedBeaconBlockAltair{
		Block:     &ethpb.BeaconBlockAltair{Slot: 66},
		Signature: []byte{0x11, 0x22},
	}
	wsb, err := altair.WrappedSignedBeaconBlock(sb)
	require.NoError(t, err)

	got, err := wsb.PbAltairBlock()
	assert.NoError(t, err)
	assert.Equal(t, sb, got)
}

func TestSignedBeaconBlock_MarshalSSZTo(t *testing.T) {
	wsb, err := altair.WrappedSignedBeaconBlock(util.HydrateSignedBeaconBlockAltair(&ethpb.SignedBeaconBlockAltair{}))
	assert.NoError(t, err)

	var b []byte
	b, err = wsb.MarshalSSZTo(b)
	assert.NoError(t, err)
	assert.NotEqual(t, 0, len(b))
}

func TestSignedBeaconBlock_SSZ(t *testing.T) {
	wsb, err := altair.WrappedSignedBeaconBlock(util.HydrateSignedBeaconBlockAltair(&ethpb.SignedBeaconBlockAltair{}))
	assert.NoError(t, err)

	b, err := wsb.MarshalSSZ()
	assert.NoError(t, err)
	assert.NotEqual(t, 0, len(b))

	assert.NotEqual(t, 0, wsb.SizeSSZ())

	assert.NoError(t, wsb.UnmarshalSSZ(b))
}

func TestSignedBeaconBlock_Version(t *testing.T) {
	wsb, err := altair.WrappedSignedBeaconBlock(&ethpb.SignedBeaconBlockAltair{Block: &ethpb.BeaconBlockAltair{}})
	require.NoError(t, err)

	assert.Equal(t, version.Altair, wsb.Version())
}

func TestBeaconBlock_Slot(t *testing.T) {
	slot := types.Slot(546)
	wb, err := altair.WrappedBeaconBlock(&ethpb.BeaconBlockAltair{Slot: slot})
	require.NoError(t, err)

	assert.Equal(t, slot, wb.Slot())
}

func TestBeaconBlock_ProposerIndex(t *testing.T) {
	pi := types.ValidatorIndex(555)
	wb, err := altair.WrappedBeaconBlock(&ethpb.BeaconBlockAltair{ProposerIndex: pi})
	require.NoError(t, err)

	assert.Equal(t, pi, wb.ProposerIndex())
}

func TestBeaconBlock_ParentRoot(t *testing.T) {
	root := []byte{0xAA, 0xBF, 0x33, 0x01}
	wb, err := altair.WrappedBeaconBlock(&ethpb.BeaconBlockAltair{ParentRoot: root})
	require.NoError(t, err)

	assert.DeepEqual(t, root, wb.ParentRoot())
}

func TestBeaconBlock_StateRoot(t *testing.T) {
	root := []byte{0xAA, 0xBF, 0x33, 0x01}
	wb, err := altair.WrappedBeaconBlock(&ethpb.BeaconBlockAltair{StateRoot: root})
	require.NoError(t, err)

	assert.DeepEqual(t, root, wb.StateRoot())
}

func TestBeaconBlock_Body(t *testing.T) {
	body := &ethpb.BeaconBlockBodyAltair{Graffiti: []byte{0x44}}
	wb, err := altair.WrappedBeaconBlock(&ethpb.BeaconBlockAltair{Body: body})
	require.NoError(t, err)

	assert.Equal(t, body, wb.Body().Proto())
}

func TestBeaconBlock_IsNil(t *testing.T) {
	_, err := altair.WrappedBeaconBlock(nil)
	require.Equal(t, typeerrors.ErrNilObjectWrapped, err)

	wb, err := altair.WrappedBeaconBlock(&ethpb.BeaconBlockAltair{})
	require.NoError(t, err)

	assert.Equal(t, false, wb.IsNil())
}

func TestBeaconBlock_IsBlinded(t *testing.T) {
	wsb, err := altair.WrappedBeaconBlock(&ethpb.BeaconBlockAltair{})
	require.NoError(t, err)
	require.Equal(t, false, wsb.IsNil())
}

func TestBeaconBlock_HashTreeRoot(t *testing.T) {
	wb, err := altair.WrappedBeaconBlock(util.HydrateBeaconBlockAltair(&ethpb.BeaconBlockAltair{}))
	require.NoError(t, err)

	rt, err := wb.HashTreeRoot()
	assert.NoError(t, err)
	assert.NotEmpty(t, rt)
}

func TestBeaconBlock_Proto(t *testing.T) {
	blk := &ethpb.BeaconBlockAltair{ProposerIndex: 234}
	wb, err := altair.WrappedBeaconBlock(blk)
	require.NoError(t, err)

	assert.Equal(t, blk, wb.Proto())
}

func TestBeaconBlock_SSZ(t *testing.T) {
	wb, err := altair.WrappedBeaconBlock(util.HydrateBeaconBlockAltair(&ethpb.BeaconBlockAltair{}))
	assert.NoError(t, err)

	b, err := wb.MarshalSSZ()
	assert.NoError(t, err)
	assert.NotEqual(t, 0, len(b))

	assert.NotEqual(t, 0, wb.SizeSSZ())

	assert.NoError(t, wb.UnmarshalSSZ(b))
}

func TestBeaconBlock_Version(t *testing.T) {
	wb, err := altair.WrappedBeaconBlock(&ethpb.BeaconBlockAltair{})
	require.NoError(t, err)

	assert.Equal(t, version.Altair, wb.Version())
}

func TestBeaconBlockBody_RandaoReveal(t *testing.T) {
	root := []byte{0xAA, 0xBF, 0x33, 0x01}
	wbb, err := altair.WrappedBeaconBlockBody(&ethpb.BeaconBlockBodyAltair{RandaoReveal: root})
	require.NoError(t, err)

	assert.DeepEqual(t, root, wbb.RandaoReveal())
}

func TestBeaconBlockBody_Eth1Data(t *testing.T) {
	data := &ethpb.Eth1Data{}
	body := &ethpb.BeaconBlockBodyAltair{
		Eth1Data: data,
	}
	wbb, err := altair.WrappedBeaconBlockBody(body)
	require.NoError(t, err)
	assert.Equal(t, data, wbb.Eth1Data())
}

func TestBeaconBlockBody_Graffiti(t *testing.T) {
	graffiti := []byte{0x66, 0xAA}
	body := &ethpb.BeaconBlockBodyAltair{Graffiti: graffiti}
	wbb, err := altair.WrappedBeaconBlockBody(body)
	require.NoError(t, err)

	assert.DeepEqual(t, graffiti, wbb.Graffiti())
}

func TestBeaconBlockBody_ProposerSlashings(t *testing.T) {
	ps := []*ethpb.ProposerSlashing{
		{Header_1: &ethpb.SignedBeaconBlockHeader{
			Signature: []byte{0x11, 0x20},
		}},
	}
	body := &ethpb.BeaconBlockBodyAltair{ProposerSlashings: ps}
	wbb, err := altair.WrappedBeaconBlockBody(body)
	require.NoError(t, err)

	assert.DeepEqual(t, ps, wbb.ProposerSlashings())
}

func TestBeaconBlockBody_AttesterSlashings(t *testing.T) {
	as := []*ethpb.AttesterSlashing{
		{Attestation_1: &ethpb.IndexedAttestation{Signature: []byte{0x11}}},
	}
	body := &ethpb.BeaconBlockBodyAltair{AttesterSlashings: as}
	wbb, err := altair.WrappedBeaconBlockBody(body)
	require.NoError(t, err)

	assert.DeepEqual(t, as, wbb.AttesterSlashings())
}

func TestBeaconBlockBody_Attestations(t *testing.T) {
	atts := []*ethpb.Attestation{{Signature: []byte{0x88}}}

	body := &ethpb.BeaconBlockBodyAltair{Attestations: atts}
	wbb, err := altair.WrappedBeaconBlockBody(body)
	require.NoError(t, err)

	assert.DeepEqual(t, atts, wbb.Attestations())
}

func TestBeaconBlockBody_Deposits(t *testing.T) {
	deposits := []*ethpb.Deposit{
		{Proof: [][]byte{{0x54, 0x10}}},
	}
	body := &ethpb.BeaconBlockBodyAltair{Deposits: deposits}
	wbb, err := altair.WrappedBeaconBlockBody(body)
	require.NoError(t, err)

	assert.DeepEqual(t, deposits, wbb.Deposits())
}

func TestBeaconBlockBody_VoluntaryExits(t *testing.T) {
	exits := []*ethpb.SignedVoluntaryExit{
		{Exit: &ethpb.VoluntaryExit{Epoch: 54}},
	}
	body := &ethpb.BeaconBlockBodyAltair{VoluntaryExits: exits}
	wbb, err := altair.WrappedBeaconBlockBody(body)
	require.NoError(t, err)

	assert.DeepEqual(t, exits, wbb.VoluntaryExits())
}

func TestBeaconBlockBody_IsNil(t *testing.T) {
	_, err := altair.WrappedBeaconBlockBody(nil)
	require.Equal(t, typeerrors.ErrNilObjectWrapped, err)

	wbb, err := altair.WrappedBeaconBlockBody(&ethpb.BeaconBlockBodyAltair{})
	require.NoError(t, err)
	assert.Equal(t, false, wbb.IsNil())

}

func TestBeaconBlockBody_HashTreeRoot(t *testing.T) {
	wb, err := altair.WrappedBeaconBlockBody(util.HydrateBeaconBlockBodyAltair(&ethpb.BeaconBlockBodyAltair{}))
	assert.NoError(t, err)

	rt, err := wb.HashTreeRoot()
	assert.NoError(t, err)
	assert.NotEmpty(t, rt)
}

func TestBeaconBlockBody_Proto(t *testing.T) {
	body := &ethpb.BeaconBlockBodyAltair{Graffiti: []byte{0x66, 0xAA}}
	wbb, err := altair.WrappedBeaconBlockBody(body)
	require.NoError(t, err)

	assert.Equal(t, body, wbb.Proto())
}

func TestBeaconBlock_PbGenericBlock(t *testing.T) {
	abb := &ethpb.SignedBeaconBlockAltair{
		Block: util.HydrateBeaconBlockAltair(&ethpb.BeaconBlockAltair{}),
	}
	wsb, err := altair.WrappedSignedBeaconBlock(abb)
	require.NoError(t, err)

	got, err := wsb.PbGenericBlock()
	require.NoError(t, err)
	assert.Equal(t, abb, got.GetAltair())
}

func TestBeaconBlock_AsSignRequestObject(t *testing.T) {
	abb := util.HydrateBeaconBlockAltair(&ethpb.BeaconBlockAltair{})
	wsb, err := altair.WrappedBeaconBlock(abb)
	require.NoError(t, err)

	sro := wsb.AsSignRequestObject()
	got, ok := sro.(*validatorpb.SignRequest_BlockV2)
	require.Equal(t, true, ok, "Not a SignRequest_BlockV2")
	assert.Equal(t, abb, got.BlockV2)
}

func TestBeaconBlock_PbBlindedBellatrixBlock(t *testing.T) {
	sb := &ethpb.SignedBeaconBlockAltair{
		Block: &ethpb.BeaconBlockAltair{Slot: 66},
	}
	wsb, err := altair.WrappedSignedBeaconBlock(sb)
	require.NoError(t, err)
	_, err = wsb.PbBlindedBellatrixBlock()
	require.ErrorContains(t, "unsupported blinded bellatrix block", err)
}

func TestBeaconBlock_ExecutionPayloadHeader(t *testing.T) {
	sb := &ethpb.SignedBeaconBlockAltair{
		Block: &ethpb.BeaconBlockAltair{Slot: 66},
	}
	wsb, err := altair.WrappedSignedBeaconBlock(sb)
	require.NoError(t, err)
	_, err = wsb.Block().Body().ExecutionPayloadHeader()
	require.ErrorContains(t, "unsupported field for block type", err)
}
