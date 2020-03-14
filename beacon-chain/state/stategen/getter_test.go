package stategen

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestStateByRoot_ColdState(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	service := New(db)
	service.splitInfo.slot = 2
	service.slotsPerArchivedPoint = 1

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	if err := service.beaconDB.SaveArchivedPointState(ctx, beaconState, 1); err != nil {
		t.Fatal(err)
	}
	r := [32]byte{'a'}
	if err := service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{
		Root: r[:],
		Slot: 1,
	}); err != nil {
		t.Fatal(err)
	}

	loadedState, err := service.StateByRoot(ctx, r)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(loadedState.InnerStateUnsafe(), beaconState.InnerStateUnsafe()) {
		t.Error("Did not correctly save state")
	}
}

func TestStateByRoot_HotStateDB(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	service := New(db)

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	boundaryRoot := [32]byte{'A'}
	blkRoot := [32]byte{'B'}
	if err := service.beaconDB.SaveState(ctx, beaconState, boundaryRoot); err != nil {
		t.Fatal(err)
	}
	targetSlot := uint64(10)
	if err := service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{
		Slot:         targetSlot,
		Root:         blkRoot[:],
		BoundaryRoot: boundaryRoot[:],
	}); err != nil {
		t.Fatal(err)
	}

	loadedState, err := service.StateByRoot(ctx, blkRoot)
	if err != nil {
		t.Fatal(err)
	}
	if loadedState.Slot() != targetSlot {
		t.Error("Did not correctly load state")
	}
}

func TestStateByRoot_HotStateCached(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	service := New(db)

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	r := [32]byte{'A'}
	service.hotStateCache.Put(r, beaconState)

	loadedState, err := service.StateByRoot(ctx, r)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(loadedState.InnerStateUnsafe(), beaconState.InnerStateUnsafe()) {
		t.Error("Did not correctly cache state")
	}
}

func TestStateBySlot_ColdState(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	service := New(db)
	service.slotsPerArchivedPoint = params.BeaconConfig().SlotsPerEpoch * 2
	service.splitInfo.slot = service.slotsPerArchivedPoint + 1

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	r := [32]byte{}
	if err := service.beaconDB.SaveArchivedPointState(ctx, beaconState, 0); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveArchivedPointRoot(ctx, r, 0); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveArchivedPointState(ctx, beaconState, 1); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveArchivedPointRoot(ctx, r, 1); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{
		Slot: service.slotsPerArchivedPoint,
		Root: r[:],
	}); err != nil {
		t.Fatal(err)
	}

	slot := uint64(20)
	loadedState, err := service.StateBySlot(ctx, slot)
	if err != nil {
		t.Fatal(err)
	}
	if loadedState.Slot() != slot {
		t.Error("Did not correctly save state")
	}
}

func TestStateBySlot_HotStateCached(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	service := New(db)

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	r := [32]byte{'A'}
	service.hotStateCache.Put(r, beaconState)
	service.setEpochBoundaryRoot(0, r)

	slot := uint64(10)
	loadedState, err := service.StateBySlot(ctx, slot)
	if err != nil {
		t.Fatal(err)
	}
	if loadedState.Slot() != slot {
		t.Error("Did not correctly load state")
	}
}

func TestStateBySlot_HotStateDB(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	service := New(db)

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	r := [32]byte{'A'}
	service.setEpochBoundaryRoot(0, r)
	if err := service.beaconDB.SaveState(ctx, beaconState, r); err != nil {
		t.Fatal(err)
	}

	slot := uint64(10)
	loadedState, err := service.StateBySlot(ctx, slot)
	if err != nil {
		t.Fatal(err)
	}
	if loadedState.Slot() != slot {
		t.Error("Did not correctly load state")
	}
}
