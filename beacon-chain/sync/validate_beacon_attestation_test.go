package sync

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsubpb "github.com/libp2p/go-libp2p-pubsub/pb"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestValidateBeaconAttestation_ValidBlock(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()

	rs := &Service{
		p2p: p,
		db:  db,
		chain: &mockChain.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
		initialSync: &mockSync.Sync{IsSyncing: false},
	}

	blk := &ethpb.BeaconBlock{
		Slot: 55,
	}
	if err := db.SaveBlock(ctx, blk); err != nil {
		t.Fatal(err)
	}

	blockRoot, err := ssz.SigningRoot(blk)
	if err != nil {
		t.Fatal(err)
	}

	msg := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			BeaconBlockRoot: blockRoot[:],
			Source: &ethpb.Checkpoint{
				Epoch: 1,
			},
			Target: &ethpb.Checkpoint{
				Epoch: 2,
			},
		},
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
	valid := rs.validateBeaconAttestation(ctx, "", m)

	if !valid {
		t.Error("Beacon attestation failed validation")
	}
}

func TestValidateBeaconAttestation_InvalidBlock(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()

	rs := &Service{
		p2p:         p,
		db:          db,
		initialSync: &mockSync.Sync{IsSyncing: false},
	}

	msg := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			BeaconBlockRoot: testutil.Random32Bytes(t),
		},
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
	valid := rs.validateBeaconAttestation(ctx, "", m)
	if valid {
		t.Error("Invalid beacon attestation passed validation when it should not have")
	}
}

func TestValidateBeaconAttestation_ValidBlock_FromSelf(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()

	rs := &Service{
		p2p:         p,
		db:          db,
		initialSync: &mockSync.Sync{IsSyncing: false},
	}

	blk := &ethpb.BeaconBlock{
		Slot: 55,
	}
	if err := db.SaveBlock(ctx, blk); err != nil {
		t.Fatal(err)
	}

	blockRoot, err := ssz.SigningRoot(blk)
	if err != nil {
		t.Fatal(err)
	}

	msg := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			BeaconBlockRoot: blockRoot[:],
		},
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
	valid := rs.validateBeaconAttestation(ctx, p.PeerID(), m)
	if valid {
		t.Error("Beacon attestation passed validation")
	}
}

func TestValidateBeaconAttestation_Syncing(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()

	rs := &Service{
		p2p:         p,
		db:          db,
		initialSync: &mockSync.Sync{IsSyncing: true},
	}

	blk := &ethpb.BeaconBlock{
		Slot: 55,
	}
	if err := db.SaveBlock(ctx, blk); err != nil {
		t.Fatal(err)
	}

	blockRoot, err := ssz.SigningRoot(blk)
	if err != nil {
		t.Fatal(err)
	}

	msg := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			BeaconBlockRoot: blockRoot[:],
		},
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
	valid := rs.validateBeaconAttestation(ctx, "", m)
	if valid {
		t.Error("Beacon attestation passed validation")
	}
}

func TestValidateBeaconAttestation_OldAttestation(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()

	rs := &Service{
		p2p: p,
		db:  db,
		chain: &mockChain.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 10,
			},
		},
		initialSync: &mockSync.Sync{IsSyncing: false},
	}

	blk := &ethpb.BeaconBlock{
		Slot: 55,
	}
	if err := db.SaveBlock(ctx, blk); err != nil {
		t.Fatal(err)
	}

	blockRoot, err := ssz.SigningRoot(blk)
	if err != nil {
		t.Fatal(err)
	}

	msg := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			BeaconBlockRoot: blockRoot[:],
			Source: &ethpb.Checkpoint{
				Epoch: 10,
			},
			Target: &ethpb.Checkpoint{
				Epoch: 2,
			},
		},
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
	valid := rs.validateBeaconAttestation(ctx, "", m)
	if valid {
		t.Error("Beacon attestation passed validation when it should have failed")
	}

	// source and target epoch same as finalized checkpoint
	msg = &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			BeaconBlockRoot: blockRoot[:],
			Source: &ethpb.Checkpoint{
				Epoch: 10,
			},
			Target: &ethpb.Checkpoint{
				Epoch: 10,
			},
		},
	}

	buf = new(bytes.Buffer)
	if _, err := p.Encoding().Encode(buf, msg); err != nil {
		t.Fatal(err)
	}
	m = &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(msg)],
			},
		},
	}
	valid = rs.validateBeaconAttestation(ctx, "", m)
	if valid {
		t.Error("Beacon attestation passed validation when it should have failed")
	}
}

func TestValidateBeaconAttestation_FirstEpoch(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()

	rs := &Service{
		db:  db,
		p2p: p,
		chain: &mockChain.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
		initialSync: &mockSync.Sync{IsSyncing: false},
	}

	blk := &ethpb.BeaconBlock{
		Slot: 1,
	}
	if err := db.SaveBlock(ctx, blk); err != nil {
		t.Fatal(err)
	}

	blockRoot, err := ssz.SigningRoot(blk)
	if err != nil {
		t.Fatal(err)
	}

	// Attestation at genesis epoch should not be rejected
	msg := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			BeaconBlockRoot: blockRoot[:],
			Source: &ethpb.Checkpoint{
				Epoch: 0,
			},
			Target: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
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
	valid := rs.validateBeaconAttestation(ctx, "", m)
	if !valid {
		t.Error("Beacon attestation did not pass validation")
	}
}
