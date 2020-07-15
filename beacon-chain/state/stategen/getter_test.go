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
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestStateByRoot_ColdState(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())
	service.finalizedInfo.slot = 2
	service.slotsPerArchivedPoint = 1

	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 1}}
	if err := db.SaveBlock(ctx, b); err != nil {
		t.Fatal(err)
	}
	bRoot, err := stateutil.BlockRoot(b.Block)
	if err != nil {
		t.Fatal(err)
	}
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	if err := beaconState.SetSlot(1); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveState(ctx, beaconState, bRoot); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveBlock(ctx, b); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveGenesisBlockRoot(ctx, bRoot); err != nil {
		t.Fatal(err)
	}
	r := [32]byte{'a'}
	if err := service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{
		Root: r[:],
		Slot: 1,
	}); err != nil {
		t.Fatal(err)
	}
	loadedState, err := service.StateByRoot(ctx, r)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(loadedState.InnerStateUnsafe(), beaconState.InnerStateUnsafe()) {
		t.Error("Did not correctly save state")
	}
}

func TestStateByRoot_HotStateUsingEpochBoundaryCacheNoReplay(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	if err := beaconState.SetSlot(10); err != nil {
		t.Fatal(err)
	}
	blk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	blkRoot, err := stateutil.BlockRoot(blk.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{Root: blkRoot[:]}); err != nil {
		t.Fatal(err)
	}
	if err := service.epochBoundaryStateCache.put(blkRoot, beaconState); err != nil {
		t.Fatal(err)
	}
	loadedState, err := service.StateByRoot(ctx, blkRoot)
	if err != nil {
		t.Fatal(err)
	}
	if loadedState.Slot() != 10 {
		t.Error("Did not correctly load state")
	}
}

func TestStateByRoot_HotStateUsingEpochBoundaryCacheWithReplay(t *testing.T) {
	ctx := context.Background()
	db, ssc := testDB.SetupDB(t)

	service := New(db, ssc)

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	blk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	blkRoot, err := stateutil.BlockRoot(blk.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.epochBoundaryStateCache.put(blkRoot, beaconState); err != nil {
		t.Fatal(err)
	}
	targetSlot := uint64(10)
	targetBlock := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 11, ParentRoot: blkRoot[:], ProposerIndex: 8}}
	if err := service.beaconDB.SaveBlock(ctx, targetBlock); err != nil {
		t.Fatal(err)
	}
	targetRoot, err := stateutil.BlockRoot(targetBlock.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{
		Slot: targetSlot,
		Root: targetRoot[:],
	}); err != nil {
		t.Fatal(err)
	}
	loadedState, err := service.StateByRoot(ctx, targetRoot)
	if err != nil {
		t.Fatal(err)
	}
	if loadedState.Slot() != targetSlot {
		t.Error("Did not correctly load state")
	}
}

func TestStateByRoot_HotStateCached(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	r := [32]byte{'A'}
	if err := service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{
		Root: r[:],
	}); err != nil {
		t.Fatal(err)
	}
	service.hotStateCache.Put(r, beaconState)

	loadedState, err := service.StateByRoot(ctx, r)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(loadedState.InnerStateUnsafe(), beaconState.InnerStateUnsafe()) {
		t.Error("Did not correctly cache state")
	}
}

func TestStateByRootInitialSync_UseEpochStateCache(t *testing.T) {
	ctx := context.Background()
	db, ssc := testDB.SetupDB(t)

	service := New(db, ssc)

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	targetSlot := uint64(10)
	if err := beaconState.SetSlot(targetSlot); err != nil {
		t.Fatal(err)
	}
	blk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	blkRoot, err := stateutil.BlockRoot(blk.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.epochBoundaryStateCache.put(blkRoot, beaconState); err != nil {
		t.Fatal(err)
	}
	loadedState, err := service.StateByRootInitialSync(ctx, blkRoot)
	if err != nil {
		t.Fatal(err)
	}
	if loadedState.Slot() != targetSlot {
		t.Error("Did not correctly load state")
	}
}

func TestStateByRootInitialSync_UseCache(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	r := [32]byte{'A'}
	if err := service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{
		Root: r[:],
	}); err != nil {
		t.Fatal(err)
	}
	service.hotStateCache.Put(r, beaconState)

	loadedState, err := service.StateByRoot(ctx, r)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(loadedState.InnerStateUnsafe(), beaconState.InnerStateUnsafe()) {
		t.Error("Did not correctly cache state")
	}
}

