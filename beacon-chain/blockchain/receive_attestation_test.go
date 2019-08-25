package blockchain

import (
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"golang.org/x/net/context"
)

func TestReceiveAttestation_ProcessCorrectly(t *testing.T) {
	hook := logTest.NewGlobal()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	ctx := context.Background()

	chainService := setupBeaconChain(t, db)
	r, _ := ssz.SigningRoot(&ethpb.BeaconBlock{})
	chainService.forkChoiceStore = &store{headRoot: r[:]}

	b := &ethpb.BeaconBlock{}
	if err := chainService.beaconDB.SaveBlock(ctx, b); err != nil {
		t.Fatal(err)
	}
	root, err := ssz.SigningRoot(b)
	if err != nil {
		t.Fatal(err)
	}
	if err := chainService.beaconDB.SaveState(ctx, &pb.BeaconState{}, root); err != nil {
		t.Fatal(err)
	}

	a := &ethpb.Attestation{Data: &ethpb.AttestationData{
		Target:    &ethpb.Checkpoint{Root: root[:]},
		Crosslink: &ethpb.Crosslink{},
	}}
	if err := chainService.ReceiveAttestation(ctx, a); err != nil {
		t.Fatal(err)
	}

	testutil.AssertLogsContain(t, hook, "Finished updating fork choice store for attestation")
	testutil.AssertLogsContain(t, hook, "Finished applying fork choice")
	testutil.AssertLogsContain(t, hook, "Saved head info")
	testutil.AssertLogsContain(t, hook, "Broadcasting attestation")
}

func TestReceiveAttestationNoPubsub_ProcessCorrectly(t *testing.T) {
	hook := logTest.NewGlobal()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	ctx := context.Background()

	chainService := setupBeaconChain(t, db)
	r, _ := ssz.SigningRoot(&ethpb.BeaconBlock{})
	chainService.forkChoiceStore = &store{headRoot: r[:]}

	b := &ethpb.BeaconBlock{}
	if err := chainService.beaconDB.SaveBlock(ctx, b); err != nil {
		t.Fatal(err)
	}
	root, err := ssz.SigningRoot(b)
	if err != nil {
		t.Fatal(err)
	}
	if err := chainService.beaconDB.SaveState(ctx, &pb.BeaconState{}, root); err != nil {
		t.Fatal(err)
	}

	a := &ethpb.Attestation{Data: &ethpb.AttestationData{
		Target:    &ethpb.Checkpoint{Root: root[:]},
		Crosslink: &ethpb.Crosslink{},
	}}
	if err := chainService.ReceiveAttestationNoPubsub(ctx, a); err != nil {
		t.Fatal(err)
	}

	testutil.AssertLogsContain(t, hook, "Finished updating fork choice store for attestation")
	testutil.AssertLogsContain(t, hook, "Finished applying fork choice")
	testutil.AssertLogsContain(t, hook, "Saved head info")
	testutil.AssertLogsDoNotContain(t, hook, "Broadcasting attestation")
}
