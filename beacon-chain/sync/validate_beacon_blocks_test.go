package sync

import (
	"context"
	"testing"

	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

// General note for writing validation tests: Use a random value for any field
// on the beacon block to avoid hitting shared global cache conditions across
// tests in this package.

func TestValidateBeaconBlockPubSub_InvalidSignature(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
	msg := &ethpb.BeaconBlock{
		ParentRoot: testutil.Random32Bytes(t),
		Signature:  []byte("fake"),
	}

	mock := &p2ptest.MockBroadcaster{}

	r := &RegularSync{db: db}
	result := r.validateBeaconBlockPubSub(
		context.Background(),
		msg,
		mock,
		false, // fromSelf
	)

	if result {
		t.Error("Expected false result, got true")
	}
	if mock.BroadcastCalled {
		t.Error("Broadcast was called when it should not have been called")
	}
}

func TestValidateBeaconBlockPubSub_BlockAlreadyPresentInDB(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
	msg := &ethpb.BeaconBlock{
		Slot:       100,
		ParentRoot: testutil.Random32Bytes(t),
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
		false, // fromSelf
	)

	if result {
		t.Error("Expected false result, got true")
	}
	if mock.BroadcastCalled {
		t.Error("Broadcast was called when it should not have been called")
	}
}

func TestValidateBeaconBlockPubSub_BlockAlreadyPresentInCache(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
	b := []byte("sk")
	b32 := bytesutil.ToBytes32(b)
	sk, err := bls.SecretKeyFromBytes(b32[:])
	if err != nil {
		t.Fatal(err)
	}
	msg := &ethpb.BeaconBlock{
		ParentRoot: testutil.Random32Bytes(t),
		Signature:  sk.Sign([]byte("data"), 0).Marshal(),
	}

	mock := &p2ptest.MockBroadcaster{}

	r := &RegularSync{db: db}
	result := r.validateBeaconBlockPubSub(
		context.Background(),
		msg,
		mock,
		false, // fromSelf
	)

	if !result {
		t.Error("Expected true result, got false")
	}
	if !mock.BroadcastCalled {
		t.Error("Broadcast was not called when it should have been called")
	}

	// The value should be cached now so the second request will fail.
	mock.BroadcastCalled = false
	result = r.validateBeaconBlockPubSub(
		context.Background(),
		msg,
		mock,
		false, // fromSelf
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
		ParentRoot: testutil.Random32Bytes(t),
		Signature:  sk.Sign([]byte("data"), 0).Marshal(),
	}

	mock := &p2ptest.MockBroadcaster{}

	r := &RegularSync{db: db}
	result := r.validateBeaconBlockPubSub(
		context.Background(),
		msg,
		mock,
		false, // fromSelf
	)

	if !result {
		t.Error("Expected true result, got false")
	}
	if !mock.BroadcastCalled {
		t.Error("Broadcast was not called when it should have been called")
	}
}

func TestValidateBeaconBlockPubSub_ValidSignature_FromSelf(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
	b := []byte("sk")
	b32 := bytesutil.ToBytes32(b)
	sk, err := bls.SecretKeyFromBytes(b32[:])
	if err != nil {
		t.Fatal(err)
	}
	msg := &ethpb.BeaconBlock{
		ParentRoot: testutil.Random32Bytes(t),
		Signature:  sk.Sign([]byte("data"), 0).Marshal(),
	}

	mock := &p2ptest.MockBroadcaster{}

	r := &RegularSync{db: db}
	result := r.validateBeaconBlockPubSub(
		context.Background(),
		msg,
		mock,
		true, // fromSelf
	)

	if result {
		t.Error("Expected false result, got true")
	}
	if mock.BroadcastCalled {
		t.Error("Broadcast was called when it should not have been called")
	}
}
