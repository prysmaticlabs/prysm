package stategen

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestMigrateToCold_CanSaveFinalizedInfo(t *testing.T) {
	ctx := context.Background()
	db, c := testDB.SetupDB(t)
	service := New(db, c)
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	b := testutil.NewBeaconBlock()
	b.Block.Slot = 1
	br, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.beaconDB.SaveBlock(ctx, b))
	require.NoError(t, service.epochBoundaryStateCache.put(br, beaconState))
	require.NoError(t, service.MigrateToCold(ctx, br))

	wanted := &finalizedInfo{state: beaconState, root: br, slot: 1}
	assert.DeepEqual(t, wanted, service.finalizedInfo, "Incorrect finalized info")
}

func TestMigrateToCold_HappyPath(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())
	service.slotsPerArchivedPoint = 1
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	stateSlot := uint64(1)
	require.NoError(t, beaconState.SetSlot(stateSlot))
	b := testutil.NewBeaconBlock()
	b.Block.Slot = 2
	fRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.beaconDB.SaveBlock(ctx, b))
	require.NoError(t, service.epochBoundaryStateCache.put(fRoot, beaconState))
	require.NoError(t, service.MigrateToCold(ctx, fRoot))

	gotState, err := service.beaconDB.State(ctx, fRoot)
	require.NoError(t, err)
	assert.DeepEqual(t, beaconState.InnerStateUnsafe(), gotState.InnerStateUnsafe(), "Did not save state")
	gotRoot := service.beaconDB.ArchivedPointRoot(ctx, stateSlot/service.slotsPerArchivedPoint)
	assert.Equal(t, fRoot, gotRoot, "Did not save archived root")
	lastIndex, err := service.beaconDB.LastArchivedSlot(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), lastIndex, "Did not save last archived index")

	require.LogsContain(t, hook, "Saved state in DB")
}

func TestMigrateToCold_RegeneratePath(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())
	service.slotsPerArchivedPoint = 1
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	stateSlot := uint64(1)
	require.NoError(t, beaconState.SetSlot(stateSlot))
	blk := testutil.NewBeaconBlock()
	blk.Block.Slot = 2
	fRoot, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.beaconDB.SaveBlock(ctx, blk))
	require.NoError(t, service.beaconDB.SaveGenesisBlockRoot(ctx, fRoot))
	require.NoError(t, service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{Slot: 1, Root: fRoot[:]}))
	service.finalizedInfo = &finalizedInfo{
		slot:  1,
		root:  fRoot,
		state: beaconState,
	}

	require.NoError(t, service.MigrateToCold(ctx, fRoot))

	gotState, err := service.beaconDB.State(ctx, fRoot)
	require.NoError(t, err)
	assert.DeepEqual(t, beaconState.InnerStateUnsafe(), gotState.InnerStateUnsafe(), "Did not save state")
	gotRoot := service.beaconDB.ArchivedPointRoot(ctx, stateSlot/service.slotsPerArchivedPoint)
	assert.Equal(t, fRoot, gotRoot, "Did not save archived root")
	lastIndex, err := service.beaconDB.LastArchivedSlot(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), lastIndex, "Did not save last archived index")

	require.LogsContain(t, hook, "Saved state in DB")
}

func TestMigrateToCold_StateExistsInDB(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())
	service.slotsPerArchivedPoint = 1
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	stateSlot := uint64(1)
	require.NoError(t, beaconState.SetSlot(stateSlot))
	b := testutil.NewBeaconBlock()
	b.Block.Slot = 2
	fRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.beaconDB.SaveBlock(ctx, b))
	require.NoError(t, service.epochBoundaryStateCache.put(fRoot, beaconState))
	require.NoError(t, service.beaconDB.SaveState(ctx, beaconState, fRoot))

	service.saveHotStateDB.savedStateRoots = [][32]byte{{1}, {2}, {3}, {4}, fRoot}
	require.NoError(t, service.MigrateToCold(ctx, fRoot))
	assert.DeepEqual(t, [][32]byte{{1}, {2}, {3}, {4}}, service.saveHotStateDB.savedStateRoots)
	assert.LogsDoNotContain(t, hook, "Saved state in DB")
}
