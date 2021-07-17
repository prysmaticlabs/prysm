package wrapper_test

import (
	"bytes"
	"testing"

	"github.com/prysmaticlabs/prysm/proto/interfaces"
	prysmv2 "github.com/prysmaticlabs/prysm/proto/prysm/v2"
	"github.com/prysmaticlabs/prysm/proto/prysm/v2/wrapper"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
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
	blk := &prysmv2.BeaconBlockAltair{}
	wsb := wrapper.WrappedAltairSignedBeaconBlock(&prysmv2.SignedBeaconBlock{Block: blk})

	// Wrapped signed block returns a wrapped beacon block.
	want := wrapper.WrappedAltairBeaconBlock(blk)
	assert.DeepEqual(t, want, wsb.Block())

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

func TestAltairSignedBeaconBlock_MarshalSSZ(t *testing.T) {
	t.Fatal("TODO")
}

func TestAltairSignedBeaconBlock_Proto(t *testing.T) {
	t.Fatal("TODO")
}

func TestAltairSignedBeaconBlock_PbPhase0Block(t *testing.T) {
	t.Fatal("TODO")
}

func TestAltairBeaconBlock_Slot(t *testing.T) {
	t.Fatal("TODO")
}

func TestAltairBeaconBlock_ProposerIndex(t *testing.T) {
	t.Fatal("TODO")
}

func TestAltairBeaconBlock_ParentRoot(t *testing.T) {
	t.Fatal("TODO")
}

func TestAltairBeaconBlock_StateRoot(t *testing.T) {
	t.Fatal("TODO")
}

func TestAltairBeaconBlock_Body(t *testing.T) {
	t.Fatal("TODO")
}

func TestAltairBeaconBlock_IsNil(t *testing.T) {
	t.Fatal("TODO")
}

func TestAltairBeaconBlock_HashTreeRoot(t *testing.T) {
	t.Fatal("TODO")
}

func TestAltairBeaconBlock_MarshalSSZ(t *testing.T) {
	t.Fatal("TODO")
}

func TestAltairBeaconBlock_Proto(t *testing.T) {
	t.Fatal("TODO")
}

func TestAltairBeaconBlockBody_RandaoReveal(t *testing.T) {
	t.Fatal("TODO")
}

func TestAltairBeaconBlockBody_Eth1Data(t *testing.T) {
	t.Fatal("TODO")
}

func TestAltairBeaconBlockBody_Graffiti(t *testing.T) {
	t.Fatal("TODO")
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
	t.Fatal("TODO")
}

func TestAltairBeaconBlockBody_HashTreeRoot(t *testing.T) {
	t.Fatal("TODO")
}

func TestAltairBeaconBlockBody_Proto(t *testing.T) {
	t.Fatal("TODO")
}
