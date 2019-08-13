package ssz_test

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	ssz "github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder/ssz"
	testpb "github.com/prysmaticlabs/prysm/proto/testing"
)

func TestSszNetworkEncoder_RoundTrip(t *testing.T) {
	e := &ssz.SszNetworkEncoder{UseSnappyCompression: false}
	msg := &testpb.TestMessage{
		Foo: "fooooo",
		Bar: "baaaar",
	}
	encoded, err := e.Encode(msg)
	if err != nil {
		t.Fatal(err)
	}
	decoded := &testpb.TestMessage{}
	if err := e.DecodeTo(encoded, decoded); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(decoded, msg) {
		t.Logf("decoded=%+v\n", decoded)
		t.Error("Decoded message is not the same as original")
	}
}

func TestSszNetworkEncoder_RoundTrip_Snappy(t *testing.T) {

}
