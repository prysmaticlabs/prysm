package stategen

import (
	"context"
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestMigrateToCold_CanSaveFinalizedInfo(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	r := [32]byte{'a'}
	if err := service.epochBoundaryStateCache.put(r, beaconState); err != nil {
		t.Fatal(err)
	}

	if err := service.MigrateToCold(ctx, 1, r); err != nil {
		t.Fatal(err)
	}

	wanted := &finalizedInfo{state: beaconState, root: r, slot: 1}
	if !reflect.DeepEqual(wanted, service.finalizedInfo) {
		t.Error("Incorrect finalized info")
	}
}

func TestMigrateToCold_HappyPath(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())
	service.finalizedInfo.slot = 1
	service.slotsPerArchivedPoint = 2

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	if err := beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch); err != nil {
		t.Fatal(err)
	}
	b := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{Slot: 2},
	}
	if err := service.beaconDB.SaveBlock(ctx, b); err != nil {
		t.Fatal(err)
	}
	bRoot, err := stateutil.BlockRoot(b.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{Root: bRoot[:], Slot: 2}); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveState(ctx, beaconState, bRoot); err != nil {
		t.Fatal(err)
	}

	newBeaconState, _ := testutil.DeterministicGenesisState(t, 32)
	if err := newBeaconState.SetSlot(3); err != nil {
		t.Fatal(err)
	}
	b = &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{Slot: 3},
	}
	if err := service.beaconDB.SaveBlock(ctx, b); err != nil {
		t.Fatal(err)
	}
	bRoot, err = stateutil.BlockRoot(b.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{Root: bRoot[:], Slot: 3}); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveState(ctx, newBeaconState, bRoot); err != nil {
		t.Fatal(err)
	}

	if err := service.MigrateToCold(ctx, beaconState.Slot(), [32]byte{}); err != nil {
		t.Fatal(err)
	}

	if !service.beaconDB.HasArchivedPoint(ctx, 1) {
		t.Error("Did not preserve archived point")
	}

	testutil.AssertLogsContain(t, hook, "Saved state in DB")
}
