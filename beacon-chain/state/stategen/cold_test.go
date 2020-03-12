package stategen

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestSaveColdState_NonArchivedPoint(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	service := New(db)
	service.slotsPerArchivedPoint = 2
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	beaconState.SetSlot(1)

	if err := service.saveColdState(ctx, [32]byte{}, beaconState); err != errSlotNonArchivedPoint {
		t.Error("Did not get wanted error")
	}
}

func TestSaveColdState_CanSave(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	service := New(db)
	service.slotsPerArchivedPoint = 1
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	beaconState.SetSlot(1)

	r := [32]byte{'a'}
	if err := service.saveColdState(ctx, r, beaconState); err != nil {
		t.Fatal(err)
	}

	if !service.beaconDB.HasArchivedPoint(ctx, 1) {
		t.Error("Did not save cold state")
	}

	if service.beaconDB.ArchivedPointRoot(ctx, 1) != r {
		t.Error("Did not get wanted root")
	}

	receivedState, err := service.beaconDB.ArchivedPointState(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(receivedState.InnerStateUnsafe(), beaconState.InnerStateUnsafe()) {
		t.Error("Did not get wanted state")
	}
}

func TestArchivedPointByIndex_HasPoint(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	service := New(db)
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	index := uint64(999)
	if err := service.beaconDB.SaveArchivedPointState(ctx, beaconState, index); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveArchivedPointRoot(ctx, [32]byte{'A'}, index); err != nil {
		t.Fatal(err)
	}

	savedArchivedState, err := service.archivedPointByIndex(ctx, index)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(beaconState.InnerStateUnsafe(), savedArchivedState.InnerStateUnsafe()) {
		t.Error("Diff saved state")
	}
}

func TestArchivedPointByIndex_DoesntHavePoint(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	service := New(db)

	gBlk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	gRoot, err := ssz.HashTreeRoot(gBlk.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveBlock(ctx, gBlk); err != nil {
		t.Fatal(err)
	}
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	if err := service.beaconDB.SaveState(ctx, beaconState, gRoot); err != nil {
		t.Fatal(err)
	}

	service.slotsPerArchivedPoint = 32
	recoveredState, err := service.archivedPointByIndex(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}

	if recoveredState.Slot() != service.slotsPerArchivedPoint*2 {
		t.Error("Diff state slot")
	}
	savedArchivedState, err := service.beaconDB.ArchivedPointState(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(recoveredState.InnerStateUnsafe(), savedArchivedState.InnerStateUnsafe()) {
		t.Error("Diff saved archived state")
	}
}

func TestRecoverArchivedPointByIndex_CanRecover(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	service := New(db)

	gBlk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	gRoot, err := ssz.HashTreeRoot(gBlk.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveBlock(ctx, gBlk); err != nil {
		t.Fatal(err)
	}
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	if err := service.beaconDB.SaveState(ctx, beaconState, gRoot); err != nil {
		t.Fatal(err)
	}

	service.slotsPerArchivedPoint = 32
	recoveredState, err := service.recoverArchivedPointByIndex(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	if recoveredState.Slot() != service.slotsPerArchivedPoint {
		t.Error("Diff state slot")
	}
	savedArchivedState, err := service.beaconDB.ArchivedPointState(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(recoveredState.InnerStateUnsafe(), savedArchivedState.InnerStateUnsafe()) {
		t.Error("Diff saved state")
	}
}
