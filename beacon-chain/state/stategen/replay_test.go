package stategen

import (
	"context"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestComputeStateUpToSlot_GenesisState(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())

	gBlk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	gRoot, err := stateutil.BlockRoot(gBlk.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveBlock(ctx, gBlk); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveGenesisBlockRoot(ctx, gRoot); err != nil {
		t.Fatal(err)
	}
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	if err := service.beaconDB.SaveState(ctx, beaconState, gRoot); err != nil {
		t.Fatal(err)
	}

	s, err := service.ComputeStateUpToSlot(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}

	if !proto.Equal(s.InnerStateUnsafe(), beaconState.InnerStateUnsafe()) {
		t.Error("Did not receive correct genesis state")
	}
}

func TestComputeStateUpToSlot_CanProcessUpTo(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())

	gBlk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	gRoot, err := stateutil.BlockRoot(gBlk.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveBlock(ctx, gBlk); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveGenesisBlockRoot(ctx, gRoot); err != nil {
		t.Fatal(err)
	}
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	if err := service.beaconDB.SaveState(ctx, beaconState, gRoot); err != nil {
		t.Fatal(err)
	}

	s, err := service.ComputeStateUpToSlot(ctx, params.BeaconConfig().SlotsPerEpoch+1)
	if err != nil {
		t.Fatal(err)
	}

	if s.Slot() != params.BeaconConfig().SlotsPerEpoch+1 {
		t.Log(s.Slot())
		t.Error("Did not receive correct processed state")
	}
}

func TestReplayBlocks_AllSkipSlots(t *testing.T) {
	db, _ := testDB.SetupDB(t)

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	genesisBlock := blocks.NewGenesisBlock([]byte{})
	bodyRoot, err := stateutil.BlockRoot(genesisBlock.Block)
	if err != nil {
		t.Fatal(err)
	}
	err = beaconState.SetLatestBlockHeader(&ethpb.BeaconBlockHeader{
		Slot:       genesisBlock.Block.Slot,
		ParentRoot: genesisBlock.Block.ParentRoot,
		StateRoot:  params.BeaconConfig().ZeroHash[:],
		BodyRoot:   bodyRoot[:],
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetSlashings(make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector)); err != nil {
		t.Fatal(err)
	}
	cp := beaconState.CurrentJustifiedCheckpoint()
	mockRoot := [32]byte{}
	copy(mockRoot[:], "hello-world")
	cp.Root = mockRoot[:]
	if err := beaconState.SetCurrentJustifiedCheckpoint(cp); err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{}); err != nil {
		t.Fatal(err)
	}

	service := New(db, cache.NewStateSummaryCache())
	targetSlot := params.BeaconConfig().SlotsPerEpoch - 1
	newState, err := service.ReplayBlocks(context.Background(), beaconState, []*ethpb.SignedBeaconBlock{}, targetSlot)
	if err != nil {
		t.Fatal(err)
	}

	if newState.Slot() != targetSlot {
		t.Error("Did not advance slots")
	}
}

func TestReplayBlocks_SameSlot(t *testing.T) {
	db, _ := testDB.SetupDB(t)

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	genesisBlock := blocks.NewGenesisBlock([]byte{})
	bodyRoot, err := stateutil.BlockRoot(genesisBlock.Block)
	if err != nil {
		t.Fatal(err)
	}
	err = beaconState.SetLatestBlockHeader(&ethpb.BeaconBlockHeader{
		Slot:       genesisBlock.Block.Slot,
		ParentRoot: genesisBlock.Block.ParentRoot,
		StateRoot:  params.BeaconConfig().ZeroHash[:],
		BodyRoot:   bodyRoot[:],
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetSlashings(make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector)); err != nil {
		t.Fatal(err)
	}
	cp := beaconState.CurrentJustifiedCheckpoint()
	mockRoot := [32]byte{}
	copy(mockRoot[:], "hello-world")
	cp.Root = mockRoot[:]
	if err := beaconState.SetCurrentJustifiedCheckpoint(cp); err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{}); err != nil {
		t.Fatal(err)
	}

	service := New(db, cache.NewStateSummaryCache())
	targetSlot := beaconState.Slot()
	newState, err := service.ReplayBlocks(context.Background(), beaconState, []*ethpb.SignedBeaconBlock{}, targetSlot)
	if err != nil {
		t.Fatal(err)
	}

	if newState.Slot() != targetSlot {
		t.Error("Did not advance slots")
	}
}

func TestLoadBlocks_FirstBranch(t *testing.T) {
	db, _ := testDB.SetupDB(t)
	ctx := context.Background()
	s := &State{
		beaconDB: db,
	}

	roots, savedBlocks, err := tree1(db, bytesutil.PadTo([]byte{'A'}, 32))
	if err != nil {
		t.Fatal(err)
	}

	filteredBlocks, err := s.LoadBlocks(ctx, 0, 8, roots[len(roots)-1])
	if err != nil {
		t.Fatal(err)
	}

	wanted := []*ethpb.SignedBeaconBlock{
		savedBlocks[8],
		savedBlocks[6],
		savedBlocks[4],
		savedBlocks[2],
		savedBlocks[1],
		savedBlocks[0],
	}

	for i, block := range wanted {
		if !proto.Equal(block, filteredBlocks[i]) {
			t.Error("Did not get wanted blocks")
		}
	}
}

func TestLoadBlocks_SecondBranch(t *testing.T) {
	db, _ := testDB.SetupDB(t)
	ctx := context.Background()
	s := &State{
		beaconDB: db,
	}

	roots, savedBlocks, err := tree1(db, []byte{'A'})
	if err != nil {
		t.Fatal(err)
	}

	filteredBlocks, err := s.LoadBlocks(ctx, 0, 5, roots[5])
	if err != nil {
		t.Fatal(err)
	}

	wanted := []*ethpb.SignedBeaconBlock{
		savedBlocks[5],
		savedBlocks[3],
		savedBlocks[1],
		savedBlocks[0],
	}

	for i, block := range wanted {
		if !proto.Equal(block, filteredBlocks[i]) {
			t.Error("Did not get wanted blocks")
		}
	}
}

func TestLoadBlocks_ThirdBranch(t *testing.T) {
	db, _ := testDB.SetupDB(t)
	ctx := context.Background()
	s := &State{
		beaconDB: db,
	}

	roots, savedBlocks, err := tree1(db, []byte{'A'})
	if err != nil {
		t.Fatal(err)
	}

	filteredBlocks, err := s.LoadBlocks(ctx, 0, 7, roots[7])
	if err != nil {
		t.Fatal(err)
	}

	wanted := []*ethpb.SignedBeaconBlock{
		savedBlocks[7],
		savedBlocks[6],
		savedBlocks[4],
		savedBlocks[2],
		savedBlocks[1],
		savedBlocks[0],
	}

	for i, block := range wanted {
		if !proto.Equal(block, filteredBlocks[i]) {
			t.Error("Did not get wanted blocks")
		}
	}
}

func TestLoadBlocks_SameSlots(t *testing.T) {
	db, _ := testDB.SetupDB(t)
	ctx := context.Background()
	s := &State{
		beaconDB: db,
	}

	roots, savedBlocks, err := tree2(db, []byte{'A'})
	if err != nil {
		t.Fatal(err)
	}

	filteredBlocks, err := s.LoadBlocks(ctx, 0, 3, roots[6])
	if err != nil {
		t.Fatal(err)
	}

	wanted := []*ethpb.SignedBeaconBlock{
		savedBlocks[6],
		savedBlocks[5],
		savedBlocks[1],
		savedBlocks[0],
	}

	for i, block := range wanted {
		if !proto.Equal(block, filteredBlocks[i]) {
			t.Error("Did not get wanted blocks")
		}
	}
}

func TestLoadBlocks_SameEndSlots(t *testing.T) {
	db, _ := testDB.SetupDB(t)
	ctx := context.Background()
	s := &State{
		beaconDB: db,
	}

	roots, savedBlocks, err := tree3(db, bytesutil.PadTo([]byte{'A'}, 32))
	if err != nil {
		t.Fatal(err)
	}

	filteredBlocks, err := s.LoadBlocks(ctx, 0, 2, roots[2])
	if err != nil {
		t.Fatal(err)
	}

	wanted := []*ethpb.SignedBeaconBlock{
		savedBlocks[2],
		savedBlocks[1],
		savedBlocks[0],
	}

	for i, block := range wanted {
		if !proto.Equal(block, filteredBlocks[i]) {
			t.Error("Did not get wanted blocks")
		}
	}
}

func TestLoadBlocks_SameEndSlotsWith2blocks(t *testing.T) {
	db, _ := testDB.SetupDB(t)
	ctx := context.Background()
	s := &State{
		beaconDB: db,
	}

	roots, savedBlocks, err := tree4(db, bytesutil.PadTo([]byte{'A'}, 32))
	if err != nil {
		t.Fatal(err)
	}

	filteredBlocks, err := s.LoadBlocks(ctx, 0, 2, roots[1])
	if err != nil {
		t.Fatal(err)
	}

	wanted := []*ethpb.SignedBeaconBlock{
		savedBlocks[1],
		savedBlocks[0],
	}

	for i, block := range wanted {
		if !proto.Equal(block, filteredBlocks[i]) {
			t.Error("Did not get wanted blocks")
		}
	}
}

func TestLoadBlocks_BadStart(t *testing.T) {
	db, _ := testDB.SetupDB(t)
	ctx := context.Background()
	s := &State{
		beaconDB: db,
	}

	roots, _, err := tree1(db, bytesutil.PadTo([]byte{'A'}, 32))
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.LoadBlocks(ctx, 0, 5, roots[8])
	if err.Error() != "end block roots don't match" {
		t.Error("Did not get wanted error")
	}
}

func TestLastSavedBlock_Genesis(t *testing.T) {
	db, _ := testDB.SetupDB(t)
	ctx := context.Background()
	s := &State{
		beaconDB:      db,
		finalizedInfo: &finalizedInfo{slot: 128},
	}

	gBlk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	gRoot, err := stateutil.BlockRoot(gBlk.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.beaconDB.SaveBlock(ctx, gBlk); err != nil {
		t.Fatal(err)
	}
	if err := s.beaconDB.SaveGenesisBlockRoot(ctx, gRoot); err != nil {
		t.Fatal(err)
	}

	savedRoot, savedSlot, err := s.lastSavedBlock(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	if savedSlot != 0 {
		t.Error("Did not save genesis slot")
	}
	if savedRoot != savedRoot {
		t.Error("Did not save genesis root")
	}
}

func TestLastSavedBlock_CanGet(t *testing.T) {
	db, _ := testDB.SetupDB(t)
	ctx := context.Background()
	s := &State{
		beaconDB:      db,
		finalizedInfo: &finalizedInfo{slot: 128},
	}

	b1 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: s.finalizedInfo.slot + 5}}
	if err := s.beaconDB.SaveBlock(ctx, b1); err != nil {
		t.Fatal(err)
	}
	b2 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: s.finalizedInfo.slot + 10}}
	if err := s.beaconDB.SaveBlock(ctx, b2); err != nil {
		t.Fatal(err)
	}
	b3 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: s.finalizedInfo.slot + 20}}
	if err := s.beaconDB.SaveBlock(ctx, b3); err != nil {
		t.Fatal(err)
	}

	savedRoot, savedSlot, err := s.lastSavedBlock(ctx, s.finalizedInfo.slot+100)
	if err != nil {
		t.Fatal(err)
	}
	if savedSlot != s.finalizedInfo.slot+20 {
		t.Error("Did not save correct slot")
	}
	wantedRoot, err := stateutil.BlockRoot(b3.Block)
	if err != nil {
		t.Fatal(err)
	}
	if savedRoot != wantedRoot {
		t.Error("Did not save correct root")
	}
}

