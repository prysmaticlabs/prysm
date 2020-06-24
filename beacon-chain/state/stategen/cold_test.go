package stategen

import (
	"context"
	"strings"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestSaveColdState_NonArchivedPoint(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())
	service.slotsPerArchivedPoint = 2
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	if err := beaconState.SetSlot(1); err != nil {
		t.Fatal(err)
	}

	if err := service.saveColdState(ctx, [32]byte{}, beaconState); err != nil {
		t.Error(err)
	}
}

func TestSaveColdState_CanSave(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())
	service.slotsPerArchivedPoint = 1
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	if err := beaconState.SetSlot(1); err != nil {
		t.Fatal(err)
	}

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
}

func TestLoadColdStateByRoot_NoStateSummary(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())
	if _, err := service.loadColdStateByRoot(ctx, [32]byte{'a'}); !strings.Contains(err.Error(), errUnknownStateSummary.Error()) {
		t.Fatal("Did not get correct error")
	}
}

func TestLoadColdStateByRoot_CanGet(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())
	service.slotsPerArchivedPoint = 1

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	blk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	blkRoot, err := stateutil.BlockRoot(blk.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveArchivedPointRoot(ctx, blkRoot, 0); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveGenesisBlockRoot(ctx, blkRoot); err != nil {
		t.Fatal(err)
	}
	if service.beaconDB.SaveBlock(ctx, blk) != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveState(ctx, beaconState, blkRoot); err != nil {
		t.Fatal(err)
	}

	if err := service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{
		Root: blkRoot[:],
		Slot: 100,
	}); err != nil {
		t.Fatal(err)
	}

	loadedState, err := service.loadColdStateByRoot(ctx, blkRoot)
	if err != nil {
		t.Fatal(err)
	}
	if loadedState.Slot() != 100 {
		t.Error("Did not correctly save state")
	}
}

func TestLoadColdStateBySlot_CanGet(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	blk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	blkRoot, err := stateutil.BlockRoot(blk.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveArchivedPointRoot(ctx, blkRoot, 0); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveGenesisBlockRoot(ctx, blkRoot); err != nil {
		t.Fatal(err)
	}
	if service.beaconDB.SaveBlock(ctx, blk) != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveState(ctx, beaconState, blkRoot); err != nil {
		t.Fatal(err)
	}

	loadedState, err := service.loadColdStateBySlot(ctx, 200)
	if err != nil {
		t.Fatal(err)
	}
	if loadedState.Slot() != 200 {
		t.Error("Did not correctly save state")
	}
}
