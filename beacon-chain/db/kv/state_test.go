package kv

import (
	"context"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

func TestState_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	s := &pb.BeaconState{Slot: 100}
	r := [32]byte{'A'}

	if db.HasState(context.Background(), r) {
		t.Fatal("Wanted false")
	}

	st, err := state.InitializeFromProto(s)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.SaveState(context.Background(), st, r); err != nil {
		t.Fatal(err)
	}

	if !db.HasState(context.Background(), r) {
		t.Fatal("Wanted true")
	}

	savedS, err := db.State(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(st, savedS) {
		t.Errorf("Did not retrieve saved state: %v != %v", s, savedS)
	}

	savedS, err = db.State(context.Background(), [32]byte{'B'})
	if err != nil {
		t.Fatal(err)
	}

	if savedS != nil {
		t.Error("Unsaved state should've been nil")
	}
}

func TestHeadState_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	s := &pb.BeaconState{Slot: 100}
	headRoot := [32]byte{'A'}

	st, err := state.InitializeFromProto(s)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.SaveState(context.Background(), st, headRoot); err != nil {
		t.Fatal(err)
	}

	if err := db.SaveHeadBlockRoot(context.Background(), headRoot); err != nil {
		t.Fatal(err)
	}

	savedHeadS, err := db.HeadState(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(st, savedHeadS) {
		t.Error("did not retrieve saved state")
	}
}

func TestGenesisState_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	s := &pb.BeaconState{Slot: 1}
	headRoot := [32]byte{'B'}

	st, err := state.InitializeFromProto(s)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.SaveGenesisBlockRoot(context.Background(), headRoot); err != nil {
		t.Fatal(err)
	}

	if err := db.SaveState(context.Background(), st, headRoot); err != nil {
		t.Fatal(err)
	}

	savedGenesisS, err := db.GenesisState(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(st, savedGenesisS) {
		t.Error("did not retrieve saved state")
	}

	if err := db.SaveGenesisBlockRoot(context.Background(), [32]byte{'C'}); err != nil {
		t.Fatal(err)
	}

	savedGenesisS, err = db.HeadState(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if savedGenesisS != nil {
		t.Error("unsaved genesis state should've been nil")
	}
}

func TestStore_StatesBatchDelete(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()
	numBlocks := 100
	totalBlocks := make([]*ethpb.SignedBeaconBlock, numBlocks)
	blockRoots := make([][32]byte, 0)
	evenBlockRoots := make([][32]byte, 0)
	for i := 0; i < len(totalBlocks); i++ {
		totalBlocks[i] = &ethpb.SignedBeaconBlock{
			Block: &ethpb.BeaconBlock{
				Slot:       uint64(i),
				ParentRoot: []byte("parent"),
			},
		}
		r, err := ssz.HashTreeRoot(totalBlocks[i].Block)
		if err != nil {
			t.Fatal(err)
		}
		st, err := state.InitializeFromProto(&pb.BeaconState{Slot: uint64(i)})
		if err != nil {
			t.Fatal(err)
		}
		if err := db.SaveState(context.Background(), st, r); err != nil {
			t.Fatal(err)
		}
		blockRoots = append(blockRoots, r)
		if i%2 == 0 {
			evenBlockRoots = append(evenBlockRoots, r)
		}
	}
	if err := db.SaveBlocks(ctx, totalBlocks); err != nil {
		t.Fatal(err)
	}
	// We delete all even indexed states.
	if err := db.DeleteStates(ctx, evenBlockRoots); err != nil {
		t.Fatal(err)
	}
	// When we retrieve the data, only the odd indexed state should remain.
	for _, r := range blockRoots {
		s, err := db.State(context.Background(), r)
		if err != nil {
			t.Fatal(err)
		}
		if s == nil {
			continue
		}
		if s.Slot()%2 == 0 {
			t.Errorf("State with slot %d should have been deleted", s.Slot())
		}
	}
}

func TestStore_DeleteGenesisState(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()

	genesisBlockRoot := [32]byte{'A'}
	if err := db.SaveGenesisBlockRoot(ctx, genesisBlockRoot); err != nil {
		t.Fatal(err)
	}
	genesisState := &pb.BeaconState{Slot: 100}
	st, err := state.InitializeFromProto(genesisState)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, st, genesisBlockRoot); err != nil {
		t.Fatal(err)
	}
	wantedErr := "cannot delete genesis, finalized, or head state"
	if err := db.DeleteState(ctx, genesisBlockRoot); err.Error() != wantedErr {
		t.Error("Did not receive wanted error")
	}
}

func TestStore_DeleteFinalizedState(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()

	genesis := bytesutil.ToBytes32([]byte{'G', 'E', 'N', 'E', 'S', 'I', 'S'})
	if err := db.SaveGenesisBlockRoot(ctx, genesis); err != nil {
		t.Fatal(err)
	}

	blk := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ParentRoot: genesis[:],
			Slot:       100,
		},
	}
	if err := db.SaveBlock(ctx, blk); err != nil {
		t.Fatal(err)
	}

	finalizedBlockRoot, err := ssz.HashTreeRoot(blk.Block)
	if err != nil {
		t.Fatal(err)
	}

	finalizedState, err := state.InitializeFromProto(&pb.BeaconState{Slot: 100})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, finalizedState, finalizedBlockRoot); err != nil {
		t.Fatal(err)
	}
	finalizedCheckpoint := &ethpb.Checkpoint{Root: finalizedBlockRoot[:]}
	if err := db.SaveFinalizedCheckpoint(ctx, finalizedCheckpoint); err != nil {
		t.Fatal(err)
	}
	wantedErr := "cannot delete genesis, finalized, or head state"
	if err := db.DeleteState(ctx, finalizedBlockRoot); err.Error() != wantedErr {
		t.Error("Did not receive wanted error")
	}
}

