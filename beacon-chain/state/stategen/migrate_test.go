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
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestMigrateToCold_CanSaveFinalizedInfo(t *testing.T) {
	ctx := context.Background()
	db, c := testDB.SetupDB(t)
	service := New(db, c)
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 1}}
	br, err := stateutil.BlockRoot(b.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveBlock(ctx, b); err != nil {
		t.Fatal(err)
	}
	if err := service.epochBoundaryStateCache.put(br, beaconState); err != nil {
		t.Fatal(err)
	}

	if err := service.MigrateToCold(ctx, br); err != nil {
		t.Fatal(err)
	}

	wanted := &finalizedInfo{state: beaconState, root: br, slot: 1}
	if !reflect.DeepEqual(wanted, service.finalizedInfo) {
		t.Error("Incorrect finalized info")
	}
}

func TestMigrateToCold_HappyPath(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())
	service.slotsPerArchivedPoint = 1
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	stateSlot := uint64(1)
	if err := beaconState.SetSlot(stateSlot); err != nil {
		t.Fatal(err)
	}
	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 2}}
	fRoot, err := stateutil.BlockRoot(b.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveBlock(ctx, b); err != nil {
		t.Fatal(err)
	}
	if err := service.epochBoundaryStateCache.put(fRoot, beaconState); err != nil {
		t.Fatal(err)
	}
	if err := service.MigrateToCold(ctx, fRoot); err != nil {
		t.Fatal(err)
	}

	gotState, err := service.beaconDB.State(ctx, fRoot)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotState.InnerStateUnsafe(), beaconState.InnerStateUnsafe()) {
		t.Error("Did not save state")
	}
	gotRoot := service.beaconDB.ArchivedPointRoot(ctx, stateSlot/service.slotsPerArchivedPoint)
	if gotRoot != fRoot {
		t.Error("Did not save archived root")
	}
	lastIndex, err := service.beaconDB.LastArchivedIndex(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if lastIndex != 1 {
		t.Error("Did not save last archived index")
	}

	testutil.AssertLogsContain(t, hook, "Saved state in DB")
}

func TestMigrateToCold_RegeneratePath(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())
	service.slotsPerArchivedPoint = 1
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	stateSlot := uint64(1)
	if err := beaconState.SetSlot(stateSlot); err != nil {
		t.Fatal(err)
	}
	blk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 2}}
	fRoot, err := stateutil.BlockRoot(blk.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveBlock(ctx, blk); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveGenesisBlockRoot(ctx, fRoot); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{
		Slot: 1,
		Root: fRoot[:],
	}); err != nil {
		t.Fatal(err)
	}
	service.finalizedInfo = &finalizedInfo{
		slot:  1,
		root:  fRoot,
		state: beaconState,
	}

	if err := service.MigrateToCold(ctx, fRoot); err != nil {
		t.Fatal(err)
	}

	gotState, err := service.beaconDB.State(ctx, fRoot)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotState.InnerStateUnsafe(), beaconState.InnerStateUnsafe()) {
		t.Error("Did not save state")
	}
	gotRoot := service.beaconDB.ArchivedPointRoot(ctx, stateSlot/service.slotsPerArchivedPoint)
	if gotRoot != fRoot {
		t.Error("Did not save archived root")
	}
	lastIndex, err := service.beaconDB.LastArchivedIndex(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if lastIndex != 1 {
		t.Error("Did not save last archived index")
	}

	testutil.AssertLogsContain(t, hook, "Saved state in DB")
}