func TestLastSavedBlock_NoSavedBlock(t *testing.T) {
	db, _ := testDB.SetupDB(t)
	ctx := context.Background()
	s := &State{
		beaconDB:      db,
		finalizedInfo: &finalizedInfo{slot: 128},
	}

	root, slot, err := s.lastSavedBlock(ctx, s.finalizedInfo.slot+1)
	if err != nil {
		t.Fatal(err)
	}
	if slot != 0 && root != [32]byte{} {
		t.Error("Did not get wanted block")
	}
}

func TestLastSavedState_Genesis(t *testing.T) {
	db, _ := testDB.SetupDB(t)
	ctx := context.Background()
	s := &State{
		beaconDB:      db,
		finalizedInfo: &finalizedInfo{slot: 128},
	}

	gBlk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	gRoot, err := stateutil.BlockRoot(gBlk.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.beaconDB.SaveBlock(ctx, gBlk); err != nil {
		t.Fatal(err)
	}
	if err := s.beaconDB.SaveGenesisBlockRoot(ctx, gRoot); err != nil {
		t.Fatal(err)
	}

	savedRoot, err := s.lastSavedState(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	if savedRoot != savedRoot {
		t.Error("Did not save genesis root")
	}
}

func TestLastSavedState_CanGet(t *testing.T) {
	db, _ := testDB.SetupDB(t)
	ctx := context.Background()
	s := &State{
		beaconDB:      db,
		finalizedInfo: &finalizedInfo{slot: 128},
	}

	b1 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: s.finalizedInfo.slot + 5}}
	if err := s.beaconDB.SaveBlock(ctx, b1); err != nil {
		t.Fatal(err)
	}
	b2 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: s.finalizedInfo.slot + 10}}
	if err := s.beaconDB.SaveBlock(ctx, b2); err != nil {
		t.Fatal(err)
	}
	b2Root, err := stateutil.BlockRoot(b2.Block)
	if err != nil {
		t.Fatal(err)
	}
	st := testutil.NewBeaconState()
	if err := st.SetSlot(s.finalizedInfo.slot + 10); err != nil {
		t.Fatal(err)
	}

	if err := s.beaconDB.SaveState(ctx, st, b2Root); err != nil {
		t.Fatal(err)
	}
	b3 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: s.finalizedInfo.slot + 20}}
	if err := s.beaconDB.SaveBlock(ctx, b3); err != nil {
		t.Fatal(err)
	}

	savedState, err := s.lastSavedState(ctx, s.finalizedInfo.slot+100)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(st.InnerStateUnsafe(), savedState.InnerStateUnsafe()) {
		t.Error("Did not save correct root")
	}
}

