package wrapper_test

import (
	"bytes"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/proto/interfaces"
	prysmv2 "github.com/prysmaticlabs/prysm/proto/prysm/v2"
	"github.com/prysmaticlabs/prysm/proto/prysm/v2/wrapper"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/version"
)

var (
	_ = interfaces.SignedBeaconBlock(&wrapper.AltairSignedBeaconBlock{})
	_ = interfaces.BeaconBlock(&wrapper.AltairBeaconBlock{})
	_ = interfaces.BeaconBlockBody(&wrapper.AltairBeaconBlockBody{})
)

func TestAltairSignedBeaconBlock_Signature(t *testing.T) {
	sig := []byte{0x11, 0x22}
	wsb := wrapper.WrappedAltairSignedBeaconBlock(&prysmv2.SignedBeaconBlock{Signature: sig})

	// Wrapped object returns signature.
	if !bytes.Equal(sig, wsb.Signature()) {
		t.Error("Wrong signature returned")
	}

	// Handles nil properly.
	wsb = wrapper.WrappedAltairSignedBeaconBlock(nil)
	if wsb.Signature() != nil {
		t.Error("Expected nil signature with nil underlying block")
	}
}

func TestAltairSignedBeaconBlock_Block(t *testing.T) {
	blk := &prysmv2.BeaconBlockAltair{Slot: 54}
	wsb := wrapper.WrappedAltairSignedBeaconBlock(&prysmv2.SignedBeaconBlock{Block: blk})

	// Wrapped signed block returns a beacon block.
	assert.DeepEqual(t, blk, wsb.Block().Proto())

	// Handles nil properly.
	wsb = wrapper.WrappedAltairSignedBeaconBlock(nil)
	if wsb.Block() != nil {
		t.Error("Expected nil block")
	}
}

func TestAltairSignedBeaconBlock_IsNil(t *testing.T) {
	wsb := wrapper.WrappedAltairSignedBeaconBlock(nil)
	assert.Equal(t, true, wsb.IsNil())
}

func TestAltairSignedBeaconBlock_Copy(t *testing.T) {
	t.Skip("TODO: Missing mutation evaluation helpers")
}

func TestAltairSignedBeaconBlock_Proto(t *testing.T) {
	sb := &prysmv2.SignedBeaconBlockAltair{
		Block:     &prysmv2.BeaconBlockAltair{Slot: 66},
		Signature: []byte{0x11, 0x22},
	}
	wsb := wrapper.WrappedAltairSignedBeaconBlock(sb)
	assert.Equal(t, sb, wsb.Proto())
}

func TestAltairSignedBeaconBlock_PbPhase0Block(t *testing.T) {
	wsb := wrapper.WrappedAltairSignedBeaconBlock(&prysmv2.SignedBeaconBlockAltair{})
	if _, err := wsb.PbPhase0Block(); err != wrapper.ErrUnsupportedPhase0Block {
		t.Errorf("Wrong error returned. Want %v got %v", wrapper.ErrUnsupportedPhase0Block, err)
	}
}

func TestAltairSignedBeaconBlock_PbAltairBlock(t *testing.T) {
	sb := &prysmv2.SignedBeaconBlockAltair{
		Block:     &prysmv2.BeaconBlockAltair{Slot: 66},
		Signature: []byte{0x11, 0x22},
	}
	wsb := wrapper.WrappedAltairSignedBeaconBlock(sb)

	got, err := wsb.PbAltairBlock()
	assert.NoError(t, err)
	assert.Equal(t, sb, got)
}

func TestAltairSignedBeaconBlock_MarshalSSZTo(t *testing.T) {
	t.Skip("TODO: Use altair generators in github.com/prysmaticlabs/prysm/shared/testutil")
}

func TestAltairSignedBeaconBlock_MarshalSSZ(t *testing.T) {
	t.Skip("TODO: Use altair generators in github.com/prysmaticlabs/prysm/shared/testutil")
}

func TestAltairSignedBeaconBlock_SizeSSZ(t *testing.T) {
	t.Skip("TODO: Use altair generators in github.com/prysmaticlabs/prysm/shared/testutil")
}

func TestAltairSignedBeaconBlock_UnmarshalSSZ(t *testing.T) {
	t.Skip("TODO: Use altair generators in github.com/prysmaticlabs/prysm/shared/testutil")
}

func TestAltairSignedBeaconBlock_Version(t *testing.T) {
	wsb := wrapper.WrappedAltairSignedBeaconBlock(&prysmv2.SignedBeaconBlockAltair{})
	assert.Equal(t, version.Altair, wsb.Version())
}

func TestAltairBeaconBlock_Slot(t *testing.T) {
	slot := types.Slot(546)
	wb := wrapper.WrappedAltairBeaconBlock(&prysmv2.BeaconBlockAltair{Slot: slot})

	// Returns correct slot.
	assert.Equal(t, slot, wb.Slot())

	// Handles nil.
	wb = wrapper.WrappedAltairBeaconBlock(nil)
	assert.Equal(t, types.Slot(0), wb.Slot())
}

func TestAltairBeaconBlock_ProposerIndex(t *testing.T) {
	pi := types.ValidatorIndex(555)
	wb := wrapper.WrappedAltairBeaconBlock(&prysmv2.BeaconBlockAltair{ProposerIndex: pi})

	// Returns correct index.
	assert.Equal(t, pi, wb.ProposerIndex())

	// Handles nil.
	wb = wrapper.WrappedAltairBeaconBlock(nil)
	assert.Equal(t, types.ValidatorIndex(0), wb.ProposerIndex())
}

