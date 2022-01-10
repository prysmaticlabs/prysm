package stategen

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestStateByRoot_GenesisState(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	service := New(beaconDB)
	b := util.NewBeaconBlock()
	bRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	beaconState, _ := util.DeterministicGenesisState(t, 32)
	require.NoError(t, service.beaconDB.SaveState(ctx, beaconState, bRoot))
	require.NoError(t, service.beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b)))
	require.NoError(t, service.beaconDB.SaveGenesisBlockRoot(ctx, bRoot))
	loadedState, err := service.StateByRoot(ctx, params.BeaconConfig().ZeroHash) // Zero hash is genesis state root.
	require.NoError(t, err)
	require.DeepSSZEqual(t, loadedState.InnerStateUnsafe(), beaconState.InnerStateUnsafe())
}

func TestStateByRoot_ColdState(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	service := New(beaconDB)
	service.finalizedInfo.slot = 2
	service.slotsPerArchivedPoint = 1

	b := util.NewBeaconBlock()
	b.Block.Slot = 1
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b)))
	bRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	beaconState, _ := util.DeterministicGenesisState(t, 32)
	require.NoError(t, beaconState.SetSlot(1))
	require.NoError(t, service.beaconDB.SaveState(ctx, beaconState, bRoot))
	require.NoError(t, service.beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b)))
	require.NoError(t, service.beaconDB.SaveGenesisBlockRoot(ctx, bRoot))
	loadedState, err := service.StateByRoot(ctx, bRoot)
	require.NoError(t, err)
	require.DeepSSZEqual(t, loadedState.InnerStateUnsafe(), beaconState.InnerStateUnsafe())
}

func TestStateByRootIfCachedNoCopy_HotState(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	service := New(beaconDB)

	beaconState, _ := util.DeterministicGenesisState(t, 32)
	r := [32]byte{'A'}
	require.NoError(t, service.beaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Root: r[:]}))
	service.hotStateCache.put(r, beaconState)

	loadedState := service.StateByRootIfCachedNoCopy(r)
	require.DeepSSZEqual(t, loadedState.InnerStateUnsafe(), beaconState.InnerStateUnsafe())
}

func TestStateByRootIfCachedNoCopy_ColdState(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	service := New(beaconDB)
	service.finalizedInfo.slot = 2
	service.slotsPerArchivedPoint = 1

	b := util.NewBeaconBlock()
	b.Block.Slot = 1
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b)))
	bRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	beaconState, _ := util.DeterministicGenesisState(t, 32)
	require.NoError(t, beaconState.SetSlot(1))
	require.NoError(t, service.beaconDB.SaveState(ctx, beaconState, bRoot))
	require.NoError(t, service.beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b)))
	require.NoError(t, service.beaconDB.SaveGenesisBlockRoot(ctx, bRoot))
	loadedState := service.StateByRootIfCachedNoCopy(bRoot)
	require.NoError(t, err)
	require.Equal(t, loadedState, nil)
}

func TestStateByRoot_HotStateUsingEpochBoundaryCacheNoReplay(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	service := New(beaconDB)

	beaconState, _ := util.DeterministicGenesisState(t, 32)
	require.NoError(t, beaconState.SetSlot(10))
	blk := util.NewBeaconBlock()
	blkRoot, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.beaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Root: blkRoot[:]}))
	require.NoError(t, service.epochBoundaryStateCache.put(blkRoot, beaconState))
	loadedState, err := service.StateByRoot(ctx, blkRoot)
	require.NoError(t, err)
	assert.Equal(t, types.Slot(10), loadedState.Slot(), "Did not correctly load state")
}

func TestStateByRoot_HotStateUsingEpochBoundaryCacheWithReplay(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	service := New(beaconDB)

	beaconState, _ := util.DeterministicGenesisState(t, 32)
	blk := util.NewBeaconBlock()
	blkRoot, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.epochBoundaryStateCache.put(blkRoot, beaconState))
	targetSlot := types.Slot(10)
	targetBlock := util.NewBeaconBlock()
	targetBlock.Block.Slot = 11
	targetBlock.Block.ParentRoot = blkRoot[:]
	targetBlock.Block.ProposerIndex = 8
	require.NoError(t, service.beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(targetBlock)))
	targetRoot, err := targetBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.beaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Slot: targetSlot, Root: targetRoot[:]}))
	loadedState, err := service.StateByRoot(ctx, targetRoot)
	require.NoError(t, err)
	assert.Equal(t, targetSlot, loadedState.Slot(), "Did not correctly load state")
}

