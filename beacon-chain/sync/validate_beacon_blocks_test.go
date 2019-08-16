package sync

import (
	"context"
	"testing"

	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

func TestValidateBeaconBlockPubSub_InvalidSignature(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
	msg := &ethpb.BeaconBlock{
		Signature: []byte("fake"),
	}

	mock := &p2ptest.MockBroadcaster{}

	r := &RegularSync{db: db}
	result := r.validateBeaconBlockPubSub(
		context.Background(),
		msg,
		mock,
	)

	if result {
		t.Error("Expected false result, got true")
	}
	if mock.BroadcastCalled {
		t.Error("Broadcast was called when it should not have been called")
	}
}

func TestValidateBeaconBlockPubSub_BlockAlreadyPresent(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
	msg := &ethpb.BeaconBlock{
		Slot: 100,
	}
	if err := db.SaveBlock(context.Background(), msg); err != nil {
		t.Fatal(err)
	}

	mock := &p2ptest.MockBroadcaster{}
	r := &RegularSync{
		db: db,
	}

	result := r.validateBeaconBlockPubSub(
		context.Background(),
		msg,
		mock,
	)

	if result {
		t.Error("Expected false result, got true")
	}
	if mock.BroadcastCalled {
		t.Error("Broadcast was called when it should not have been called")
	}
}

func TestValidateBeaconBlockPubSub_ValidSignature(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
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

	r := &RegularSync{db: db}
	result := r.validateBeaconBlockPubSub(
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