func TestLastSavedState_NoSavedBlockState(t *testing.T) {
	db, _ := testDB.SetupDB(t)
	ctx := context.Background()
	s := &State{
		beaconDB:      db,
		finalizedInfo: &finalizedInfo{slot: 128},
	}

	b1 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 127}}
	if err := s.beaconDB.SaveBlock(ctx, b1); err != nil {
		t.Fatal(err)
	}

	_, err := s.lastSavedState(ctx, s.finalizedInfo.slot+1)
	if err != errUnknownState {
		t.Error("Did not get wanted error")
	}
}

func TestArchivedState_CanGetSpecificIndex(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)
	service := New(db, cache.NewStateSummaryCache())

	r := [32]byte{'a'}
	if err := db.SaveArchivedPointRoot(ctx, r, 1); err != nil {
		t.Fatal(err)
	}
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	if err := db.SaveState(ctx, beaconState, r); err != nil {
		t.Fatal(err)
	}
	got, err := service.archivedState(ctx, params.BeaconConfig().SlotsPerArchivedPoint)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.InnerStateUnsafe(), beaconState.InnerStateUnsafe()) {
		t.Error("Did not get wanted state")
	}
	got, err = service.archivedState(ctx, params.BeaconConfig().SlotsPerArchivedPoint*2)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.InnerStateUnsafe(), beaconState.InnerStateUnsafe()) {
		t.Error("Did not get wanted state")
	}
}