func TestStateByRoot_HotStateCached(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	service := New(beaconDB)

	beaconState, _ := util.DeterministicGenesisState(t, 32)
	r := [32]byte{'A'}
	require.NoError(t, service.beaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Root: r[:]}))
	service.hotStateCache.put(r, beaconState)

	loadedState, err := service.StateByRoot(ctx, r)
	require.NoError(t, err)
	require.DeepSSZEqual(t, loadedState.InnerStateUnsafe(), beaconState.InnerStateUnsafe())
}

func TestStateByRoot_StateByRootInitialSync(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	service := New(beaconDB)
	b := util.NewBeaconBlock()
	bRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	beaconState, _ := util.DeterministicGenesisState(t, 32)
	require.NoError(t, service.beaconDB.SaveState(ctx, beaconState, bRoot))
	require.NoError(t, service.beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b)))
	require.NoError(t, service.beaconDB.SaveGenesisBlockRoot(ctx, bRoot))
	loadedState, err := service.StateByRootInitialSync(ctx, params.BeaconConfig().ZeroHash) // Zero hash is genesis state root.
	require.NoError(t, err)
	require.DeepSSZEqual(t, loadedState.InnerStateUnsafe(), beaconState.InnerStateUnsafe())
}

func TestStateByRootInitialSync_UseEpochStateCache(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	service := New(beaconDB)

	beaconState, _ := util.DeterministicGenesisState(t, 32)
	targetSlot := types.Slot(10)
	require.NoError(t, beaconState.SetSlot(targetSlot))
	blk := util.NewBeaconBlock()
	blkRoot, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.epochBoundaryStateCache.put(blkRoot, beaconState))
	loadedState, err := service.StateByRootInitialSync(ctx, blkRoot)
	require.NoError(t, err)
	assert.Equal(t, targetSlot, loadedState.Slot(), "Did not correctly load state")
}

func TestStateByRootInitialSync_UseCache(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	service := New(beaconDB)

	beaconState, _ := util.DeterministicGenesisState(t, 32)
	r := [32]byte{'A'}
	require.NoError(t, service.beaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Root: r[:]}))
	service.hotStateCache.put(r, beaconState)

	loadedState, err := service.StateByRootInitialSync(ctx, r)
	require.NoError(t, err)
	require.DeepSSZEqual(t, loadedState.InnerStateUnsafe(), beaconState.InnerStateUnsafe())
	if service.hotStateCache.has(r) {
		t.Error("Hot state cache was not invalidated")
	}
}

func TestStateByRootInitialSync_CanProcessUpTo(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service := New(beaconDB)

	beaconState, _ := util.DeterministicGenesisState(t, 32)
	blk := util.NewBeaconBlock()
	blkRoot, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.epochBoundaryStateCache.put(blkRoot, beaconState))
	targetSlot := types.Slot(10)
	targetBlk := util.NewBeaconBlock()
	targetBlk.Block.Slot = 11
	targetBlk.Block.ParentRoot = blkRoot[:]
	targetRoot, err := targetBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(targetBlk)))
	require.NoError(t, service.beaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Slot: targetSlot, Root: targetRoot[:]}))

	loadedState, err := service.StateByRootInitialSync(ctx, targetRoot)
	require.NoError(t, err)
	assert.Equal(t, targetSlot, loadedState.Slot(), "Did not correctly load state")
}

