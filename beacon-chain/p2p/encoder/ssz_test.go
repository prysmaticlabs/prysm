package encoder_test

import (
	"bytes"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
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
	buf := new(bytes.Buffer)
	msg := &testpb.TestSimpleMessage{
		Foo: []byte("fooooo"),
		Bar: 9001,
	}
	_, err := e.Encode(buf, msg)
	if err != nil {
		t.Fatal(err)
	}
	decoded := &testpb.TestSimpleMessage{}
	if err := e.Decode(buf, decoded); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(decoded, msg) {
		t.Logf("decoded=%+v\n", decoded)
		t.Error("Decoded message is not the same as original")
	}
}