func TestProcessStateUpToSlot_CanExitEarly(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	if err := beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch + 1); err != nil {
		t.Fatal(err)
	}
	s, err := service.processStateUpTo(ctx, beaconState, params.BeaconConfig().SlotsPerEpoch)
	if err != nil {
		t.Fatal(err)
	}

	if s.Slot() != params.BeaconConfig().SlotsPerEpoch+1 {
		t.Error("Did not receive correct processed state")
	}
}

func TestProcessStateUpToSlot_CanProcess(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	service := New(db, cache.NewStateSummaryCache())
	beaconState, _ := testutil.DeterministicGenesisState(t, 32)

	s, err := service.processStateUpTo(ctx, beaconState, params.BeaconConfig().SlotsPerEpoch+1)
	if err != nil {
		t.Fatal(err)
	}

	if s.Slot() != params.BeaconConfig().SlotsPerEpoch+1 {
		t.Error("Did not receive correct processed state")
	}
}

// tree1 constructs the following tree:
// B0 - B1 - - B3 -- B5
//        \- B2 -- B4 -- B6 ----- B8
//                         \- B7
func tree1(db db.Database, genesisRoot []byte) ([][32]byte, []*ethpb.SignedBeaconBlock, error) {
	b0 := &ethpb.BeaconBlock{Slot: 0, ParentRoot: genesisRoot}
	r0, err := ssz.HashTreeRoot(b0)
	if err != nil {
		return nil, nil, err
	}
	b1 := &ethpb.BeaconBlock{Slot: 1, ParentRoot: r0[:]}
	r1, err := ssz.HashTreeRoot(b1)
	if err != nil {
		return nil, nil, err
	}
	b2 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: r1[:]}
	r2, err := ssz.HashTreeRoot(b2)
	if err != nil {
		return nil, nil, err
	}
	b3 := &ethpb.BeaconBlock{Slot: 3, ParentRoot: r1[:]}
	r3, err := ssz.HashTreeRoot(b3)
	if err != nil {
		return nil, nil, err
	}
	b4 := &ethpb.BeaconBlock{Slot: 4, ParentRoot: r2[:]}
	r4, err := ssz.HashTreeRoot(b4)
	if err != nil {
		return nil, nil, err
	}
	b5 := &ethpb.BeaconBlock{Slot: 5, ParentRoot: r3[:]}
	r5, err := ssz.HashTreeRoot(b5)
	if err != nil {
		return nil, nil, err
	}
	b6 := &ethpb.BeaconBlock{Slot: 6, ParentRoot: r4[:]}
	r6, err := ssz.HashTreeRoot(b6)
	if err != nil {
		return nil, nil, err
	}
	b7 := &ethpb.BeaconBlock{Slot: 7, ParentRoot: r6[:]}
	r7, err := ssz.HashTreeRoot(b7)
	if err != nil {
		return nil, nil, err
	}
	b8 := &ethpb.BeaconBlock{Slot: 8, ParentRoot: r6[:]}
	r8, err := ssz.HashTreeRoot(b8)
	if err != nil {
		return nil, nil, err
	}
	st := testutil.NewBeaconState()

	returnedBlocks := make([]*ethpb.SignedBeaconBlock, 0)
	for _, b := range []*ethpb.BeaconBlock{b0, b1, b2, b3, b4, b5, b6, b7, b8} {
		beaconBlock := testutil.NewBeaconBlock()
		beaconBlock.Block.Slot = b.Slot
		beaconBlock.Block.ParentRoot = bytesutil.PadTo(b.ParentRoot, 32)
		if err := db.SaveBlock(context.Background(), beaconBlock); err != nil {
			return nil, nil, err
		}
		if err := db.SaveState(context.Background(), st.Copy(), bytesutil.ToBytes32(beaconBlock.Block.ParentRoot)); err != nil {
			return nil, nil, err
		}
		returnedBlocks = append(returnedBlocks, beaconBlock)
	}
	return [][32]byte{r0, r1, r2, r3, r4, r5, r6, r7, r8}, returnedBlocks, nil
}

