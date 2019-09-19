package encoder_test

import (
	"bytes"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	testpb "github.com/prysmaticlabs/prysm/proto/testing"
)

func TestSszNetworkEncoder_RoundTrip(t *testing.T) {
	e := &encoder.SszNetworkEncoder{UseSnappyCompression: false}
	testRoundTrip(t, e)
	testRoundTripWithLength(t, e)
}

func TestSszNetworkEncoder_RoundTrip_Snappy(t *testing.T) {
	e := &encoder.SszNetworkEncoder{UseSnappyCompression: true}
	testRoundTrip(t, e)
	testRoundTripWithLength(t, e)
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
	if err := e.Decode(buf.Bytes(), decoded); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(decoded, msg) {
		t.Logf("decoded=%+v\n", decoded)
		t.Error("Decoded message is not the same as original")
	}
}

func testRoundTripWithLength(t *testing.T, e *encoder.SszNetworkEncoder) {
	buf := new(bytes.Buffer)
	msg := &testpb.TestSimpleMessage{
		Foo: []byte("fooooo"),
		Bar: 9001,
	}
	_, err := e.EncodeWithLength(buf, msg)
	if err != nil {
		t.Fatal(err)
	}
	decoded := &testpb.TestSimpleMessage{}
	if err := e.DecodeWithLength(buf, decoded); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(decoded, msg) {
		t.Logf("decoded=%+v\n", decoded)
		t.Error("Decoded message is not the same as original")
	}
}

// Regression test to see that a block array response received from Sigma Prime's lighthouse would decode.
func TestLighthouseBeaconBlockResponse(t *testing.T) {
	b := []byte{4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 28, 71, 156, 79, 199, 27, 222, 126, 43, 250, 217, 225, 182, 66, 10, 239, 42, 82, 185, 124, 196, 228, 234, 124, 248, 85, 153, 182, 92, 139, 53, 220, 172, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 224, 0, 0, 0, 224, 0, 0, 0, 224, 0, 0, 0, 224, 0, 0, 0, 224, 0, 0, 0, 224, 0, 0, 0}
	decoded := make([]ethpb.BeaconBlock, 0)
	e := &encoder.SszNetworkEncoder{UseSnappyCompression: false}
	if err := e.Decode(b, &decoded); err != nil {
		t.Fatal(err)
	}
}
