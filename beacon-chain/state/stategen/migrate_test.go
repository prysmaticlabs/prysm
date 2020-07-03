package stategen

import (
	"context"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestMigrateToCold_NoBlock(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())
	service.finalizedInfo.slot = 1
	if err := service.MigrateToCold(ctx, params.BeaconConfig().SlotsPerEpoch, [32]byte{}); err != nil {
		t.Fatal(err)
	}

	testutil.AssertLogsContain(t, hook, "Set hot and cold state split point")
}

func TestMigrateToCold_HigherSplitSlot(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())
	service.finalizedInfo.slot = 2
	if err := service.MigrateToCold(ctx, 1, [32]byte{}); err != nil {
		t.Fatal(err)
	}

	testutil.AssertLogsDoNotContain(t, hook, "Set hot and cold state split point")
}

func TestMigrateToCold_MigrationCompletes(t *testing.T) {
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

	testutil.AssertLogsContain(t, hook, "Saved archived point during state migration")
	testutil.AssertLogsContain(t, hook, "Deleted state during migration")
	testutil.AssertLogsContain(t, hook, "Set hot and cold state split point")
}

func TestMigrateToCold_CantDeleteCurrentArchivedIndex(t *testing.T) {
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
	if err := service.beaconDB.SaveArchivedPointRoot(ctx, bRoot, 1); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveLastArchivedIndex(ctx, 1); err != nil {
		t.Fatal(err)
	}

	if err := service.MigrateToCold(ctx, beaconState.Slot(), [32]byte{}); err != nil {
		t.Fatal(err)
	}

	if !service.beaconDB.HasArchivedPoint(ctx, 1) {
		t.Error("Did not preserve archived point")
	}
	if !service.beaconDB.HasState(ctx, bRoot) {
		t.Error("State should not be deleted")
	}
}

func TestSkippedArchivedPoint_CanRecover(t *testing.T) {
	db, _ := testDB.SetupDB(t)
	ctx := context.Background()
	service := New(db, cache.NewStateSummaryCache())
	service.slotsPerArchivedPoint = 32

	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 31}}
	if err := service.beaconDB.SaveBlock(ctx, b); err != nil {
		t.Fatal(err)
	}
	r, err := ssz.HashTreeRoot(b.Block)
	if err != nil {
		t.Fatal(err)
	}

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	if err := beaconState.SetSlot(31); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveState(ctx, beaconState, r); err != nil {
		t.Fatal(err)
	}

	currentArchivedPoint := uint64(2)
	lastPoint, err := service.recoverArchivedPoint(ctx, currentArchivedPoint)
	if err != nil {
		t.Fatal(err)
	}

	if lastPoint != currentArchivedPoint-1 {
		t.Error("Did not get wanted point")
	}
	if !service.beaconDB.HasArchivedPoint(ctx, lastPoint) {
		t.Error("Did not save archived point index")
	}
	if service.beaconDB.ArchivedPointRoot(ctx, lastPoint) != r {
		t.Error("Did not get wanted archived index root")
	}
}

func TestSkippedArchivedPoint(t *testing.T) {
	tests := []struct {
		a uint64
		b uint64
		c bool
	}{
		{
			a: 0,
			b: 1,
			c: false,
		},
		{
			a: 1,
			b: 1,
			c: false,
		},
		{
			a: 1,
			b: 2,
			c: false,
		},
		{
			a: 1,
			b: 3,
			c: true,
		},
	}
	for _, tt := range tests {
		if tt.c != skippedArchivedPoint(tt.b, tt.a) {
			t.Fatalf("skippedArchivedPoint(%d, %d) = %v, wanted: %v", tt.b, tt.a, skippedArchivedPoint(tt.b, tt.a), tt.c)
		}
	}
}