func TestAltairBeaconBlock_ParentRoot(t *testing.T) {
	root := []byte{0xAA, 0xBF, 0x33, 0x01}
	wb := wrapper.WrappedAltairBeaconBlock(&prysmv2.BeaconBlockAltair{ParentRoot: root})

	// Returns correct root.
	assert.DeepEqual(t, root, wb.ParentRoot())

	// Handles nil.
	wb = wrapper.WrappedAltairBeaconBlock(nil)
	assert.Equal(t, nil, wb.ParentRoot())
}

func TestAltairBeaconBlock_StateRoot(t *testing.T) {
	root := []byte{0xAA, 0xBF, 0x33, 0x01}
	wb := wrapper.WrappedAltairBeaconBlock(&prysmv2.BeaconBlockAltair{StateRoot: root})

	assert.DeepEqual(t, root, wb.StateRoot())
}

func TestAltairBeaconBlock_Body(t *testing.T) {
	t.Fatal("TODO")
}

func TestAltairBeaconBlock_IsNil(t *testing.T) {
	wb := wrapper.WrappedAltairBeaconBlock(&prysmv2.BeaconBlockAltair{})
	assert.Equal(t, false, wb.IsNil())

	wb = wrapper.WrappedAltairBeaconBlock(nil)
	assert.Equal(t, true, wb.IsNil())
}

func TestAltairBeaconBlock_HashTreeRoot(t *testing.T) {
	t.Skip("TODO: Use altair generators in github.com/prysmaticlabs/prysm/shared/testutil")
}

func TestAltairBeaconBlock_Proto(t *testing.T) {
	blk := &prysmv2.BeaconBlockAltair{ProposerIndex: 234}
	wb := wrapper.WrappedAltairBeaconBlock(blk)

	assert.Equal(t, blk, wb.Proto())
}

func TestAltairBeaconBlock_MarshalSSZTo(t *testing.T) {
	t.Skip("TODO: Use altair generators in github.com/prysmaticlabs/prysm/shared/testutil")
}

func TestAltairBeaconBlock_MarshalSSZ(t *testing.T) {
	t.Skip("TODO: Use altair generators in github.com/prysmaticlabs/prysm/shared/testutil")
}

func TestAltairBeaconBlock_SizeSSZ(t *testing.T) {
	t.Skip("TODO: Use altair generators in github.com/prysmaticlabs/prysm/shared/testutil")
}

func TestAltairBeaconBlock_UnmarshalSSZ(t *testing.T) {
	t.Skip("TODO: Use altair generators in github.com/prysmaticlabs/prysm/shared/testutil")
}

func TestAltairBeaconBlock_Version(t *testing.T) {
	wb := wrapper.WrappedAltairBeaconBlock(&prysmv2.BeaconBlockAltair{})

	assert.Equal(t, version.Altair, wb.Version())
}

func TestAltairBeaconBlockBody_RandaoReveal(t *testing.T) {
	root := []byte{0xAA, 0xBF, 0x33, 0x01}
	wbb := wrapper.WrappedAltairBeaconBlockBody(&prysmv2.BeaconBlockBodyAltair{RandaoReveal: root})

	assert.DeepEqual(t, root, wbb.RandaoReveal())
}

func TestAltairBeaconBlockBody_Eth1Data(t *testing.T) {
	t.Fatal("TODO")
}

func TestAltairBeaconBlockBody_Graffiti(t *testing.T) {
	body := &prysmv2.BeaconBlockBodyAltair{Graffiti: []byte{0x66, 0xAA}}
	wbb := wrapper.WrappedAltairBeaconBlockBody(body)

	assert.DeepEqual(t, body.Graffiti, wbb.Graffiti())
}

func TestAltairBeaconBlockBody_ProposerSlashings(t *testing.T) {
	t.Fatal("TODO")
}

func TestAltairBeaconBlockBody_AttesterSlashings(t *testing.T) {
	t.Fatal("TODO")
}

func TestAltairBeaconBlockBody_Attestations(t *testing.T) {
	t.Fatal("TODO")
}

func TestAltairBeaconBlockBody_Deposits(t *testing.T) {
	t.Fatal("TODO")
}

func TestAltairBeaconBlockBody_VoluntaryExits(t *testing.T) {
	t.Fatal("TODO")
}

func TestAltairBeaconBlockBody_IsNil(t *testing.T) {
	wbb := wrapper.WrappedAltairBeaconBlockBody(&prysmv2.BeaconBlockBodyAltair{})
	assert.Equal(t, false, wbb.IsNil())

	wbb = wrapper.WrappedAltairBeaconBlockBody(nil)
	assert.Equal(t, true, wbb.IsNil())
}

func TestAltairBeaconBlockBody_HashTreeRoot(t *testing.T) {
	t.Skip("TODO: Use altair generators in github.com/prysmaticlabs/prysm/shared/testutil")
}

func TestAltairBeaconBlockBody_Proto(t *testing.T) {
	body := &prysmv2.BeaconBlockBodyAltair{Graffiti: []byte{0x66, 0xAA}}
	wbb := wrapper.WrappedAltairBeaconBlockBody(body)

	assert.Equal(t, body, wbb.Proto())
}
