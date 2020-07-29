package stategen

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestSaveHotState_AlreadyHas(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)
	service := New(db, cache.NewStateSummaryCache())

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	require.NoError(t, beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch))
	r := [32]byte{'A'}

	// Pre cache the hot state.
	service.hotStateCache.Put(r, beaconState)
	require.NoError(t, service.saveHotState(ctx, r, beaconState))

	// Should not save the state and state summary.
	assert.Equal(t, false, service.beaconDB.HasState(ctx, r), "Should not have saved the state")
	assert.Equal(t, false, service.beaconDB.HasStateSummary(ctx, r), "Should have saved the state summary")
	testutil.AssertLogsDoNotContain(t, hook, "Saved full state on epoch boundary")
}

func TestSaveHotState_CanSaveOnEpochBoundary(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)
	service := New(db, cache.NewStateSummaryCache())

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	require.NoError(t, beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch))
	r := [32]byte{'A'}

	require.NoError(t, service.saveHotState(ctx, r, beaconState))

	// Should save both state and state summary.
	_, ok, err := service.epochBoundaryStateCache.getByRoot(r)
	require.NoError(t, err)
	require.Equal(t, true, ok, "Did not save epoch boundary state")
	assert.Equal(t, true, service.stateSummaryCache.Has(r), "Should have saved the state summary")
}

func TestSaveHotState_NoSaveNotEpochBoundary(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)
	service := New(db, cache.NewStateSummaryCache())

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	require.NoError(t, beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch-1))
	r := [32]byte{'A'}
	b := testutil.NewBeaconBlock()
	require.NoError(t, db.SaveBlock(ctx, b))
	gRoot, err := stateutil.BlockRoot(b.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, gRoot))
	require.NoError(t, service.saveHotState(ctx, r, beaconState))

	// Should only save state summary.
	assert.Equal(t, false, service.beaconDB.HasState(ctx, r), "Should not have saved the state")
	assert.Equal(t, true, service.stateSummaryCache.Has(r), "Should have saved the state summary")
	testutil.AssertLogsDoNotContain(t, hook, "Saved full state on epoch boundary")
}

func TestLoadHoteStateByRoot_Cached(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)
	service := New(db, cache.NewStateSummaryCache())

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	r := [32]byte{'A'}
	service.hotStateCache.Put(r, beaconState)

	// This tests where hot state was already cached.
	loadedState, err := service.loadHotStateByRoot(ctx, r)
	require.NoError(t, err)

	if !proto.Equal(loadedState.InnerStateUnsafe(), beaconState.InnerStateUnsafe()) {
		t.Error("Did not correctly cache state")
	}
}

func TestLoadHoteStateByRoot_EpochBoundaryStateCanProcess(t *testing.T) {
	ctx := context.Background()
	db, ssc := testDB.SetupDB(t)
	service := New(db, ssc)

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	gBlk := testutil.NewBeaconBlock()
	gBlkRoot, err := stateutil.BlockRoot(gBlk.Block)
	require.NoError(t, err)
	require.NoError(t, service.epochBoundaryStateCache.put(gBlkRoot, beaconState))

	blk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 11, ParentRoot: gBlkRoot[:], ProposerIndex: 8}}
	require.NoError(t, service.beaconDB.SaveBlock(ctx, blk))
	blkRoot, err := stateutil.BlockRoot(blk.Block)
	require.NoError(t, err)
	if err := service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{
		Slot: 10,
		Root: blkRoot[:],
	}); err != nil {
		t.Fatal(err)
	}

	// This tests where hot state was not cached and needs processing.
	loadedState, err := service.loadHotStateByRoot(ctx, blkRoot)
	require.NoError(t, err)
	assert.Equal(t, uint64(10), loadedState.Slot(), "Did not correctly load state")
}

func TestLoadHoteStateByRoot_FromDBBoundaryCase(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)
	service := New(db, cache.NewStateSummaryCache())

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	blk := testutil.NewBeaconBlock()
	blkRoot, err := stateutil.BlockRoot(blk.Block)
	require.NoError(t, err)
	require.NoError(t, service.epochBoundaryStateCache.put(blkRoot, beaconState))
	targetSlot := uint64(0)
	if err := service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{
		Slot: targetSlot,
		Root: blkRoot[:],
	}); err != nil {
		t.Fatal(err)
	}

	// This tests where hot state was not cached but doesn't need processing
	// because it on the epoch boundary slot.
	loadedState, err := service.loadHotStateByRoot(ctx, blkRoot)
	require.NoError(t, err)
	assert.Equal(t, targetSlot, loadedState.Slot(), "Did not correctly load state")
}

func TestLoadHoteStateBySlot_CanAdvanceSlotUsingDB(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)
	service := New(db, cache.NewStateSummaryCache())
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	b := testutil.NewBeaconBlock()
	require.NoError(t, service.beaconDB.SaveBlock(ctx, b))
	gRoot, err := stateutil.BlockRoot(b.Block)
	require.NoError(t, err)
	require.NoError(t, service.beaconDB.SaveGenesisBlockRoot(ctx, gRoot))
	require.NoError(t, service.beaconDB.SaveState(ctx, beaconState, gRoot))

	slot := uint64(10)
	loadedState, err := service.loadHotStateBySlot(ctx, slot)
	require.NoError(t, err)
	assert.Equal(t, slot, loadedState.Slot(), "Did not correctly load state")
}

func TestLastAncestorState_CanGet(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)
	service := New(db, cache.NewStateSummaryCache())

	b0 := testutil.NewBeaconBlock()
	b0.Block.ParentRoot = bytesutil.PadTo([]byte{'a'}, 32)
	r0, err := b0.Block.HashTreeRoot()
	require.NoError(t, err)
	b1 := testutil.NewBeaconBlock()
	b1.Block.Slot = 1
	b1.Block.ParentRoot = bytesutil.PadTo(r0[:], 32)
	r1, err := b1.Block.HashTreeRoot()
	require.NoError(t, err)
	b2 := testutil.NewBeaconBlock()
	b2.Block.Slot = 2
	b2.Block.ParentRoot = bytesutil.PadTo(r1[:], 32)
	r2, err := b2.Block.HashTreeRoot()
	require.NoError(t, err)
	b3 := testutil.NewBeaconBlock()
	b3.Block.Slot = 3
	b3.Block.ParentRoot = bytesutil.PadTo(r2[:], 32)
	r3, err := b3.Block.HashTreeRoot()
	require.NoError(t, err)

	b1State := testutil.NewBeaconState()
	require.NoError(t, b1State.SetSlot(1))

	require.NoError(t, service.beaconDB.SaveBlock(ctx, b0))
	require.NoError(t, service.beaconDB.SaveBlock(ctx, b1))
	require.NoError(t, service.beaconDB.SaveBlock(ctx, b2))
	require.NoError(t, service.beaconDB.SaveBlock(ctx, b3))
	require.NoError(t, service.beaconDB.SaveState(ctx, b1State, r1))

	lastState, err := service.lastAncestorState(ctx, r3)
	require.NoError(t, err)
	assert.Equal(t, b1State.Slot(), lastState.Slot(), "Did not get wanted state")
}
