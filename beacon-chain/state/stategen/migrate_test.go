package stategen

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/blocks"
	testDB "github.com/prysmaticlabs/prysm/v3/beacon-chain/db/testing"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestMigrateToCold_CanSaveFinalizedInfo(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service := New(beaconDB, newTestSaver(beaconDB))
	beaconState, _ := util.DeterministicGenesisState(t, 32)
	b := util.NewBeaconBlock()
	b.Block.Slot = 1
	br, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	util.SaveBlock(t, ctx, service.beaconDB, b)
	require.NoError(t, service.epochBoundaryStateCache.put(br, beaconState))
	require.NoError(t, service.MigrateToCold(ctx, br))

	wanted := &finalizedInfo{state: beaconState, root: br, slot: 1}
	assert.DeepEqual(t, wanted, service.finalizedInfo, "Incorrect finalized info")
}

func TestMigrateToCold_HappyPath(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	var zero, one, two, three, four, five types.Slot = 50, 51, 150, 151, 152, 200
	specs := []mockHistorySpec{
		{slot: zero},
		{slot: one, savedState: true},
		{slot: two},
		{slot: three},
		{slot: four},
		{slot: five, canonicalBlock: true},
	}

	hist := newMockHistory(t, specs, five+1)
	ch := NewCanonicalHistory(hist, hist, hist)
	service := New(beaconDB, newTestSaver(beaconDB), WithReplayerBuilder(ch))

	service.slotsPerArchivedPoint = 1
	beaconState, _ := util.DeterministicGenesisState(t, 32)
	stateSlot := types.Slot(1)
	require.NoError(t, beaconState.SetSlot(stateSlot))
	b := util.NewBeaconBlock()
	b.Block.Slot = 2
	fRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	util.SaveBlock(t, ctx, service.beaconDB, b)
	require.NoError(t, service.epochBoundaryStateCache.put(fRoot, beaconState))
	require.NoError(t, service.MigrateToCold(ctx, fRoot))

	gotState, err := service.beaconDB.State(ctx, fRoot)
	require.NoError(t, err)
	assert.DeepSSZEqual(t, beaconState.InnerStateUnsafe(), gotState.InnerStateUnsafe(), "Did not save state")
	gotRoot := service.beaconDB.ArchivedPointRoot(ctx, stateSlot/service.slotsPerArchivedPoint)
	assert.Equal(t, fRoot, gotRoot, "Did not save archived root")
	lastIndex, err := service.beaconDB.LastArchivedSlot(ctx)
	require.NoError(t, err)
	assert.Equal(t, types.Slot(1), lastIndex, "Did not save last archived index")

	require.LogsContain(t, hook, "Saved state in DB")
}

func TestMigrateToCold_RegeneratePath(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	mockCanon := newMockCanonicalMap()
	// picking 5 because the highest block in the below setup code is 4 and we're not testing
	// slot bounds weirdness here.
	ch := NewCanonicalHistory(beaconDB, mockCanon, &mockCurrentSlotter{Slot: 5})
	service := New(beaconDB, newTestSaver(beaconDB), WithReplayerBuilder(ch))
	service.slotsPerArchivedPoint = 1
	beaconState, pks := util.DeterministicGenesisState(t, 32)
	genesisStateRoot, err := beaconState.HashTreeRoot(ctx)
	require.NoError(t, err)
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	util.SaveBlock(t, ctx, beaconDB, genesis)
	gRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err)
	assert.NoError(t, beaconDB.SaveState(ctx, beaconState, gRoot))
	assert.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, gRoot))

	b1, err := util.GenerateFullBlock(beaconState, pks, util.DefaultBlockGenConfig(), 1)
	require.NoError(t, err)
	r1, err := b1.Block.HashTreeRoot()
	require.NoError(t, err)
	mockCanon.AddCanonical(r1)
	util.SaveBlock(t, ctx, service.beaconDB, b1)
	require.NoError(t, service.beaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Slot: 1, Root: r1[:]}))

	b4, err := util.GenerateFullBlock(beaconState, pks, util.DefaultBlockGenConfig(), 4)
	require.NoError(t, err)
	r4, err := b4.Block.HashTreeRoot()
	require.NoError(t, err)
	mockCanon.AddCanonical(r4)
	util.SaveBlock(t, ctx, service.beaconDB, b4)
	require.NoError(t, service.beaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Slot: 4, Root: r4[:]}))
	service.finalizedInfo = &finalizedInfo{
		slot:  0,
		root:  genesisStateRoot,
		state: beaconState,
	}

	require.NoError(t, service.MigrateToCold(ctx, r4))

	s1, err := service.beaconDB.State(ctx, r1)
	require.NoError(t, err)
	require.NotEqual(t, nil, s1)

	require.Equal(t, false, service.beaconDB.HasState(ctx, r4))
	s4, err := service.beaconDB.State(ctx, r4)
	require.NoError(t, err)
	require.Equal(t, nil, s4)

	require.LogsContain(t, hook, "Saved state in DB")
}

// TODO!!! fix this migrate to cold test
/*
func TestMigrateToCold_StateExistsInDB(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	service := New(beaconDB)
	service.slotsPerArchivedPoint = 1
	beaconState, _ := util.DeterministicGenesisState(t, 32)
	stateSlot := types.Slot(1)
	require.NoError(t, beaconState.SetSlot(stateSlot))
	b := util.NewBeaconBlock()
	b.Block.Slot = 2
	fRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	util.SaveBlock(t, ctx, service.beaconDB, b)
	require.NoError(t, service.epochBoundaryStateCache.put(fRoot, beaconState))
	require.NoError(t, service.beaconDB.SaveState(ctx, beaconState, fRoot))

	service.saver.savedRoots = [][32]byte{{1}, {2}, {3}, {4}, fRoot}
	require.NoError(t, service.MigrateToCold(ctx, fRoot))
	assert.DeepEqual(t, [][32]byte{{1}, {2}, {3}, {4}}, service.saver.savedRoots)
	assert.LogsDoNotContain(t, hook, "Saved state in DB")
}
*/
