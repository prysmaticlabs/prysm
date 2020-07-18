package stategen

import (
	"context"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
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
	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 1}}
	br, err := stateutil.BlockRoot(b.Block)
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
	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 2}}
	fRoot, err := stateutil.BlockRoot(b.Block)
	require.NoError(t, err)
	require.NoError(t, service.beaconDB.SaveBlock(ctx, b))
	require.NoError(t, service.epochBoundaryStateCache.put(fRoot, beaconState))
	require.NoError(t, service.MigrateToCold(ctx, fRoot))

	gotState, err := service.beaconDB.State(ctx, fRoot)
	require.NoError(t, err)
	assert.DeepEqual(t, beaconState.InnerStateUnsafe(), gotState.InnerStateUnsafe(), "Did not save state")
	gotRoot := service.beaconDB.ArchivedPointRoot(ctx, stateSlot/service.slotsPerArchivedPoint)
	assert.Equal(t, fRoot, gotRoot, "Did not save archived root")
	lastIndex, err := service.beaconDB.LastArchivedIndex(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), lastIndex, "Did not save last archived index")

	testutil.AssertLogsContain(t, hook, "Saved state in DB")
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
	blk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 2}}
	fRoot, err := stateutil.BlockRoot(blk.Block)
	require.NoError(t, err)
	require.NoError(t, service.beaconDB.SaveBlock(ctx, blk))
	require.NoError(t, service.beaconDB.SaveGenesisBlockRoot(ctx, fRoot))
	if err := service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{
		Slot: 1,
		Root: fRoot[:],
	}); err != nil {
		t.Fatal(err)
	}
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
	lastIndex, err := service.beaconDB.LastArchivedIndex(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), lastIndex, "Did not save last archived index")

	testutil.AssertLogsContain(t, hook, "Saved state in DB")
}