func TestStore_DeleteHeadState(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()

	genesis := bytesutil.ToBytes32([]byte{'G', 'E', 'N', 'E', 'S', 'I', 'S'})
	if err := db.SaveGenesisBlockRoot(ctx, genesis); err != nil {
		t.Fatal(err)
	}

	blk := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ParentRoot: genesis[:],
			Slot:       100,
		},
	}
	if err := db.SaveBlock(ctx, blk); err != nil {
		t.Fatal(err)
	}

	headBlockRoot, err := ssz.HashTreeRoot(blk.Block)
	if err != nil {
		t.Fatal(err)
	}
	headState := &pb.BeaconState{Slot: 100}
	st, err := state.InitializeFromProto(headState)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, st, headBlockRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveHeadBlockRoot(ctx, headBlockRoot); err != nil {
		t.Fatal(err)
	}
	wantedErr := "cannot delete genesis, finalized, or head state"
	if err := db.DeleteState(ctx, headBlockRoot); err.Error() != wantedErr {
		t.Error("Did not receive wanted error")
	}
}

func TestStore_SaveDeleteState_CanGetHighest(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	s0 := &pb.BeaconState{Slot: 1}
	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 1}}
	r, _ := ssz.HashTreeRoot(b.Block)
	if err := db.SaveBlock(context.Background(), b); err != nil {
		t.Fatal(err)
	}
	st, err := state.InitializeFromProto(s0)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(context.Background(), st, r); err != nil {
		t.Fatal(err)
	}

	s1 := &pb.BeaconState{Slot: 999}
	b = &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 999}}
	r1, _ := ssz.HashTreeRoot(b.Block)
	if err := db.SaveBlock(context.Background(), b); err != nil {
		t.Fatal(err)
	}
	st, err = state.InitializeFromProto(s1)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(context.Background(), st, r1); err != nil {
		t.Fatal(err)
	}

	highest, err := db.HighestSlotStates(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(highest[0].InnerStateUnsafe(), s1) {
		t.Errorf("Did not retrieve saved state: %v != %v", highest, s1)
	}

	s2 := &pb.BeaconState{Slot: 1000}
	b = &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 1000}}
	r2, _ := ssz.HashTreeRoot(b.Block)
	if err := db.SaveBlock(context.Background(), b); err != nil {
		t.Fatal(err)
	}
	st, err = state.InitializeFromProto(s2)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(context.Background(), st, r2); err != nil {
		t.Fatal(err)
	}

	highest, err = db.HighestSlotStates(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(highest[0].InnerStateUnsafe(), s2) {
		t.Errorf("Did not retrieve saved state: %v != %v", highest, s2)
	}

	db.DeleteState(context.Background(), r2)
	highest, err = db.HighestSlotStates(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(highest[0].InnerStateUnsafe(), s1) {
		t.Errorf("Did not retrieve saved state: %v != %v", highest, s1)
	}

	db.DeleteState(context.Background(), r1)
	highest, err = db.HighestSlotStates(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(highest[0].InnerStateUnsafe(), s0) {
		t.Errorf("Did not retrieve saved state: %v != %v", highest, s1)
	}
}

func TestStore_SaveDeleteState_CanGetHighestBelow(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	s0 := &pb.BeaconState{Slot: 1}
	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 1}}
	r, _ := ssz.HashTreeRoot(b.Block)
	if err := db.SaveBlock(context.Background(), b); err != nil {
		t.Fatal(err)
	}
	st, err := state.InitializeFromProto(s0)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(context.Background(), st, r); err != nil {
		t.Fatal(err)
	}

	s1 := &pb.BeaconState{Slot: 100}
	b = &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 100}}
	r1, _ := ssz.HashTreeRoot(b.Block)
	if err := db.SaveBlock(context.Background(), b); err != nil {
		t.Fatal(err)
	}
	st, err = state.InitializeFromProto(s1)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(context.Background(), st, r1); err != nil {
		t.Fatal(err)
	}

	highest, err := db.HighestSlotStates(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(highest[0].InnerStateUnsafe(), s1) {
		t.Errorf("Did not retrieve saved state: %v != %v", highest, s1)
	}

	s2 := &pb.BeaconState{Slot: 1000}
	b = &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 1000}}
	r2, _ := ssz.HashTreeRoot(b.Block)
	if err := db.SaveBlock(context.Background(), b); err != nil {
		t.Fatal(err)
	}
	st, err = state.InitializeFromProto(s2)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(context.Background(), st, r2); err != nil {
		t.Fatal(err)
	}

	highest, err = db.HighestSlotStatesBelow(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(highest[0].InnerStateUnsafe(), s0) {
		t.Errorf("Did not retrieve saved state: %v != %v", highest, s0)
	}

	highest, err = db.HighestSlotStatesBelow(context.Background(), 101)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(highest[0].InnerStateUnsafe(), s1) {
		t.Errorf("Did not retrieve saved state: %v != %v", highest, s1)
	}

	highest, err = db.HighestSlotStatesBelow(context.Background(), 1001)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(highest[0].InnerStateUnsafe(), s2) {
		t.Errorf("Did not retrieve saved state: %v != %v", highest, s2)
	}
}

