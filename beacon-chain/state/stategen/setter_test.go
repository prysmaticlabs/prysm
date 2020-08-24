package stategen

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
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
	require.NoError(t, beaconState.SetSlot(slot))
	service.finalizedInfo.slot = slot + 1

	r := [32]byte{'a'}
	require.NoError(t, service.SaveState(ctx, r, beaconState))

	assert.Equal(t, true, service.beaconDB.HasArchivedPoint(ctx, 1), "Did not save cold state")
	assert.Equal(t, r, service.beaconDB.ArchivedPointRoot(ctx, 1), "Did not get wanted root")

	require.LogsContain(t, hook, "Saved full state on archived point")
}

func TestSaveState_HotStateCanBeSaved(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())
	service.slotsPerArchivedPoint = 1
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	// This goes to hot section, verify it can save on epoch boundary.
	require.NoError(t, beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch))

	r := [32]byte{'a'}
	require.NoError(t, service.SaveState(ctx, r, beaconState))

	// Should save both state and state summary.
	_, ok, err := service.epochBoundaryStateCache.getByRoot(r)
	require.NoError(t, err)
	assert.Equal(t, true, ok, "Should have saved the state")
	assert.Equal(t, true, service.stateSummaryCache.Has(r), "Should have saved the state summary")
}

func TestSaveState_HotStateCached(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())
	service.slotsPerArchivedPoint = 1
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	require.NoError(t, beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch))

	// Cache the state prior.
	r := [32]byte{'a'}
	service.hotStateCache.Put(r, beaconState)
	require.NoError(t, service.SaveState(ctx, r, beaconState))

	// Should not save the state and state summary.
	assert.Equal(t, false, service.beaconDB.HasState(ctx, r), "Should not have saved the state")
	assert.Equal(t, false, service.beaconDB.HasStateSummary(ctx, r), "Should have saved the state summary")
	require.LogsDoNotContain(t, hook, "Saved full state on epoch boundary")
}

func TestState_ForceCheckpoint_SavesStateToDatabase(t *testing.T) {
	ctx := context.Background()
	db, ssc := testDB.SetupDB(t)

	svc := New(db, ssc)
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	if err := beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch); err != nil {
		t.Fatal(err)
	}

	r := [32]byte{'a'}
	svc.hotStateCache.Put(r, beaconState)

	if db.HasState(ctx, r) {
		t.Fatal("Database has state stored already")
	}
	if err := svc.ForceCheckpoint(ctx, r[:]); err != nil {
		t.Error(err)
	}
	if !db.HasState(ctx, r) {
		t.Error("Did not save checkpoint to database")
	}

	// Should not panic with genesis finalized root.
	if err := svc.ForceCheckpoint(ctx, params.BeaconConfig().ZeroHash[:]); err != nil {
		t.Error(err)
	}
}
