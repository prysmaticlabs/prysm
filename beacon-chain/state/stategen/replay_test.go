package stategen

import (
	"context"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestComputeStateUpToSlot_GenesisState(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	service := New(db)

	gBlk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	gRoot, err := ssz.HashTreeRoot(gBlk.Block)
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
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	service := New(db)

	gBlk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	gRoot, err := ssz.HashTreeRoot(gBlk.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveBlock(ctx, gBlk); err != nil {
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
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	genesisBlock := blocks.NewGenesisBlock([]byte{})
	bodyRoot, err := ssz.HashTreeRoot(genesisBlock.Block)
	if err != nil {
		t.Fatal(err)
	}
	beaconState.SetLatestBlockHeader(&ethpb.BeaconBlockHeader{
		Slot:       genesisBlock.Block.Slot,
		ParentRoot: genesisBlock.Block.ParentRoot,
		StateRoot:  params.BeaconConfig().ZeroHash[:],
		BodyRoot:   bodyRoot[:],
	})
	beaconState.SetSlashings(make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector))
	cp := beaconState.CurrentJustifiedCheckpoint()
	mockRoot := [32]byte{}
	copy(mockRoot[:], "hello-world")
	cp.Root = mockRoot[:]
	beaconState.SetCurrentJustifiedCheckpoint(cp)
	beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{})

	service := New(db)
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
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	genesisBlock := blocks.NewGenesisBlock([]byte{})
	bodyRoot, err := ssz.HashTreeRoot(genesisBlock.Block)
	if err != nil {
		t.Fatal(err)
	}
	beaconState.SetLatestBlockHeader(&ethpb.BeaconBlockHeader{
		Slot:       genesisBlock.Block.Slot,
		ParentRoot: genesisBlock.Block.ParentRoot,
		StateRoot:  params.BeaconConfig().ZeroHash[:],
		BodyRoot:   bodyRoot[:],
	})
	beaconState.SetSlashings(make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector))
	cp := beaconState.CurrentJustifiedCheckpoint()
	mockRoot := [32]byte{}
	copy(mockRoot[:], "hello-world")
	cp.Root = mockRoot[:]
	beaconState.SetCurrentJustifiedCheckpoint(cp)
	beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{})

	service := New(db)
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
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	ctx := context.Background()
	s := &State{
		beaconDB: db,
	}

	roots, savedBlocks, err := tree1(db, []byte{'A'})
	if err != nil {
		t.Fatal(err)
	}

	filteredBlocks, err := s.LoadBlocks(ctx, 0, 8, roots[len(roots)-1])
	if err != nil {
		t.Fatal(err)
	}

	wanted := []*ethpb.SignedBeaconBlock{
		{Block: savedBlocks[8]},
		{Block: savedBlocks[6]},
		{Block: savedBlocks[4]},
		{Block: savedBlocks[2]},
		{Block: savedBlocks[1]},
		{Block: savedBlocks[0]},
	}
	if !reflect.DeepEqual(filteredBlocks, wanted) {
		t.Error("Did not get wanted blocks")
	}
}

func TestLoadBlocks_SecondBranch(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
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
		{Block: savedBlocks[5]},
		{Block: savedBlocks[3]},
		{Block: savedBlocks[1]},
		{Block: savedBlocks[0]},
	}
	if !reflect.DeepEqual(filteredBlocks, wanted) {
		t.Error("Did not get wanted blocks")
	}
}

func TestLoadBlocks_ThirdBranch(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
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
		{Block: savedBlocks[7]},
		{Block: savedBlocks[6]},
		{Block: savedBlocks[4]},
		{Block: savedBlocks[2]},
		{Block: savedBlocks[1]},
		{Block: savedBlocks[0]},
	}
	if !reflect.DeepEqual(filteredBlocks, wanted) {
		t.Error("Did not get wanted blocks")
	}
}

func TestLoadBlocks_SameSlots(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
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
		{Block: savedBlocks[6]},
		{Block: savedBlocks[5]},
		{Block: savedBlocks[1]},
		{Block: savedBlocks[0]},
	}
	if !reflect.DeepEqual(filteredBlocks, wanted) {
		t.Error("Did not get wanted blocks")
	}
}

