package stategen

import (
	"context"
	"testing"

	testDB "github.com/prysmaticlabs/prysm/v3/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestSaveState_HotStateCanBeSaved(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	service := New(beaconDB)
	service.slotsPerArchivedPoint = 1
	beaconState, _ := util.DeterministicGenesisState(t, 32)
	// This goes to hot section, verify it can save on epoch boundary.
	require.NoError(t, beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch))

	r := [32]byte{'a'}
	require.NoError(t, service.SaveState(ctx, r, beaconState))

	// Should save both state and state summary.
	_, ok, err := service.epochBoundaryStateCache.getByBlockRoot(r)
	require.NoError(t, err)
	assert.Equal(t, true, ok, "Should have saved the state")
	assert.Equal(t, true, service.beaconDB.HasStateSummary(ctx, r), "Should have saved the state summary")
}

func TestSaveState_HotStateCached(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	service := New(beaconDB)
	service.slotsPerArchivedPoint = 1
	beaconState, _ := util.DeterministicGenesisState(t, 32)
	require.NoError(t, beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch))

	// Cache the state prior.
	r := [32]byte{'a'}
	service.hotStateCache.put(r, beaconState)
	require.NoError(t, service.SaveState(ctx, r, beaconState))

	// Should not save the state and state summary.
	assert.Equal(t, false, service.beaconDB.HasState(ctx, r), "Should not have saved the state")
	assert.Equal(t, false, service.beaconDB.HasStateSummary(ctx, r), "Should have saved the state summary")
	require.LogsDoNotContain(t, hook, "Saved full state on epoch boundary")
}

func TestState_ForceCheckpoint_SavesStateToDatabase(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	svc := New(beaconDB)
	beaconState, _ := util.DeterministicGenesisState(t, 32)
	require.NoError(t, beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch))

	r := [32]byte{'a'}
	svc.hotStateCache.put(r, beaconState)

	require.Equal(t, false, beaconDB.HasState(ctx, r), "Database has state stored already")
	assert.NoError(t, svc.ForceCheckpoint(ctx, r[:]))
	assert.Equal(t, true, beaconDB.HasState(ctx, r), "Did not save checkpoint to database")

	// Should not panic with genesis finalized root.
	assert.NoError(t, svc.ForceCheckpoint(ctx, params.BeaconConfig().ZeroHash[:]))
}

func TestSaveState_Alreadyhas(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service := New(beaconDB)

	beaconState, _ := util.DeterministicGenesisState(t, 32)
	require.NoError(t, beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch))
	r := [32]byte{'A'}

	// Pre cache the hot state.
	service.hotStateCache.put(r, beaconState)
	require.NoError(t, service.saveStateByRoot(ctx, r, beaconState))

	// Should not save the state and state summary.
	assert.Equal(t, false, service.beaconDB.HasState(ctx, r), "Should not have saved the state")
	assert.Equal(t, false, service.beaconDB.HasStateSummary(ctx, r), "Should have saved the state summary")
	require.LogsDoNotContain(t, hook, "Saved full state on epoch boundary")
}

func TestSaveState_CanSaveOnEpochBoundary(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service := New(beaconDB)

	beaconState, _ := util.DeterministicGenesisState(t, 32)
	require.NoError(t, beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch))
	r := [32]byte{'A'}

	require.NoError(t, service.saveStateByRoot(ctx, r, beaconState))

	// Should save both state and state summary.
	_, ok, err := service.epochBoundaryStateCache.getByBlockRoot(r)
	require.NoError(t, err)
	require.Equal(t, true, ok, "Did not save epoch boundary state")
	assert.Equal(t, true, service.beaconDB.HasStateSummary(ctx, r), "Should have saved the state summary")
	// Should have not been saved in DB.
	require.Equal(t, false, beaconDB.HasState(ctx, r))
}

func TestSaveState_NoSaveNotEpochBoundary(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service := New(beaconDB)

	beaconState, _ := util.DeterministicGenesisState(t, 32)
	require.NoError(t, beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch-1))
	r := [32]byte{'A'}
	b := util.NewBeaconBlock()
	util.SaveBlock(t, ctx, beaconDB, b)
	gRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, gRoot))
	require.NoError(t, service.SaveState(ctx, r, beaconState))

	// Should only save state summary.
	assert.Equal(t, false, service.beaconDB.HasState(ctx, r), "Should not have saved the state")
	assert.Equal(t, true, service.beaconDB.HasStateSummary(ctx, r), "Should have saved the state summary")
	require.LogsDoNotContain(t, hook, "Saved full state on epoch boundary")
	// Should have not been saved in DB.
	require.Equal(t, false, beaconDB.HasState(ctx, r))
}

func TestSaveState_CanSaveHotStateToDB(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service := New(beaconDB)
	service.EnableSaveHotStateToDB(ctx)
	beaconState, _ := util.DeterministicGenesisState(t, 32)
	require.NoError(t, beaconState.SetSlot(defaultHotStateDBInterval))

	r := [32]byte{'A'}
	require.NoError(t, service.saveStateByRoot(ctx, r, beaconState))

	require.LogsContain(t, hook, "Saving hot state to DB")
	// Should have saved in DB.
	require.Equal(t, true, beaconDB.HasState(ctx, r))
}

func TestEnableSaveHotStateToDB_Enabled(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service := New(beaconDB)

	service.EnableSaveHotStateToDB(ctx)
	require.LogsContain(t, hook, "Entering mode to save hot states in DB")
	require.Equal(t, true, service.saveHotStateDB.enabled)
}

func TestEnableSaveHotStateToDB_AlreadyEnabled(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service := New(beaconDB)
	service.saveHotStateDB.enabled = true
	service.EnableSaveHotStateToDB(ctx)
	require.LogsDoNotContain(t, hook, "Entering mode to save hot states in DB")
	require.Equal(t, true, service.saveHotStateDB.enabled)
}

func TestEnableSaveHotStateToDB_Disabled(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service := New(beaconDB)
	service.saveHotStateDB.enabled = true
	b := util.NewBeaconBlock()
	util.SaveBlock(t, ctx, beaconDB, b)
	r, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	service.saveHotStateDB.blockRootsOfSavedStates = [][32]byte{r}
	require.NoError(t, service.DisableSaveHotStateToDB(ctx))
	require.LogsContain(t, hook, "Exiting mode to save hot states in DB")
	require.Equal(t, false, service.saveHotStateDB.enabled)
	require.Equal(t, 0, len(service.saveHotStateDB.blockRootsOfSavedStates))
}

func TestEnableSaveHotStateToDB_AlreadyDisabled(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service := New(beaconDB)
	require.NoError(t, service.DisableSaveHotStateToDB(ctx))
	require.LogsDoNotContain(t, hook, "Exiting mode to save hot states in DB")
	require.Equal(t, false, service.saveHotStateDB.enabled)
}
