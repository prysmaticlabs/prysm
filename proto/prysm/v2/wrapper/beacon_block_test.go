package wrapper_test

import (
	"bytes"
	"testing"

	"github.com/prysmaticlabs/prysm/proto/interfaces"
	prysmv2 "github.com/prysmaticlabs/prysm/proto/prysm/v2"
	"github.com/prysmaticlabs/prysm/proto/prysm/v2/wrapper"
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
	t.Fatal("TODO")
}