func TestLoadBlocks_SameEndSlots(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	ctx := context.Background()
	s := &State{
		beaconDB: db,
	}

	roots, savedBlocks, err := tree3(db, []byte{'A'})
	if err != nil {
		t.Fatal(err)
	}

	filteredBlocks, err := s.LoadBlocks(ctx, 0, 2, roots[2])
	if err != nil {
		t.Fatal(err)
	}

	wanted := []*ethpb.SignedBeaconBlock{
		{Block: savedBlocks[2]},
		{Block: savedBlocks[1]},
		{Block: savedBlocks[0]},
	}
	if !reflect.DeepEqual(filteredBlocks, wanted) {
		t.Error("Did not get wanted blocks")
	}
}

func TestLoadBlocks_SameEndSlotsWith2blocks(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	ctx := context.Background()
	s := &State{
		beaconDB: db,
	}

	roots, savedBlocks, err := tree4(db, []byte{'A'})
	if err != nil {
		t.Fatal(err)
	}

	filteredBlocks, err := s.LoadBlocks(ctx, 0, 2, roots[1])
	if err != nil {
		t.Fatal(err)
	}

	wanted := []*ethpb.SignedBeaconBlock{
		{Block: savedBlocks[1]},
		{Block: savedBlocks[0]},
	}
	if !reflect.DeepEqual(filteredBlocks, wanted) {
		t.Error("Did not get wanted blocks")
	}
}

func TestLoadBlocks_BadStart(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	ctx := context.Background()
	s := &State{
		beaconDB: db,
	}

	roots, _, err := tree1(db, []byte{'A'})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.LoadBlocks(ctx, 0, 5, roots[8])
	if err.Error() != "end block roots don't match" {
		t.Error("Did not get wanted error")
	}
}

func TestLastSavedBlock_Genesis(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	ctx := context.Background()
	s := &State{
		beaconDB:         db,
		lastArchivedSlot: 128,
	}

	gBlk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	gRoot, err := ssz.HashTreeRoot(gBlk.Block)
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
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	ctx := context.Background()
	s := &State{
		beaconDB:         db,
		lastArchivedSlot: 128,
	}

	b1 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: s.lastArchivedSlot + 5}}
	if err := s.beaconDB.SaveBlock(ctx, b1); err != nil {
		t.Fatal(err)
	}
	b2 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: s.lastArchivedSlot + 10}}
	if err := s.beaconDB.SaveBlock(ctx, b2); err != nil {
		t.Fatal(err)
	}
	b3 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: s.lastArchivedSlot + 20}}
	if err := s.beaconDB.SaveBlock(ctx, b3); err != nil {
		t.Fatal(err)
	}

	savedRoot, savedSlot, err := s.lastSavedBlock(ctx, s.lastArchivedSlot+100)
	if err != nil {
		t.Fatal(err)
	}
	if savedSlot != s.lastArchivedSlot+20 {
		t.Error("Did not save correct slot")
	}
	wantedRoot, _ := ssz.HashTreeRoot(b3.Block)
	if savedRoot != wantedRoot {
		t.Error("Did not save correct root")
	}
}

func TestLastSavedBlock_NoSavedBlock(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	ctx := context.Background()
	s := &State{
		beaconDB:         db,
		lastArchivedSlot: 128,
	}

	b1 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 127}}
	if err := s.beaconDB.SaveBlock(ctx, b1); err != nil {
		t.Fatal(err)
	}

	r, slot, err := s.lastSavedBlock(ctx, s.lastArchivedSlot+1)
	if err != nil {
		t.Fatal(err)
	}
	if slot != 0 || r != params.BeaconConfig().ZeroHash {
		t.Error("Did not get no saved block info")
	}
}

func TestLastSavedState_Genesis(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	ctx := context.Background()
	s := &State{
		beaconDB:         db,
		lastArchivedSlot: 128,
	}

	gBlk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	gRoot, err := ssz.HashTreeRoot(gBlk.Block)
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
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	ctx := context.Background()
	s := &State{
		beaconDB:         db,
		lastArchivedSlot: 128,
	}

	b1 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: s.lastArchivedSlot + 5}}
	if err := s.beaconDB.SaveBlock(ctx, b1); err != nil {
		t.Fatal(err)
	}
	b2 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: s.lastArchivedSlot + 10}}
	if err := s.beaconDB.SaveBlock(ctx, b2); err != nil {
		t.Fatal(err)
	}
	b2Root, _ := ssz.HashTreeRoot(b2.Block)
	st, err := stateTrie.InitializeFromProtoUnsafe(&pb.BeaconState{Slot: s.lastArchivedSlot + 10})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.beaconDB.SaveState(ctx, st, b2Root); err != nil {
		t.Fatal(err)
	}
	b3 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: s.lastArchivedSlot + 20}}
	if err := s.beaconDB.SaveBlock(ctx, b3); err != nil {
		t.Fatal(err)
	}

	savedRoot, err := s.lastSavedState(ctx, s.lastArchivedSlot+100)
	if err != nil {
		t.Fatal(err)
	}
	if savedRoot != b2Root {
		t.Error("Did not save correct root")
	}
}

