package sync

import (
	"context"
	"testing"

	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"

	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

func TestValidateBeaconBlockPubSub_InvalidSignature(t *testing.T) {
	msg := &ethpb.BeaconBlock{
		Signature: []byte("fake"),
	}

	mock := &p2ptest.MockBroadcaster{}

	result := validateBeaconBlockPubSub(
		context.Background(),
		msg,
		mock,
	)

	if !result {
		t.Error("Expected false result, got true")
	}
	if mock.BroadcastCalled {
		t.Error("Broadcast was called when it should not have been called")
	}
}

func TestValidateBeaconBlockPubSub_ValidSignature(t *testing.T) {
	b := []byte("sk")
	b32 := bytesutil.ToBytes32(b)
	sk, err := bls.SecretKeyFromBytes(b32[:])
	if err != nil {
		t.Fatal(err)
	}
	msg := &ethpb.BeaconBlock{
		Signature: sk.Sign([]byte("data"), 0).Marshal(),
	}

	mock := &p2ptest.MockBroadcaster{}

	result := validateBeaconBlockPubSub(
		context.Background(),
		msg,
		mock,
	)

	if !result {
		t.Error("Expected true result, got false")
	}
	if !mock.BroadcastCalled {
		t.Error("Broadcast was not called when it should have been called")
	}
}
