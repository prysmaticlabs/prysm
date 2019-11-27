package sync

import (
	"context"
	"strings"
	"testing"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
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
		Slot:       1,
		ParentRoot: testutil.Random32Bytes(t),
		Signature:  []byte("fake"),
	}

	mockBroadcaster := &p2ptest.MockBroadcaster{}

	r := &RegularSync{
		db:          db,
		initialSync: &mockSync.Sync{IsSyncing: false},
		chain: &mock.ChainService{Genesis: time.Now(),
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			}},
	}
	result, err := r.validateBeaconBlockPubSub(
		context.Background(),
		msg,
		mockBroadcaster,
		false, // fromSelf
	)

	if err == nil {
		t.Error("expected an error")
	}

	if result {
		t.Error("Expected false result, got true")
	}
	if mockBroadcaster.BroadcastCalled {
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

	mockBroadcaster := &p2ptest.MockBroadcaster{}
	r := &RegularSync{
		db:          db,
		initialSync: &mockSync.Sync{IsSyncing: false},
		chain:       &mock.ChainService{Genesis: time.Now()},
	}

	result, _ := r.validateBeaconBlockPubSub(
		context.Background(),
		msg,
		mockBroadcaster,
		false, // fromSelf
	)

	if result {
		t.Error("Expected false result, got true")
	}
	if mockBroadcaster.BroadcastCalled {
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

	mockBroadcaster := &p2ptest.MockBroadcaster{}

	r := &RegularSync{
		db:          db,
		initialSync: &mockSync.Sync{IsSyncing: false},
		chain: &mock.ChainService{Genesis: time.Now(),
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			}},
	}
	result, err := r.validateBeaconBlockPubSub(
		context.Background(),
		msg,
		mockBroadcaster,
		false, // fromSelf
	)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !result {
		t.Error("Expected true result, got false")
	}
	if !mockBroadcaster.BroadcastCalled {
		t.Error("Broadcast was not called when it should have been called")
	}

	// The value should be cached now so the second request will fail.
	mockBroadcaster.BroadcastCalled = false
	result, _ = r.validateBeaconBlockPubSub(
		context.Background(),
		msg,
		mockBroadcaster,
		false, // fromSelf
	)
	if result {
		t.Error("Expected false result, got true")
	}
	if mockBroadcaster.BroadcastCalled {
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

	mockBroadcaster := &p2ptest.MockBroadcaster{}

	r := &RegularSync{
		db:          db,
		initialSync: &mockSync.Sync{IsSyncing: false},
		chain: &mock.ChainService{Genesis: time.Now(),
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			}},
	}
	result, _ := r.validateBeaconBlockPubSub(
		context.Background(),
		msg,
		mockBroadcaster,
		false, // fromSelf
	)

	if !result {
		t.Error("Expected true result, got false")
	}
	if !mockBroadcaster.BroadcastCalled {
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
		Slot:       1,
		ParentRoot: testutil.Random32Bytes(t),
		Signature:  sk.Sign([]byte("data"), 0).Marshal(),
	}

	mockBroadcaster := &p2ptest.MockBroadcaster{}

	r := &RegularSync{
		db:          db,
		initialSync: &mockSync.Sync{IsSyncing: false},
		chain:       &mock.ChainService{Genesis: time.Now()},
	}
	result, _ := r.validateBeaconBlockPubSub(
		context.Background(),
		msg,
		mockBroadcaster,
		true, // fromSelf
	)

	if result {
		t.Error("Expected false result, got true")
	}
	if mockBroadcaster.BroadcastCalled {
		t.Error("Broadcast was called when it should not have been called")
	}
}

func TestValidateBeaconBlockPubSub_Syncing(t *testing.T) {
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

	mockBroadcaster := &p2ptest.MockBroadcaster{}

	r := &RegularSync{
		db:          db,
		initialSync: &mockSync.Sync{IsSyncing: true},
		chain: &mock.ChainService{
			Genesis: time.Now(),
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			}},
	}
	result, _ := r.validateBeaconBlockPubSub(
		context.Background(),
		msg,
		mockBroadcaster,
		false, // fromSelf
	)

	if result {
		t.Error("Expected false result, got true")
	}
	if !mockBroadcaster.BroadcastCalled {
		t.Error("Broadcast was not called when it should have been called")
	}
}

func TestValidateBeaconBlockPubSub_RejectBlocksFromFuture(t *testing.T) {
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
		Slot:       1000,
	}

	mockBroadcaster := &p2ptest.MockBroadcaster{}

	r := &RegularSync{
		db:          db,
		initialSync: &mockSync.Sync{IsSyncing: false},
		chain:       &mock.ChainService{Genesis: time.Now()},
	}
	result, err := r.validateBeaconBlockPubSub(
		context.Background(),
		msg,
		mockBroadcaster,
		false, // fromSelf
	)

	if err == nil || !strings.Contains(err.Error(), "could not process slot from the future") {
		t.Errorf("Err = %v, wanted substring %s", err, "could not process slot from the future")
	}

	if result {
		t.Error("Expected false result, got true")
	}
	if mockBroadcaster.BroadcastCalled {
		t.Error("Broadcast was called when it should not have been called")
	}
}

func TestValidateBeaconBlockPubSub_RejectBlocksFromThePast(t *testing.T) {
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
		Slot:       10,
	}

	mockBroadcaster := &p2ptest.MockBroadcaster{}
	genesisTime := time.Now()
	r := &RegularSync{
		db:          db,
		initialSync: &mockSync.Sync{IsSyncing: false},
		chain: &mock.ChainService{
			Genesis: time.Unix(genesisTime.Unix()-1000, 0),
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 1,
			}},
	}
	result, err := r.validateBeaconBlockPubSub(
		context.Background(),
		msg,
		mockBroadcaster,
		false, // fromSelf
	)
	if result {
		t.Error("Expected false result, got true")
	}
	if mockBroadcaster.BroadcastCalled {
		t.Error("Broadcast was called when it should not have been called")
	}
}
