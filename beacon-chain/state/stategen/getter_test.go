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
)

func TestStateByRoot_ColdState(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	service := New(db)
	service.splitInfo.slot = 2
	service.slotsPerArchivedPoint = 1

	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 1}}
	if err := db.SaveBlock(ctx, b); err != nil {
		t.Fatal(err)
	}
	bRoot, _ := ssz.HashTreeRoot(b.Block)

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	beaconState.SetSlot(1)
	service.beaconDB.SaveState(ctx, beaconState, bRoot)
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
	blk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	blkRoot, _ := ssz.HashTreeRoot(blk.Block)
	service.beaconDB.SaveGenesisBlockRoot(ctx, blkRoot)
	if err := service.beaconDB.SaveState(ctx, beaconState, blkRoot); err != nil {
		t.Fatal(err)
	}
	targetSlot := uint64(10)

	if err := service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{
		Slot: targetSlot,
		Root: blkRoot[:],
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
	if err := service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{
		Root: r[:],
	}); err != nil {
		t.Fatal(err)
	}
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
	beaconState.SetSlot(1)
	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 1}}
	if err := db.SaveBlock(ctx, b); err != nil {
		t.Fatal(err)
	}
	bRoot, _ := ssz.HashTreeRoot(b.Block)
	if err := db.SaveState(ctx, beaconState, bRoot); err != nil {
		t.Fatal(err)
	}
	db.SaveGenesisBlockRoot(ctx, bRoot)

	r := [32]byte{}
	if err := service.beaconDB.SaveArchivedPointRoot(ctx, r, 0); err != nil {
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

func TestStateBySlot_HotStateDB(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	service := New(db)

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	if err := db.SaveBlock(ctx, b); err != nil {
		t.Fatal(err)
	}
	bRoot, _ := ssz.HashTreeRoot(b.Block)
	if err := db.SaveState(ctx, beaconState, bRoot); err != nil {
		t.Fatal(err)
	}
	db.SaveGenesisBlockRoot(ctx, bRoot)

	slot := uint64(10)
	loadedState, err := service.StateBySlot(ctx, slot)
	if err != nil {
		t.Fatal(err)
	}
	if loadedState.Slot() != slot {
		t.Error("Did not correctly load state")
	}
}
