package encoder_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	testpb "github.com/prysmaticlabs/prysm/proto/testing"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestSszNetworkEncoder_RoundTrip(t *testing.T) {
	e := &encoder.SszNetworkEncoder{}
	testRoundTripWithLength(t, e)
	testRoundTripWithGossip(t, e)
}

func TestSszNetworkEncoder_FailsSnappyLength(t *testing.T) {
	e := &encoder.SszNetworkEncoder{}
	att := &testpb.TestSimpleMessage{}
	data := make([]byte, 32)
	binary.PutUvarint(data, encoder.MaxGossipSize+32)
	err := e.DecodeGossip(data, att)
	require.ErrorContains(t, "gossip message exceeds max gossip size", err)
}

func testRoundTripWithLength(t *testing.T, e *encoder.SszNetworkEncoder) {
	buf := new(bytes.Buffer)
	msg := &testpb.TestSimpleMessage{
		Foo: []byte("fooooo"),
		Bar: 9001,
	}
	_, err := e.EncodeWithMaxLength(buf, msg)
	require.NoError(t, err)
	decoded := &testpb.TestSimpleMessage{}
	require.NoError(t, e.DecodeWithMaxLength(buf, decoded))
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
	require.NoError(t, err)
	decoded := &testpb.TestSimpleMessage{}
	require.NoError(t, e.DecodeGossip(buf.Bytes(), decoded))
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
	e := &encoder.SszNetworkEncoder{}
	params.SetupTestConfigCleanup(t)
	c := params.BeaconNetworkConfig()
	c.MaxChunkSize = uint64(5)
	params.OverrideBeaconNetworkConfig(c)
	_, err := e.EncodeWithMaxLength(buf, msg)
	wanted := fmt.Sprintf("which is larger than the provided max limit of %d", params.BeaconNetworkConfig().MaxChunkSize)
	assert.ErrorContains(t, wanted, err)
}

func TestSszNetworkEncoder_DecodeWithMaxLength(t *testing.T) {
	buf := new(bytes.Buffer)
	msg := &testpb.TestSimpleMessage{
		Foo: []byte("fooooo"),
		Bar: 4242,
	}
	e := &encoder.SszNetworkEncoder{}
	params.SetupTestConfigCleanup(t)
	c := params.BeaconNetworkConfig()
	maxChunkSize := uint64(5)
	c.MaxChunkSize = maxChunkSize
	params.OverrideBeaconNetworkConfig(c)
	_, err := e.EncodeGossip(buf, msg)
	require.NoError(t, err)
	decoded := &testpb.TestSimpleMessage{}
	err = e.DecodeWithMaxLength(buf, decoded)
	wanted := fmt.Sprintf("goes over the provided max limit of %d", maxChunkSize)
	assert.ErrorContains(t, wanted, err)
}

func TestSszNetworkEncoder_DecodeWithMultipleFrames(t *testing.T) {
	buf := new(bytes.Buffer)
	st, _ := testutil.DeterministicGenesisState(t, 100)
	e := &encoder.SszNetworkEncoder{}
	params.SetupTestConfigCleanup(t)
	c := params.BeaconNetworkConfig()
	// 4 * 1 Mib
	maxChunkSize := uint64(1 << 22)
	c.MaxChunkSize = maxChunkSize
	params.OverrideBeaconNetworkConfig(c)
	_, err := e.EncodeWithMaxLength(buf, st.InnerStateUnsafe())
	require.NoError(t, err)
	// Max snappy block size
	if buf.Len() <= 76490 {
		t.Errorf("buffer smaller than expected, wanted > %d but got %d", 76490, buf.Len())
	}
	decoded := new(pb.BeaconState)
	err = e.DecodeWithMaxLength(buf, decoded)
	assert.NoError(t, err)
}

func TestSszNetworkEncoder_NegativeMaxLength(t *testing.T) {
	e := &encoder.SszNetworkEncoder{}
	length, err := e.MaxLength(0xfffffffffff)

	assert.Equal(t, 0, length, "Received non zero length on bad message length")
	assert.ErrorContains(t, "max encoded length is negative", err)
}