// tree2 constructs the following tree:
// B0 - B1
//        \- B2
//        \- B2
//        \- B2
//        \- B2 -- B3
func tree2(db db.Database, genesisRoot []byte) ([][32]byte, []*ethpb.SignedBeaconBlock, error) {
	b0 := &ethpb.BeaconBlock{Slot: 0, ParentRoot: genesisRoot}
	r0, err := ssz.HashTreeRoot(b0)
	if err != nil {
		return nil, nil, err
	}
	b1 := &ethpb.BeaconBlock{Slot: 1, ParentRoot: r0[:]}
	r1, err := ssz.HashTreeRoot(b1)
	if err != nil {
		return nil, nil, err
	}
	b21 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: r1[:], StateRoot: []byte{'A'}}
	r21, err := ssz.HashTreeRoot(b21)
	if err != nil {
		return nil, nil, err
	}
	b22 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: r1[:], StateRoot: []byte{'B'}}
	r22, err := ssz.HashTreeRoot(b22)
	if err != nil {
		return nil, nil, err
	}
	b23 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: r1[:], StateRoot: []byte{'C'}}
	r23, err := ssz.HashTreeRoot(b23)
	if err != nil {
		return nil, nil, err
	}
	b24 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: r1[:], StateRoot: []byte{'D'}}
	r24, err := ssz.HashTreeRoot(b24)
	if err != nil {
		return nil, nil, err
	}
	b3 := &ethpb.BeaconBlock{Slot: 3, ParentRoot: r24[:]}
	r3, err := ssz.HashTreeRoot(b3)
	if err != nil {
		return nil, nil, err
	}
	st := testutil.NewBeaconState()

	returnedBlocks := make([]*ethpb.SignedBeaconBlock, 0)
	for _, b := range []*ethpb.BeaconBlock{b0, b1, b21, b22, b23, b24, b3} {
		beaconBlock := testutil.NewBeaconBlock()
		beaconBlock.Block.Slot = b.Slot
		beaconBlock.Block.ParentRoot = bytesutil.PadTo(b.ParentRoot, 32)
		beaconBlock.Block.StateRoot = bytesutil.PadTo(b.StateRoot, 32)
		if err := db.SaveBlock(context.Background(), beaconBlock); err != nil {
			return nil, nil, err
		}
		if err := db.SaveState(context.Background(), st.Copy(), bytesutil.ToBytes32(beaconBlock.Block.ParentRoot)); err != nil {
			return nil, nil, err
		}
		returnedBlocks = append(returnedBlocks, beaconBlock)
	}
	return [][32]byte{r0, r1, r21, r22, r23, r24, r3}, returnedBlocks, nil
}

