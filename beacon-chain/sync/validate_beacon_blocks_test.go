package sync

import (
	"bytes"
	"context"
	"reflect"
	"testing"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsubpb "github.com/libp2p/go-libp2p-pubsub/pb"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
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
	ctx := context.Background()
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
	msg := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot:       1,
			ParentRoot: testutil.Random32Bytes(t),
		},
		Signature: []byte("fake"),
	}

	p := p2ptest.NewTestP2P(t)

	r := &Service{
		db:          db,
		p2p:         p,
		initialSync: &mockSync.Sync{IsSyncing: false},
		chain: &mock.ChainService{Genesis: time.Now(),
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			}},
	}

	buf := new(bytes.Buffer)
	if _, err := p.Encoding().Encode(buf, msg); err != nil {
		t.Fatal(err)
	}
	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(msg)],
			},
		},
	}
	result := r.validateBeaconBlockPubSub(ctx, "", m)

	if result {
		t.Error("Expected false result, got true")
	}
}

func TestValidateBeaconBlockPubSub_BlockAlreadyPresentInDB(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
	ctx := context.Background()

	p := p2ptest.NewTestP2P(t)
	msg := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot:       100,
			ParentRoot: testutil.Random32Bytes(t),
		},
	}
	if err := db.SaveBlock(context.Background(), msg); err != nil {
		t.Fatal(err)
	}

	r := &Service{
		db:          db,
		p2p:         p,
		initialSync: &mockSync.Sync{IsSyncing: false},
		chain:       &mock.ChainService{Genesis: time.Now()},
	}

	buf := new(bytes.Buffer)
	if _, err := p.Encoding().Encode(buf, msg); err != nil {
		t.Fatal(err)
	}

	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(msg)],
			},
		},
	}
	result := r.validateBeaconBlockPubSub(ctx, "", m)

	if result {
		t.Error("Expected false result, got true")
	}
}

func TestValidateBeaconBlockPubSub_ValidSignature(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()
	b := []byte("sk")
	b32 := bytesutil.ToBytes32(b)
	sk, err := bls.SecretKeyFromBytes(b32[:])
	if err != nil {
		t.Fatal(err)
	}
	msg := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ParentRoot: testutil.Random32Bytes(t),
		},
		Signature: sk.Sign([]byte("data"), 0).Marshal(),
	}

	r := &Service{
		db:          db,
		p2p:         p,
		initialSync: &mockSync.Sync{IsSyncing: false},
		chain: &mock.ChainService{Genesis: time.Now(),
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			}},
	}

	buf := new(bytes.Buffer)
	if _, err := p.Encoding().Encode(buf, msg); err != nil {
		t.Fatal(err)
	}
	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(msg)],
			},
		},
	}
	result := r.validateBeaconBlockPubSub(ctx, "", m)
	if !result {
		t.Error("Expected true result, got false")
	}

	if m.ValidatorData == nil {
		t.Error("Decoded message was not set on the message validator data")
	}
}

func TestValidateBeaconBlockPubSub_Syncing(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()
	b := []byte("sk")
	b32 := bytesutil.ToBytes32(b)
	sk, err := bls.SecretKeyFromBytes(b32[:])
	if err != nil {
		t.Fatal(err)
	}
	msg := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ParentRoot: testutil.Random32Bytes(t),
		},
		Signature: sk.Sign([]byte("data"), 0).Marshal(),
	}

	r := &Service{
		db:          db,
		p2p:         p,
		initialSync: &mockSync.Sync{IsSyncing: true},
		chain: &mock.ChainService{
			Genesis: time.Now(),
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			}},
	}

	buf := new(bytes.Buffer)
	if _, err := p.Encoding().Encode(buf, msg); err != nil {
		t.Fatal(err)
	}
	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(msg)],
			},
		},
	}
	result := r.validateBeaconBlockPubSub(ctx, "", m)
	if result {
		t.Error("Expected false result, got true")
	}
}

func TestValidateBeaconBlockPubSub_RejectBlocksFromFuture(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()
	b := []byte("sk")
	b32 := bytesutil.ToBytes32(b)
	sk, err := bls.SecretKeyFromBytes(b32[:])
	if err != nil {
		t.Fatal(err)
	}
	msg := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ParentRoot: testutil.Random32Bytes(t),
			Slot:       1000,
		},
		Signature: sk.Sign([]byte("data"), 0).Marshal(),
	}

	r := &Service{
		p2p:         p,
		db:          db,
		initialSync: &mockSync.Sync{IsSyncing: false},
		chain:       &mock.ChainService{Genesis: time.Now()},
	}

	buf := new(bytes.Buffer)
	if _, err := p.Encoding().Encode(buf, msg); err != nil {
		t.Fatal(err)
	}
	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(msg)],
			},
		},
	}
	result := r.validateBeaconBlockPubSub(ctx, "", m)
	if result {
		t.Error("Expected false result, got true")
	}
}

func TestValidateBeaconBlockPubSub_RejectBlocksFromThePast(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
	b := []byte("sk")
	b32 := bytesutil.ToBytes32(b)
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()
	sk, err := bls.SecretKeyFromBytes(b32[:])
	if err != nil {
		t.Fatal(err)
	}
	msg := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ParentRoot: testutil.Random32Bytes(t),
			Slot:       10,
		},
		Signature: sk.Sign([]byte("data"), 0).Marshal(),
	}

	genesisTime := time.Now()
	r := &Service{
		db:          db,
		p2p:         p,
		initialSync: &mockSync.Sync{IsSyncing: false},
		chain: &mock.ChainService{
			Genesis: time.Unix(genesisTime.Unix()-1000, 0),
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 1,
			}},
	}

	buf := new(bytes.Buffer)
	if _, err := p.Encoding().Encode(buf, msg); err != nil {
		t.Fatal(err)
	}
	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(msg)],
			},
		},
	}
	result := r.validateBeaconBlockPubSub(ctx, "", m)

	if result {
		t.Error("Expected false result, got true")
	}
}