func TestLastSavedState_NoSavedBlockState(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	ctx := context.Background()
	s := &State{
		beaconDB:         db,
		lastArchivedSlot: 128,
	}

	b1 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 127}}
	if err := s.beaconDB.SaveBlock(ctx, b1); err != nil {
		t.Fatal(err)
	}

	r, err := s.lastSavedState(ctx, s.lastArchivedSlot+1)
	if err != nil {
		t.Fatal(err)
	}
	if r != params.BeaconConfig().ZeroHash {
		t.Error("Did not get no saved block info")
	}
}

// tree1 constructs the following tree:
// B0 - B1 - - B3 -- B5
//        \- B2 -- B4 -- B6 ----- B8
//                         \- B7
func tree1(db db.Database, genesisRoot []byte) ([][32]byte, []*ethpb.BeaconBlock, error) {
	b0 := &ethpb.BeaconBlock{Slot: 0, ParentRoot: genesisRoot}
	r0, _ := ssz.HashTreeRoot(b0)
	b1 := &ethpb.BeaconBlock{Slot: 1, ParentRoot: r0[:]}
	r1, _ := ssz.HashTreeRoot(b1)
	b2 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: r1[:]}
	r2, _ := ssz.HashTreeRoot(b2)
	b3 := &ethpb.BeaconBlock{Slot: 3, ParentRoot: r1[:]}
	r3, _ := ssz.HashTreeRoot(b3)
	b4 := &ethpb.BeaconBlock{Slot: 4, ParentRoot: r2[:]}
	r4, _ := ssz.HashTreeRoot(b4)
	b5 := &ethpb.BeaconBlock{Slot: 5, ParentRoot: r3[:]}
	r5, _ := ssz.HashTreeRoot(b5)
	b6 := &ethpb.BeaconBlock{Slot: 6, ParentRoot: r4[:]}
	r6, _ := ssz.HashTreeRoot(b6)
	b7 := &ethpb.BeaconBlock{Slot: 7, ParentRoot: r6[:]}
	r7, _ := ssz.HashTreeRoot(b7)
	b8 := &ethpb.BeaconBlock{Slot: 8, ParentRoot: r6[:]}
	r8, _ := ssz.HashTreeRoot(b8)
	st, err := stateTrie.InitializeFromProtoUnsafe(&pb.BeaconState{})
	if err != nil {
		return nil, nil, err
	}
	for _, b := range []*ethpb.BeaconBlock{b0, b1, b2, b3, b4, b5, b6, b7, b8} {
		if err := db.SaveBlock(context.Background(), &ethpb.SignedBeaconBlock{Block: b}); err != nil {
			return nil, nil, err
		}
		if err := db.SaveState(context.Background(), st.Copy(), bytesutil.ToBytes32(b.ParentRoot)); err != nil {
			return nil, nil, err
		}
	}
	return [][32]byte{r0, r1, r2, r3, r4, r5, r6, r7, r8}, []*ethpb.BeaconBlock{b0, b1, b2, b3, b4, b5, b6, b7, b8}, nil
}

// tree2 constructs the following tree:
// B0 - B1
//        \- B2
//        \- B2
//        \- B2
//        \- B2 -- B3
func tree2(db db.Database, genesisRoot []byte) ([][32]byte, []*ethpb.BeaconBlock, error) {
	b0 := &ethpb.BeaconBlock{Slot: 0, ParentRoot: genesisRoot}
	r0, _ := ssz.HashTreeRoot(b0)
	b1 := &ethpb.BeaconBlock{Slot: 1, ParentRoot: r0[:]}
	r1, _ := ssz.HashTreeRoot(b1)
	b21 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: r1[:], StateRoot: []byte{'A'}}
	r21, _ := ssz.HashTreeRoot(b21)
	b22 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: r1[:], StateRoot: []byte{'B'}}
	r22, _ := ssz.HashTreeRoot(b22)
	b23 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: r1[:], StateRoot: []byte{'C'}}
	r23, _ := ssz.HashTreeRoot(b23)
	b24 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: r1[:], StateRoot: []byte{'D'}}
	r24, _ := ssz.HashTreeRoot(b24)
	b3 := &ethpb.BeaconBlock{Slot: 3, ParentRoot: r24[:]}
	r3, _ := ssz.HashTreeRoot(b3)
	st, err := stateTrie.InitializeFromProtoUnsafe(&pb.BeaconState{})
	if err != nil {
		return nil, nil, err
	}

	for _, b := range []*ethpb.BeaconBlock{b0, b1, b21, b22, b23, b24, b3} {
		if err := db.SaveBlock(context.Background(), &ethpb.SignedBeaconBlock{Block: b}); err != nil {
			return nil, nil, err
		}
		if err := db.SaveState(context.Background(), st.Copy(), bytesutil.ToBytes32(b.ParentRoot)); err != nil {
			return nil, nil, err
		}
	}
	return [][32]byte{r0, r1, r21, r22, r23, r24, r3}, []*ethpb.BeaconBlock{b0, b1, b21, b22, b23, b24, b3}, nil
}

