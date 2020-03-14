package stategen

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestSaveState_ColdStateCanBeSaved(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	service := New(db)
	service.slotsPerArchivedPoint = 1
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)

	// This goes to cold section.
	slot := uint64(1)
	beaconState.SetSlot(slot)
	service.splitInfo.slot = slot + 1

	r := [32]byte{'a'}
	if err := service.SaveState(ctx, r, beaconState); err != nil {
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

	testutil.AssertLogsContain(t, hook, "Saved full state on archived point")
}

func TestSaveState_HotStateCanBeSaved(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	service := New(db)
	service.slotsPerArchivedPoint = 1
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	// This goes to hot section, verify it can save on epoch boundary.
	beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch)

	r := [32]byte{'a'}
	if err := service.SaveState(ctx, r, beaconState); err != nil {
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

func TestSaveState_HotStateCached(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	service := New(db)
	service.slotsPerArchivedPoint = 1
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch)

	// Cache the state prior.
	r := [32]byte{'a'}
	service.hotStateCache.Put(r, beaconState)
	if err := service.SaveState(ctx, r, beaconState); err != nil {
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