// tree3 constructs the following tree:
// B0 - B1
//        \- B2
//        \- B2
//        \- B2
//        \- B2
func tree3(db db.Database, genesisRoot []byte) ([][32]byte, []*ethpb.SignedBeaconBlock, error) {
	b0 := &ethpb.BeaconBlock{Slot: 0, ParentRoot: genesisRoot}
	r0, err := ssz.HashTreeRoot(b0)
	if err != nil {
		return nil, nil, err
	}
	b1 := &ethpb.BeaconBlock{Slot: 1, ParentRoot: r0[:]}
	r1, err := ssz.HashTreeRoot(b1)
	if err != nil {
		return nil, nil, err
	}
	b21 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: r1[:], StateRoot: []byte{'A'}}
	r21, err := ssz.HashTreeRoot(b21)
	if err != nil {
		return nil, nil, err
	}
	b22 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: r1[:], StateRoot: []byte{'B'}}
	r22, err := ssz.HashTreeRoot(b22)
	if err != nil {
		return nil, nil, err
	}
	b23 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: r1[:], StateRoot: []byte{'C'}}
	r23, err := ssz.HashTreeRoot(b23)
	if err != nil {
		return nil, nil, err
	}
	b24 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: r1[:], StateRoot: []byte{'D'}}
	r24, err := ssz.HashTreeRoot(b24)
	if err != nil {
		return nil, nil, err
	}
	st := testutil.NewBeaconState()

	returnedBlocks := make([]*ethpb.SignedBeaconBlock, 0)
	for _, b := range []*ethpb.BeaconBlock{b0, b1, b21, b22, b23, b24} {
		beaconBlock := testutil.NewBeaconBlock()
		beaconBlock.Block.Slot = b.Slot
		beaconBlock.Block.ParentRoot = bytesutil.PadTo(b.ParentRoot, 32)
		beaconBlock.Block.StateRoot = bytesutil.PadTo(b.StateRoot, 32)
		if err := db.SaveBlock(context.Background(), beaconBlock); err != nil {
			return nil, nil, err
		}
		if err := db.SaveState(context.Background(), st.Copy(), bytesutil.ToBytes32(beaconBlock.Block.ParentRoot)); err != nil {
			return nil, nil, err
		}
		returnedBlocks = append(returnedBlocks, beaconBlock)
	}

	return [][32]byte{r0, r1, r21, r22, r23, r24}, returnedBlocks, nil
}