func TestStore_GenesisState_CanGetHighestBelow(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	s := &pb.BeaconState{}
	genesisState, err := state.InitializeFromProto(s)
	if err != nil {
		t.Fatal(err)
	}
	genesisRoot := [32]byte{'a'}
	db.SaveGenesisBlockRoot(context.Background(), genesisRoot)
	db.SaveState(context.Background(), genesisState, genesisRoot)

	s0 := &pb.BeaconState{Slot: 1}
	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 1}}
	r, _ := ssz.HashTreeRoot(b.Block)
	if err := db.SaveBlock(context.Background(), b); err != nil {
		t.Fatal(err)
	}
	st, err := state.InitializeFromProto(s0)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(context.Background(), st, r); err != nil {
		t.Fatal(err)
	}

	highest, err := db.HighestSlotStatesBelow(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(highest[0].InnerStateUnsafe(), s0) {
		t.Errorf("Did not retrieve saved state: %v != %v", highest, s0)
	}

	highest, err = db.HighestSlotStatesBelow(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(highest[0].InnerStateUnsafe(), genesisState.InnerStateUnsafe()) {
		t.Errorf("Did not retrieve saved state: %v != %v", highest, s0)
	}
	highest, err = db.HighestSlotStatesBelow(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(highest[0].InnerStateUnsafe(), genesisState.InnerStateUnsafe()) {
		t.Errorf("Did not retrieve saved state: %v != %v", highest, s0)
	}
}
