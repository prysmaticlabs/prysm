package encoder_test

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	encoder "github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
	testpb "github.com/prysmaticlabs/prysm/proto/testing"
)

func TestSszNetworkEncoder_RoundTrip(t *testing.T) {
	e := &encoder.SszNetworkEncoder{UseSnappyCompression: false}
	testRoundTrip(t, e)
}

func TestSszNetworkEncoder_RoundTrip_Snappy(t *testing.T) {
	e := &encoder.SszNetworkEncoder{UseSnappyCompression: true}
	testRoundTrip(t, e)
}

func testRoundTrip(t *testing.T, e *encoder.SszNetworkEncoder) {
	msg := &testpb.TestSimpleMessage{
		Foo: []byte("fooooo"),
		Bar: 9001,
	}
	encoded, err := e.Encode(msg)
	if err != nil {
		t.Fatal(err)
	}
	decoded := &testpb.TestSimpleMessage{}
	if err := e.DecodeTo(encoded, decoded); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(decoded, msg) {
		t.Logf("decoded=%+v\n", decoded)
		t.Error("Decoded message is not the same as original")
	}
}