// tree4 constructs the following tree:
// B0
//   \- B2
//   \- B2
//   \- B2
//   \- B2
func tree4(db db.Database, genesisRoot []byte) ([][32]byte, []*ethpb.SignedBeaconBlock, error) {
	b0 := &ethpb.BeaconBlock{Slot: 0, ParentRoot: genesisRoot}
	r0, err := ssz.HashTreeRoot(b0)
	if err != nil {
		return nil, nil, err
	}
	b21 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: r0[:], StateRoot: []byte{'A'}}
	r21, err := ssz.HashTreeRoot(b21)
	if err != nil {
		return nil, nil, err
	}
	b22 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: r0[:], StateRoot: []byte{'B'}}
	r22, err := ssz.HashTreeRoot(b22)
	if err != nil {
		return nil, nil, err
	}
	b23 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: r0[:], StateRoot: []byte{'C'}}
	r23, err := ssz.HashTreeRoot(b23)
	if err != nil {
		return nil, nil, err
	}
	b24 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: r0[:], StateRoot: []byte{'D'}}
	r24, err := ssz.HashTreeRoot(b24)
	if err != nil {
		return nil, nil, err
	}
	st := testutil.NewBeaconState()

	returnedBlocks := make([]*ethpb.SignedBeaconBlock, 0)
	for _, b := range []*ethpb.BeaconBlock{b0, b21, b22, b23, b24} {
		beaconBlock := testutil.NewBeaconBlock()
		beaconBlock.Block.Slot = b.Slot
		beaconBlock.Block.ParentRoot = bytesutil.PadTo(b.ParentRoot, 32)
		beaconBlock.Block.StateRoot = bytesutil.PadTo(b.StateRoot, 32)
		if err := db.SaveBlock(context.Background(), beaconBlock); err != nil {
			return nil, nil, err
		}
		if err := db.SaveState(context.Background(), st.Copy(), bytesutil.ToBytes32(beaconBlock.Block.ParentRoot)); err != nil {
			return nil, nil, err
		}
		returnedBlocks = append(returnedBlocks, beaconBlock)
	}

	return [][32]byte{r0, r21, r22, r23, r24}, returnedBlocks, nil
}