func TestStateBySlot_ColdState(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	service := New(beaconDB)
	service.slotsPerArchivedPoint = params.BeaconConfig().SlotsPerEpoch * 2
	service.finalizedInfo.slot = service.slotsPerArchivedPoint + 1

	beaconState, pks := util.DeterministicGenesisState(t, 32)
	genesisStateRoot, err := beaconState.HashTreeRoot(ctx)
	require.NoError(t, err)
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	assert.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesis)))
	gRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err)
	assert.NoError(t, beaconDB.SaveState(ctx, beaconState, gRoot))
	assert.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, gRoot))

	b, err := util.GenerateFullBlock(beaconState, pks, util.DefaultBlockGenConfig(), 1)
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b)))
	bRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveState(ctx, beaconState, bRoot))
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, bRoot))

	r := [32]byte{}
	require.NoError(t, service.beaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Slot: service.slotsPerArchivedPoint, Root: r[:]}))

	slot := types.Slot(20)
	loadedState, err := service.StateBySlot(ctx, slot)
	require.NoError(t, err)
	assert.Equal(t, slot, loadedState.Slot(), "Did not correctly save state")
}

func TestStateBySlot_HotStateDB(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	service := New(beaconDB)

	beaconState, _ := util.DeterministicGenesisState(t, 32)
	genesisStateRoot, err := beaconState.HashTreeRoot(ctx)
	require.NoError(t, err)
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	assert.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesis)))
	gRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err)
	assert.NoError(t, beaconDB.SaveState(ctx, beaconState, gRoot))
	assert.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, gRoot))

	slot := types.Slot(10)
	loadedState, err := service.StateBySlot(ctx, slot)
	require.NoError(t, err)
	assert.Equal(t, slot, loadedState.Slot(), "Did not correctly load state")
}

func TestLoadeStateByRoot_Cached(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service := New(beaconDB)

	beaconState, _ := util.DeterministicGenesisState(t, 32)
	r := [32]byte{'A'}
	service.hotStateCache.put(r, beaconState)

	// This tests where hot state was already cached.
	loadedState, err := service.loadStateByRoot(ctx, r)
	require.NoError(t, err)
	require.DeepSSZEqual(t, loadedState.InnerStateUnsafe(), beaconState.InnerStateUnsafe())
}

func TestLoadeStateByRoot_FinalizedState(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service := New(beaconDB)

	beaconState, _ := util.DeterministicGenesisState(t, 32)
	genesisStateRoot, err := beaconState.HashTreeRoot(ctx)
	require.NoError(t, err)
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	assert.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesis)))
	gRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.beaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Slot: 0, Root: gRoot[:]}))

	service.finalizedInfo.state = beaconState
	service.finalizedInfo.slot = beaconState.Slot()
	service.finalizedInfo.root = gRoot

	// This tests where hot state was already cached.
	loadedState, err := service.loadStateByRoot(ctx, gRoot)
	require.NoError(t, err)
	require.DeepSSZEqual(t, loadedState.InnerStateUnsafe(), beaconState.InnerStateUnsafe())
}

func TestLoadeStateByRoot_EpochBoundaryStateCanProcess(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service := New(beaconDB)

	beaconState, _ := util.DeterministicGenesisState(t, 32)
	gBlk := util.NewBeaconBlock()
	gBlkRoot, err := gBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.epochBoundaryStateCache.put(gBlkRoot, beaconState))

	blk := util.NewBeaconBlock()
	blk.Block.Slot = 11
	blk.Block.ProposerIndex = 8
	blk.Block.ParentRoot = gBlkRoot[:]
	require.NoError(t, service.beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(blk)))
	blkRoot, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.beaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Slot: 10, Root: blkRoot[:]}))

	// This tests where hot state was not cached and needs processing.
	loadedState, err := service.loadStateByRoot(ctx, blkRoot)
	require.NoError(t, err)
	assert.Equal(t, types.Slot(10), loadedState.Slot(), "Did not correctly load state")
}

func TestLoadeStateByRoot_FromDBBoundaryCase(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service := New(beaconDB)

	beaconState, _ := util.DeterministicGenesisState(t, 32)
	gBlk := util.NewBeaconBlock()
	gBlkRoot, err := gBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.epochBoundaryStateCache.put(gBlkRoot, beaconState))

	blk := util.NewBeaconBlock()
	blk.Block.Slot = 11
	blk.Block.ProposerIndex = 8
	blk.Block.ParentRoot = gBlkRoot[:]
	require.NoError(t, service.beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(blk)))
	blkRoot, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.beaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Slot: 10, Root: blkRoot[:]}))

	// This tests where hot state was not cached and needs processing.
	loadedState, err := service.loadStateByRoot(ctx, blkRoot)
	require.NoError(t, err)
	assert.Equal(t, types.Slot(10), loadedState.Slot(), "Did not correctly load state")
}

