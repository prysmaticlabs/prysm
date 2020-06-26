package encoder_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
	testpb "github.com/prysmaticlabs/prysm/proto/testing"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestSszNetworkEncoder_RoundTrip(t *testing.T) {
	e := &encoder.SszNetworkEncoder{UseSnappyCompression: false}
	testRoundTripWithLength(t, e)
	testRoundTripWithGossip(t, e)
}

func TestSszNetworkEncoder_RoundTrip_Snappy(t *testing.T) {
	e := &encoder.SszNetworkEncoder{UseSnappyCompression: true}
	testRoundTripWithLength(t, e)
	testRoundTripWithGossip(t, e)
}

func testRoundTripWithLength(t *testing.T, e *encoder.SszNetworkEncoder) {
	buf := new(bytes.Buffer)
	msg := &testpb.TestSimpleMessage{
		Foo: []byte("fooooo"),
		Bar: 9001,
	}
	_, err := e.EncodeWithMaxLength(buf, msg)
	if err != nil {
		t.Fatal(err)
	}
	decoded := &testpb.TestSimpleMessage{}
	if err := e.DecodeWithMaxLength(buf, decoded); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(decoded, msg) {
		t.Logf("decoded=%+v\n", decoded)
		t.Error("Decoded message is not the same as original")
	}
}

func testRoundTripWithGossip(t *testing.T, e *encoder.SszNetworkEncoder) {
	buf := new(bytes.Buffer)
	msg := &testpb.TestSimpleMessage{
		Foo: []byte("fooooo"),
		Bar: 9001,
	}
	_, err := e.EncodeGossip(buf, msg)
	if err != nil {
		t.Fatal(err)
	}
	decoded := &testpb.TestSimpleMessage{}
	if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(decoded, msg) {
		t.Logf("decoded=%+v\n", decoded)
		t.Error("Decoded message is not the same as original")
	}
}

func TestSszNetworkEncoder_EncodeWithMaxLength(t *testing.T) {
	buf := new(bytes.Buffer)
	msg := &testpb.TestSimpleMessage{
		Foo: []byte("fooooo"),
		Bar: 9001,
	}
	e := &encoder.SszNetworkEncoder{UseSnappyCompression: false}
	//maxLength := uint64(5)
	_, err := e.EncodeWithMaxLength(buf, msg)
	wanted := fmt.Sprintf("which is larger than the provided max limit of %d", params.BeaconNetworkConfig().MaxChunkSize)
	if err == nil {
		t.Fatalf("wanted this error %s but got nothing", wanted)
	}
	if !strings.Contains(err.Error(), wanted) {
		t.Errorf("error did not contain wanted message. Wanted: %s but Got: %s", wanted, err.Error())
	}
}

func TestSszNetworkEncoder_DecodeWithMaxLength(t *testing.T) {
	buf := new(bytes.Buffer)
	msg := &testpb.TestSimpleMessage{
		Foo: []byte("fooooo"),
		Bar: 4242,
	}
	e := &encoder.SszNetworkEncoder{UseSnappyCompression: false}
	maxLength := uint64(5)
	_, err := e.EncodeGossip(buf, msg)
	if err != nil {
		t.Fatal(err)
	}
	decoded := &testpb.TestSimpleMessage{}
	err = e.DecodeWithMaxLength(buf, decoded)
	wanted := fmt.Sprintf("goes over the provided max limit of %d", maxLength)
	if err == nil {
		t.Fatalf("wanted this error %s but got nothing", wanted)
	}
	if !strings.Contains(err.Error(), wanted) {
		t.Errorf("error did not contain wanted message. Wanted: %s but Got: %s", wanted, err.Error())
	}
}
