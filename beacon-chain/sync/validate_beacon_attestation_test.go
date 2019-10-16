package sync

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestValidateBeaconAttestation_ValidBlock(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()

	rs := &RegularSync{
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

	valid, err := rs.validateBeaconAttestation(ctx, msg, p, false /*fromSelf*/)
	if err != nil {
		t.Errorf("Beacon attestation failed validation: %v", err)
	}
	if !valid {
		t.Error("Beacon attestation failed validation")
	}

	if !p.BroadcastCalled {
		t.Error("No message was broadcast")
	}

	// It should ignore duplicate identical attestations.
	p.BroadcastCalled = false
	valid, _ = rs.validateBeaconAttestation(ctx, msg, p, false /*fromSelf*/)
	if valid {
		t.Error("Second identical beacon attestation passed validation when it should not have")
	}
	if p.BroadcastCalled {
		t.Error("Second identcial beacon attestation was re-broadcast")
	}
}

func TestValidateBeaconAttestation_InvalidBlock(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()

	rs := &RegularSync{
		db:          db,
		initialSync: &mockSync.Sync{IsSyncing: false},
	}

	msg := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			BeaconBlockRoot: testutil.Random32Bytes(t),
		},
	}

	valid, _ := rs.validateBeaconAttestation(ctx, msg, p, false /*fromSelf*/)
	if valid {
		t.Error("Invalid beacon attestation passed validation when it should not have")
	}
	if p.BroadcastCalled {
		t.Error("Invalid beacon attestation was broadcast")
	}
}

func TestValidateBeaconAttestation_ValidBlock_FromSelf(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()

	rs := &RegularSync{
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

	valid, _ := rs.validateBeaconAttestation(ctx, msg, p, true /*fromSelf*/)
	if valid {
		t.Error("Beacon attestation passed validation")
	}

	if p.BroadcastCalled {
		t.Error("Message was broadcast")
	}
}

func TestValidateBeaconAttestation_Syncing(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()

	rs := &RegularSync{
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

	valid, err := rs.validateBeaconAttestation(ctx, msg, p, false /*fromSelf*/)
	if valid {
		t.Error("Beacon attestation passed validation")
	}
}