func TestLoadeStateBySlot_CanAdvanceSlotUsingDB(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service := New(beaconDB)
	beaconState, _ := util.DeterministicGenesisState(t, 32)
	b := util.NewBeaconBlock()
	require.NoError(t, service.beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b)))
	gRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.beaconDB.SaveGenesisBlockRoot(ctx, gRoot))
	require.NoError(t, service.beaconDB.SaveState(ctx, beaconState, gRoot))

	slot := types.Slot(10)
	loadedState, err := service.loadStateBySlot(ctx, slot)
	require.NoError(t, err)
	assert.Equal(t, slot, loadedState.Slot(), "Did not correctly load state")
}

func TestLoadeStateBySlot_CanReplayBlock(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service := New(beaconDB)
	genesis, keys := util.DeterministicGenesisState(t, 64)
	genesisBlockRoot := bytesutil.ToBytes32(nil)
	require.NoError(t, beaconDB.SaveState(ctx, genesis, genesisBlockRoot))
	stateRoot, err := genesis.HashTreeRoot(ctx)
	require.NoError(t, err)
	genesisBlk := blocks.NewGenesisBlock(stateRoot[:])
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesisBlk)))
	genesisBlkRoot, err := genesisBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, genesisBlkRoot))

	b1, err := util.GenerateFullBlock(genesis, keys, util.DefaultBlockGenConfig(), 1)
	assert.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b1)))
	r1, err := b1.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.beaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Slot: 1, Root: r1[:]}))
	service.hotStateCache.put(bytesutil.ToBytes32(b1.Block.ParentRoot), genesis)

	loadedState, err := service.loadStateBySlot(ctx, 2)
	require.NoError(t, err)
	assert.Equal(t, types.Slot(2), loadedState.Slot(), "Did not correctly load state")
}

func TestLoadeStateBySlot_DoesntReplayBlockOnRequestedSlot(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service := New(beaconDB)
	genesis, keys := util.DeterministicGenesisState(t, 64)
	genesisBlockRoot := bytesutil.ToBytes32(nil)
	require.NoError(t, beaconDB.SaveState(ctx, genesis, genesisBlockRoot))
	stateRoot, err := genesis.HashTreeRoot(ctx)
	require.NoError(t, err)
	genesisBlk := blocks.NewGenesisBlock(stateRoot[:])
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesisBlk)))
	genesisBlkRoot, err := genesisBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, genesisBlkRoot))

	b1, err := util.GenerateFullBlock(genesis, keys, util.DefaultBlockGenConfig(), 1)
	assert.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b1)))
	r1, err := b1.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.beaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Slot: 1, Root: r1[:]}))
	service.hotStateCache.put(bytesutil.ToBytes32(b1.Block.ParentRoot), genesis)

	loadedState, err := service.loadStateBySlot(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, types.Slot(1), loadedState.Slot(), "Did not correctly load state")

	// Latest block header's state root should not be zero. Zero means the current slot's block has been processed.
	require.NotEqual(t, params.BeaconConfig().ZeroHash, bytesutil.ToBytes32(loadedState.LatestBlockHeader().StateRoot))
}