// tree3 constructs the following tree:
// B0 - B1
//        \- B2
//        \- B2
//        \- B2
//        \- B2
func tree3(db db.Database, genesisRoot []byte) ([][32]byte, []*ethpb.BeaconBlock, error) {
	b0 := &ethpb.BeaconBlock{Slot: 0, ParentRoot: genesisRoot}
	r0, _ := ssz.HashTreeRoot(b0)
	b1 := &ethpb.BeaconBlock{Slot: 1, ParentRoot: r0[:]}
	r1, _ := ssz.HashTreeRoot(b1)
	b21 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: r1[:], StateRoot: []byte{'A'}}
	r21, _ := ssz.HashTreeRoot(b21)
	b22 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: r1[:], StateRoot: []byte{'B'}}
	r22, _ := ssz.HashTreeRoot(b22)
	b23 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: r1[:], StateRoot: []byte{'C'}}
	r23, _ := ssz.HashTreeRoot(b23)
	b24 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: r1[:], StateRoot: []byte{'D'}}
	r24, _ := ssz.HashTreeRoot(b24)
	st, err := stateTrie.InitializeFromProtoUnsafe(&pb.BeaconState{})
	if err != nil {
		return nil, nil, err
	}

	for _, b := range []*ethpb.BeaconBlock{b0, b1, b21, b22, b23, b24} {
		if err := db.SaveBlock(context.Background(), &ethpb.SignedBeaconBlock{Block: b}); err != nil {
			return nil, nil, err
		}
		if err := db.SaveState(context.Background(), st.Copy(), bytesutil.ToBytes32(b.ParentRoot)); err != nil {
			return nil, nil, err
		}
	}

	return [][32]byte{r0, r1, r21, r22, r23, r24}, []*ethpb.BeaconBlock{b0, b1, b21, b22, b23, b24}, nil
}

// tree4 constructs the following tree:
// B0
//   \- B2
//   \- B2
//   \- B2
//   \- B2
func tree4(db db.Database, genesisRoot []byte) ([][32]byte, []*ethpb.BeaconBlock, error) {
	b0 := &ethpb.BeaconBlock{Slot: 0, ParentRoot: genesisRoot}
	r0, _ := ssz.HashTreeRoot(b0)
	b21 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: r0[:], StateRoot: []byte{'A'}}
	r21, _ := ssz.HashTreeRoot(b21)
	b22 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: r0[:], StateRoot: []byte{'B'}}
	r22, _ := ssz.HashTreeRoot(b22)
	b23 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: r0[:], StateRoot: []byte{'C'}}
	r23, _ := ssz.HashTreeRoot(b23)
	b24 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: r0[:], StateRoot: []byte{'D'}}
	r24, _ := ssz.HashTreeRoot(b24)
	st, err := stateTrie.InitializeFromProtoUnsafe(&pb.BeaconState{})
	if err != nil {
		return nil, nil, err
	}

	for _, b := range []*ethpb.BeaconBlock{b0, b21, b22, b23, b24} {
		if err := db.SaveBlock(context.Background(), &ethpb.SignedBeaconBlock{Block: b}); err != nil {
			return nil, nil, err
		}
		if err := db.SaveState(context.Background(), st.Copy(), bytesutil.ToBytes32(b.ParentRoot)); err != nil {
			return nil, nil, err
		}
	}

	return [][32]byte{r0, r21, r22, r23, r24}, []*ethpb.BeaconBlock{b0, b21, b22, b23, b24}, nil
}
