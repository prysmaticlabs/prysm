package stategen

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"

	//pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestSaveHotState_AlreadyHas(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)
	service := New(db, cache.NewStateSummaryCache())

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	if err := beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch); err != nil {
		t.Fatal(err)
	}
	r := [32]byte{'A'}

	// Pre cache the hot state.
	service.hotStateCache.Put(r, beaconState)
	if err := service.saveHotState(ctx, r, beaconState); err != nil {
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

func TestSaveHotState_CanSaveOnEpochBoundary(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)
	service := New(db, cache.NewStateSummaryCache())

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	if err := beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch); err != nil {
		t.Fatal(err)
	}
	r := [32]byte{'A'}

	if err := service.saveHotState(ctx, r, beaconState); err != nil {
		t.Fatal(err)
	}

	// Should save both state and state summary.
	_, ok, err := service.epochBoundaryStateCache.getByRoot(r)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("Did not save epoch boundary state")
	}
	if !service.stateSummaryCache.Has(r) {
		t.Error("Should have saved the state summary")
	}
}

func TestSaveHotState_NoSaveNotEpochBoundary(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)
	service := New(db, cache.NewStateSummaryCache())

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	if err := beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch - 1); err != nil {
		t.Fatal(err)
	}
	r := [32]byte{'A'}
	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	if err := db.SaveBlock(ctx, b); err != nil {
		t.Fatal(err)
	}
	gRoot, err := stateutil.BlockRoot(b.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveGenesisBlockRoot(ctx, gRoot); err != nil {
		t.Fatal(err)
	}

	if err := service.saveHotState(ctx, r, beaconState); err != nil {
		t.Fatal(err)
	}

	// Should only save state summary.
	if service.beaconDB.HasState(ctx, r) {
		t.Error("Should not have saved the state")
	}
	if !service.stateSummaryCache.Has(r) {
		t.Error("Should have saved the state summary")
	}
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
	if err != nil {
		t.Fatal(err)
	}

	if !proto.Equal(loadedState.InnerStateUnsafe(), beaconState.InnerStateUnsafe()) {
		t.Error("Did not correctly cache state")
	}
}

func TestLoadHoteStateByRoot_EpochBoundaryStateCanProcess(t *testing.T) {
	ctx := context.Background()
	db, ssc := testDB.SetupDB(t)
	service := New(db, ssc)

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	gBlk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	gBlkRoot, err := stateutil.BlockRoot(gBlk.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.epochBoundaryStateCache.put(gBlkRoot, beaconState); err != nil {
		t.Fatal(err)
	}

	blk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 11, ParentRoot: gBlkRoot[:], ProposerIndex: 8}}
	if err := service.beaconDB.SaveBlock(ctx, blk); err != nil {
		t.Fatal(err)
	}
	blkRoot, err := stateutil.BlockRoot(blk.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{
		Slot: 10,
		Root: blkRoot[:],
	}); err != nil {
		t.Fatal(err)
	}

	// This tests where hot state was not cached and needs processing.
	loadedState, err := service.loadHotStateByRoot(ctx, blkRoot)
	if err != nil {
		t.Fatal(err)
	}

	if loadedState.Slot() != 10 {
		t.Error("Did not correctly load state")
	}
}

func TestLoadHoteStateByRoot_FromDBBoundaryCase(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)
	service := New(db, cache.NewStateSummaryCache())

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	blk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	blkRoot, err := stateutil.BlockRoot(blk.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.epochBoundaryStateCache.put(blkRoot, beaconState); err != nil {
		t.Fatal(err)
	}
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
	if err != nil {
		t.Fatal(err)
	}
	if loadedState.Slot() != targetSlot {
		t.Error("Did not correctly load state")
	}
}

func TestLoadHoteStateBySlot_CanAdvanceSlotUsingDB(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)
	service := New(db, cache.NewStateSummaryCache())
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	if err := service.beaconDB.SaveBlock(ctx, b); err != nil {
		t.Fatal(err)
	}
	gRoot, err := stateutil.BlockRoot(b.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveGenesisBlockRoot(ctx, gRoot); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveState(ctx, beaconState, gRoot); err != nil {
		t.Fatal(err)
	}

	slot := uint64(10)
	loadedState, err := service.loadHotStateBySlot(ctx, slot)
	if err != nil {
		t.Fatal(err)
	}
	if loadedState.Slot() != slot {
		t.Error("Did not correctly load state")
	}
}

func TestLastAncestorState_CanGet(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)
	service := New(db, cache.NewStateSummaryCache())

	b0 := testutil.NewBeaconBlock()
	b0.Block.ParentRoot = bytesutil.PadTo([]byte{'a'}, 32)
	r0, err := ssz.HashTreeRoot(b0.Block)
	if err != nil {
		t.Fatal(err)
	}
	b1 := testutil.NewBeaconBlock()
	b1.Block.Slot = 1
	b1.Block.ParentRoot = bytesutil.PadTo(r0[:], 32)
	r1, err := ssz.HashTreeRoot(b1.Block)
	if err != nil {
		t.Fatal(err)
	}
	b2 := testutil.NewBeaconBlock()
	b2.Block.Slot = 2
	b2.Block.ParentRoot = bytesutil.PadTo(r1[:], 32)
	r2, err := ssz.HashTreeRoot(b2.Block)
	if err != nil {
		t.Fatal(err)
	}
	b3 := testutil.NewBeaconBlock()
	b3.Block.Slot = 3
	b3.Block.ParentRoot = bytesutil.PadTo(r2[:], 32)
	r3, err := ssz.HashTreeRoot(b3.Block)
	if err != nil {
		t.Fatal(err)
	}

	b1State := testutil.NewBeaconState()
	if err := b1State.SetSlot(1); err != nil {
		t.Fatal(err)
	}

	if err := service.beaconDB.SaveBlock(ctx, b0); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveBlock(ctx, b1); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveBlock(ctx, b2); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveBlock(ctx, b3); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveState(ctx, b1State, r1); err != nil {
		t.Fatal(err)
	}

	lastState, err := service.lastAncestorState(ctx, r3)
	if err != nil {
		t.Fatal(err)
	}
	if lastState.Slot() != b1State.Slot() {
		t.Error("Did not get wanted state")
	}
}
