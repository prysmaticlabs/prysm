package stategen

import (
	"context"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"sync"
	"testing"

	//"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestSaveState_ColdStateCanBeSaved(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())
	service.slotsPerArchivedPoint = 1
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)

	// This goes to cold section.
	slot := uint64(1)
	if err := beaconState.SetSlot(slot); err != nil {
		t.Fatal(err)
	}
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

	testutil.AssertLogsContain(t, hook, "Saved full state on archived point")
}

func TestSaveState_HotStateCanBeSaved(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())
	service.slotsPerArchivedPoint = 1
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	// This goes to hot section, verify it can save on epoch boundary.
	if err := beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch); err != nil {
		t.Fatal(err)
	}

	r := [32]byte{'a'}
	if err := service.SaveState(ctx, r, beaconState); err != nil {
		t.Fatal(err)
	}

	// Should save both state and state summary.
	if !service.beaconDB.HasState(ctx, r) {
		t.Error("Should have saved the state")
	}
	if !service.stateSummaryCache.Has(r) {
		t.Error("Should have saved the state summary")
	}
	testutil.AssertLogsContain(t, hook, "Saved full state on epoch boundary")
}

func TestSaveState_HotStateCached(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())
	service.slotsPerArchivedPoint = 1
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	if err := beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch); err != nil {
		t.Fatal(err)
	}

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

func TestState_ForceCheckpoint(t *testing.T) {
	t.Fail() // TODO!

	type fields struct {
		beaconDB                db.NoHeadAccessDatabase
		slotsPerArchivedPoint   uint64
		epochBoundarySlotToRoot map[uint64][32]byte
		epochBoundaryLock       sync.RWMutex
		hotStateCache           *cache.HotStateCache
		splitInfo               *splitSlotAndRoot
		stateSummaryCache       *cache.StateSummaryCache
	}
	type args struct {
		ctx   context.Context
		root  [32]byte
		state *state.BeaconState
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &State{
				beaconDB:                tt.fields.beaconDB,
				slotsPerArchivedPoint:   tt.fields.slotsPerArchivedPoint,
				epochBoundarySlotToRoot: tt.fields.epochBoundarySlotToRoot,
				epochBoundaryLock:       tt.fields.epochBoundaryLock,
				hotStateCache:           tt.fields.hotStateCache,
				splitInfo:               tt.fields.splitInfo,
				stateSummaryCache:       tt.fields.stateSummaryCache,
			}
			if err := s.ForceCheckpoint(tt.args.ctx, tt.args.root, tt.args.state); (err != nil) != tt.wantErr {
				t.Errorf("ForceCheckpoint() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}