func TestStateByRootInitialSync_CanProcessUpTo(t *testing.T) {
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
	targetSlot := uint64(10)
	targetBlk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 11, ParentRoot: blkRoot[:]}}
	targetRoot, err := stateutil.BlockRoot(targetBlk.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveBlock(ctx, targetBlk); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{
		Slot: targetSlot,
		Root: targetRoot[:],
	}); err != nil {
		t.Fatal(err)
	}

	loadedState, err := service.StateByRootInitialSync(ctx, targetRoot)
	if err != nil {
		t.Fatal(err)
	}
	if loadedState.Slot() != targetSlot {
		t.Error("Did not correctly load state")
	}
}

func TestStateBySlot_ColdState(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())
	service.slotsPerArchivedPoint = params.BeaconConfig().SlotsPerEpoch * 2
	service.finalizedInfo.slot = service.slotsPerArchivedPoint + 1

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	if err := beaconState.SetSlot(1); err != nil {
		t.Fatal(err)
	}
	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 1}}
	if err := db.SaveBlock(ctx, b); err != nil {
		t.Fatal(err)
	}
	bRoot, err := stateutil.BlockRoot(b.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, beaconState, bRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveGenesisBlockRoot(ctx, bRoot); err != nil {
		t.Fatal(err)
	}

	r := [32]byte{}
	if err := service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{
		Slot: service.slotsPerArchivedPoint,
		Root: r[:],
	}); err != nil {
		t.Fatal(err)
	}

	slot := uint64(20)
	loadedState, err := service.StateBySlot(ctx, slot)
	if err != nil {
		t.Fatal(err)
	}
	if loadedState.Slot() != slot {
		t.Error("Did not correctly save state")
	}
}

func TestStateBySlot_HotStateDB(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	if err := db.SaveBlock(ctx, b); err != nil {
		t.Fatal(err)
	}
	bRoot, err := stateutil.BlockRoot(b.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, beaconState, bRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveGenesisBlockRoot(ctx, bRoot); err != nil {
		t.Fatal(err)
	}

	slot := uint64(10)
	loadedState, err := service.StateBySlot(ctx, slot)
	if err != nil {
		t.Fatal(err)
	}
	if loadedState.Slot() != slot {
		t.Error("Did not correctly load state")
	}
}

func TestStateSummary_CanGetFromCacheOrDB(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())

	r := [32]byte{'a'}
	summary := &pb.StateSummary{Slot: 100}
	_, err := service.stateSummary(ctx, r)
	if err != errUnknownStateSummary {
		t.Fatal("Did not get wanted error")
	}

	service.stateSummaryCache.Put(r, summary)
	got, err := service.stateSummary(ctx, r)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(got, summary) {
		t.Error("Did not get wanted summary")
	}

	r = [32]byte{'b'}
	summary = &pb.StateSummary{Root: r[:], Slot: 101}
	_, err = service.stateSummary(ctx, r)
	if err != errUnknownStateSummary {
		t.Fatal("Did not get wanted error")
	}

	if err := service.beaconDB.SaveStateSummary(ctx, summary); err != nil {
		t.Fatal(err)
	}
	got, err = service.stateSummary(ctx, r)
	if err != nil {
		t.Fatal("Did not get wanted error")
	}
	if !proto.Equal(got, summary) {
		t.Error("Did not get wanted summary")
	}
}
