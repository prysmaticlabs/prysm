package stategen

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestSaveHotState_AlreadyHas(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	service := New(db)

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch)
	r := [32]byte{'A'}

	// Pre cache the hot state.
	service.hotStateCache.Put(r, beaconState)
	if err := service.saveHotState(ctx, r, beaconState); err != nil {
		t.Fatal(err)
	}

	// Should not save the state and state summary.
	if service.beaconDB.HasState(ctx, r) {
		t.Error("Should not have saved the state")
	}
	if service.beaconDB.HasStateSummary(ctx, r) {
		t.Error("Should have saved the state summary")
	}
	testutil.AssertLogsDoNotContain(t, hook, "Saved full state on epoch boundary")
}

func TestSaveHotState_CanSaveOnEpochBoundary(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	service := New(db)

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch)
	r := [32]byte{'A'}

	if err := service.saveHotState(ctx, r, beaconState); err != nil {
		t.Fatal(err)
	}

	// Should save both state and state summary.
	if !service.beaconDB.HasState(ctx, r) {
		t.Error("Should have saved the state")
	}
	if !service.beaconDB.HasStateSummary(ctx, r) {
		t.Error("Should have saved the state summary")
	}
	testutil.AssertLogsContain(t, hook, "Saved full state on epoch boundary")
}

func TestSaveHotState_NoSaveNotEpochBoundary(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	service := New(db)

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch - 1)
	r := [32]byte{'A'}
	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	if err := db.SaveBlock(ctx, b); err != nil {
		t.Fatal(err)
	}
	gRoot, _ := ssz.HashTreeRoot(b.Block)
	if err := db.SaveGenesisBlockRoot(ctx, gRoot); err != nil {
		t.Fatal(err)
	}

	if err := service.saveHotState(ctx, r, beaconState); err != nil {
		t.Fatal(err)
	}

	// Should only save state summary.
	if service.beaconDB.HasState(ctx, r) {
		t.Error("Should not have saved the state")
	}
	if !service.beaconDB.HasStateSummary(ctx, r) {
		t.Error("Should have saved the state summary")
	}
	testutil.AssertLogsDoNotContain(t, hook, "Saved full state on epoch boundary")
}

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

func TestLoadEpochBoundaryRoot_Exists(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	service := New(db)
	r := [32]byte{'a'}
	service.setEpochBoundaryRoot(params.BeaconConfig().SlotsPerEpoch, r)

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	boundaryRoot, err := service.loadEpochBoundaryRoot(ctx, r, beaconState)
	if err != nil {
		t.Fatal(err)
	}

	if r != boundaryRoot {
		t.Error("Did not correctly load boundary root")
	}
}

func TestLoadEpochBoundaryRoot_SameSlot(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	service := New(db)
	r := [32]byte{'a'}

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch)
	boundaryRoot, err := service.loadEpochBoundaryRoot(ctx, r, beaconState)
	if err != nil {
		t.Fatal(err)
	}

	if r != boundaryRoot {
		t.Error("Did not correctly load boundary root")
	}
}

func TestLoadEpochBoundaryRoot_Genesis(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	service := New(db)
	r := [32]byte{'a'}

	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	if err := db.SaveBlock(ctx, b); err != nil {
		t.Fatal(err)
	}
	gRoot, _ := ssz.HashTreeRoot(b.Block)
	if err := db.SaveGenesisBlockRoot(ctx, gRoot); err != nil {
		t.Fatal(err)
	}
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	beaconState.SetSlot(1)
	boundaryRoot, err := service.loadEpochBoundaryRoot(ctx, r, beaconState)
	if err != nil {
		t.Fatal(err)
	}

	if boundaryRoot != gRoot {
		t.Error("Did not correctly load boundary root")
	}
}

func TestLoadEpochBoundaryRoot_LastSavedBlock(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	service := New(db)

	b1 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: service.splitInfo.slot + 5}}
	if err := service.beaconDB.SaveBlock(ctx, b1); err != nil {
		t.Fatal(err)
	}
	b1Root, _ := ssz.HashTreeRoot(b1.Block)

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch + 10)
	boundaryRoot, err := service.loadEpochBoundaryRoot(ctx, [32]byte{}, beaconState)
	if err != nil {
		t.Fatal(err)
	}

	if boundaryRoot != b1Root {
		t.Error("Did not correctly load boundary root")
	}
}
