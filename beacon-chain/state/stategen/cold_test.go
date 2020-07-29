package stategen

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestSaveColdState_NonArchivedPoint(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())
	service.slotsPerArchivedPoint = 2
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	require.NoError(t, beaconState.SetSlot(1))
	assert.NoError(t, service.saveColdState(ctx, [32]byte{}, beaconState))
}

func TestSaveColdState_CanSave(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())
	service.slotsPerArchivedPoint = 1
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	require.NoError(t, beaconState.SetSlot(1))

	r := [32]byte{'a'}
	require.NoError(t, service.saveColdState(ctx, r, beaconState))

	assert.Equal(t, true, service.beaconDB.HasArchivedPoint(ctx, 1), "Did not save cold state")
	assert.Equal(t, r, service.beaconDB.ArchivedPointRoot(ctx, 1), "Did not get wanted root")
}

func TestLoadColdStateByRoot_NoStateSummary(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())
	_, err := service.loadColdStateByRoot(ctx, [32]byte{'a'})
	require.ErrorContains(t, errUnknownStateSummary.Error(), err, "Did not get correct error")
}

func TestLoadStateByRoot_CanGet(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())
	service.slotsPerArchivedPoint = 1

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	blk := testutil.NewBeaconBlock()
	blkRoot, err := stateutil.BlockRoot(blk.Block)
	require.NoError(t, err)
	require.NoError(t, service.beaconDB.SaveGenesisBlockRoot(ctx, blkRoot))
	require.NoError(t, service.beaconDB.SaveBlock(ctx, blk))
	require.NoError(t, service.beaconDB.SaveState(ctx, beaconState, blkRoot))

	if err := service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{
		Root: blkRoot[:],
		Slot: 100,
	}); err != nil {
		t.Fatal(err)
	}

	loadedState, err := service.StateByRoot(ctx, blkRoot)
	require.NoError(t, err)
	assert.DeepEqual(t, beaconState.InnerStateUnsafe(), loadedState.InnerStateUnsafe(), "Did not correctly save state")
}

func TestLoadColdStateBySlot_CanGet(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	blk := testutil.NewBeaconBlock()
	blkRoot, err := stateutil.BlockRoot(blk.Block)
	require.NoError(t, err)
	require.NoError(t, service.beaconDB.SaveGenesisBlockRoot(ctx, blkRoot))
	require.NoError(t, service.beaconDB.SaveBlock(ctx, blk))
	require.NoError(t, service.beaconDB.SaveState(ctx, beaconState, blkRoot))

	loadedState, err := service.loadColdStateBySlot(ctx, 200)
	require.NoError(t, err)
	assert.Equal(t, uint64(200), loadedState.Slot(), "Did not correctly save state")
}
