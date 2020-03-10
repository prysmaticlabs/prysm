package stategen

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestLoadHoteStateByRoot_Cached(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	service := New(db)

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	r := [32]byte{'A'}
	service.hotStateCache.Put(r, beaconState)

	// This tests where hot state was already cached.
	loadedState, err := service.loadHotStateByRoot(ctx, r)
	if err != nil {
		t.Fatal(err)
	}

	if !proto.Equal(loadedState.InnerStateUnsafe(), beaconState.InnerStateUnsafe()) {
		t.Error("Did not correctly cache state")
	}
}

func TestLoadHoteStateByRoot_FromDBCanProcess(t *testing.T) {
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

	// This tests where hot state was not cached and needs processing.
	loadedState, err := service.loadHotStateByRoot(ctx, blkRoot)
	if err != nil {
		t.Fatal(err)
	}

	if loadedState.Slot() != targetSlot {
		t.Error("Did not correctly load state")
	}
}

func TestLoadHoteStateByRoot_FromDBBoundaryCase(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	service := New(db)

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	boundaryRoot := [32]byte{'A'}
	if err := service.beaconDB.SaveState(ctx, beaconState, boundaryRoot); err != nil {
		t.Fatal(err)
	}
	targetSlot := uint64(0)
	if err := service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{
		Slot:         targetSlot,
		Root:         boundaryRoot[:],
		BoundaryRoot: boundaryRoot[:],
	}); err != nil {
		t.Fatal(err)
	}

	// This tests where hot state was not cached but doesn't need processing
	// because it on the epoch boundary slot.
	loadedState, err := service.loadHotStateByRoot(ctx, boundaryRoot)
	if err != nil {
		t.Fatal(err)
	}

	if loadedState.Slot() != targetSlot {
		t.Error("Did not correctly load state")
	}
}

func TestLoadHoteStateBySlot_CanAdvanceSlotUsingCache(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	service := New(db)
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	r := [32]byte{'A'}
	service.hotStateCache.Put(r, beaconState)
	service.setEpochBoundaryRoot(0, r)

	slot := uint64(10)
	loadedState, err := service.loadHotStateBySlot(ctx, slot)
	if err != nil {
		t.Fatal(err)
	}
	if loadedState.Slot() != slot {
		t.Error("Did not correctly load state")
	}
}

func TestLoadHoteStateBySlot_CanAdvanceSlotUsingDB(t *testing.T) {
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
	loadedState, err := service.loadHotStateBySlot(ctx, slot)
	if err != nil {
		t.Fatal(err)
	}
	if loadedState.Slot() != slot {
		t.Error("Did not correctly load state")
	}
}