func TestLastAncestorState_CanGetUsingDB(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service := New(beaconDB)

	b0 := util.NewBeaconBlock()
	b0.Block.ParentRoot = bytesutil.PadTo([]byte{'a'}, 32)
	r0, err := b0.Block.HashTreeRoot()
	require.NoError(t, err)
	b1 := util.NewBeaconBlock()
	b1.Block.Slot = 1
	b1.Block.ParentRoot = bytesutil.PadTo(r0[:], 32)
	r1, err := b1.Block.HashTreeRoot()
	require.NoError(t, err)
	b2 := util.NewBeaconBlock()
	b2.Block.Slot = 2
	b2.Block.ParentRoot = bytesutil.PadTo(r1[:], 32)
	r2, err := b2.Block.HashTreeRoot()
	require.NoError(t, err)
	b3 := util.NewBeaconBlock()
	b3.Block.Slot = 3
	b3.Block.ParentRoot = bytesutil.PadTo(r2[:], 32)
	r3, err := b3.Block.HashTreeRoot()
	require.NoError(t, err)

	b1State, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, b1State.SetSlot(1))

	require.NoError(t, service.beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b0)))
	require.NoError(t, service.beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b1)))
	require.NoError(t, service.beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b2)))
	require.NoError(t, service.beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b3)))
	require.NoError(t, service.beaconDB.SaveState(ctx, b1State, r1))

	lastState, err := service.lastAncestorState(ctx, r3)
	require.NoError(t, err)
	assert.Equal(t, b1State.Slot(), lastState.Slot(), "Did not get wanted state")
}

func TestLastAncestorState_CanGetUsingCache(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service := New(beaconDB)

	b0 := util.NewBeaconBlock()
	b0.Block.ParentRoot = bytesutil.PadTo([]byte{'a'}, 32)
	r0, err := b0.Block.HashTreeRoot()
	require.NoError(t, err)
	b1 := util.NewBeaconBlock()
	b1.Block.Slot = 1
	b1.Block.ParentRoot = bytesutil.PadTo(r0[:], 32)
	r1, err := b1.Block.HashTreeRoot()
	require.NoError(t, err)
	b2 := util.NewBeaconBlock()
	b2.Block.Slot = 2
	b2.Block.ParentRoot = bytesutil.PadTo(r1[:], 32)
	r2, err := b2.Block.HashTreeRoot()
	require.NoError(t, err)
	b3 := util.NewBeaconBlock()
	b3.Block.Slot = 3
	b3.Block.ParentRoot = bytesutil.PadTo(r2[:], 32)
	r3, err := b3.Block.HashTreeRoot()
	require.NoError(t, err)

	b1State, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, b1State.SetSlot(1))

	require.NoError(t, service.beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b0)))
	require.NoError(t, service.beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b1)))
	require.NoError(t, service.beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b2)))
	require.NoError(t, service.beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b3)))
	service.hotStateCache.put(r1, b1State)

	lastState, err := service.lastAncestorState(ctx, r3)
	require.NoError(t, err)
	assert.Equal(t, b1State.Slot(), lastState.Slot(), "Did not get wanted state")
}

func TestState_HasState(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service := New(beaconDB)
	s, err := util.NewBeaconState()
	require.NoError(t, err)
	rHit1 := [32]byte{1}
	rHit2 := [32]byte{2}
	rMiss := [32]byte{3}
	service.hotStateCache.put(rHit1, s)
	require.NoError(t, service.epochBoundaryStateCache.put(rHit2, s))

	b := util.NewBeaconBlock()
	rHit3, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.beaconDB.SaveState(ctx, s, rHit3))
	tt := []struct {
		root [32]byte
		want bool
	}{
		{rHit1, true},
		{rHit2, true},
		{rMiss, false},
		{rHit3, true},
	}
	for _, tc := range tt {
		got, err := service.HasState(ctx, tc.root)
		require.NoError(t, err)
		require.Equal(t, tc.want, got)
	}
}

func TestState_HasStateInCache(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service := New(beaconDB)
	s, err := util.NewBeaconState()
	require.NoError(t, err)
	rHit1 := [32]byte{1}
	rHit2 := [32]byte{2}
	rMiss := [32]byte{3}
	service.hotStateCache.put(rHit1, s)
	require.NoError(t, service.epochBoundaryStateCache.put(rHit2, s))

	tt := []struct {
		root [32]byte
		want bool
	}{
		{rHit1, true},
		{rHit2, true},
		{rMiss, false},
	}
	for _, tc := range tt {
		got, err := service.HasStateInCache(ctx, tc.root)
		require.NoError(t, err)
		require.Equal(t, tc.want, got)
	}
}
