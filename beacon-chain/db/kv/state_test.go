package kv

import (
	"context"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

func TestState_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	s := &pb.BeaconState{Slot: 100}
	r := [32]byte{'A'}

	if err := db.SaveState(context.Background(), s, r); err != nil {
		t.Fatal(err)
	}

	savedS, err := db.State(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(s, savedS) {
		t.Error("did not retrieve saved state")
	}

	savedS, err = db.State(context.Background(), [32]byte{'B'})
	if err != nil {
		t.Fatal(err)
	}

	if savedS != nil {
		t.Error("unsaved state should've been nil")
	}
}

func TestHeadState_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	s := &pb.BeaconState{Slot: 100}
	headRoot := [32]byte{'A'}

	if err := db.SaveHeadBlockRoot(context.Background(), headRoot); err != nil {
		t.Fatal(err)
	}

	if err := db.SaveState(context.Background(), s, headRoot); err != nil {
		t.Fatal(err)
	}

	savedHeadS, err := db.HeadState(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(s, savedHeadS) {
		t.Error("did not retrieve saved state")
	}

	if err := db.SaveHeadBlockRoot(context.Background(), [32]byte{'B'}); err != nil {
		t.Fatal(err)
	}

	savedHeadS, err = db.HeadState(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if savedHeadS != nil {
		t.Error("unsaved head state should've been nil")
	}
}

func TestGenesisState_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	s := &pb.BeaconState{Slot: 1}
	headRoot := [32]byte{'B'}

	if err := db.SaveGenesisBlockRoot(context.Background(), headRoot); err != nil {
		t.Fatal(err)
	}

	if err := db.SaveState(context.Background(), s, headRoot); err != nil {
		t.Fatal(err)
	}

	savedGenesisS, err := db.GenesisState(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(s, savedGenesisS) {
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
	totalBlocks := make([]*ethpb.BeaconBlock, numBlocks)
	blockRoots := make([][32]byte, 0)
	evenBlockRoots := make([][32]byte, 0)
	for i := 0; i < len(totalBlocks); i++ {
		totalBlocks[i] = &ethpb.BeaconBlock{
			Slot:       uint64(i),
			ParentRoot: []byte("parent"),
		}
		r, err := ssz.SigningRoot(totalBlocks[i])
		if err != nil {
			t.Fatal(err)
		}
		if err := db.SaveState(context.Background(), &pb.BeaconState{Slot: uint64(i)}, r); err != nil {
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
		if s.Slot%2 == 0 {
			t.Errorf("State with slot %d should have been deleted", s.Slot)
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
	if err := db.SaveState(ctx, genesisState, genesisBlockRoot); err != nil {
		t.Fatal(err)
	}
	wantedErr := "could not delete genesis or finalized state"
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

	blk := &ethpb.BeaconBlock{
		ParentRoot: genesis[:],
		Slot:       100,
	}
	if err := db.SaveBlock(ctx, blk); err != nil {
		t.Fatal(err)
	}

	finalizedBlockRoot, err := ssz.SigningRoot(blk)
	if err != nil {
		t.Fatal(err)
	}

	finalizedState := &pb.BeaconState{Slot: 100}
	if err := db.SaveState(ctx, finalizedState, finalizedBlockRoot); err != nil {
		t.Fatal(err)
	}
	finalizedCheckpoint := &ethpb.Checkpoint{Root: finalizedBlockRoot[:]}
	if err := db.SaveFinalizedCheckpoint(ctx, finalizedCheckpoint); err != nil {
		t.Fatal(err)
	}
	wantedErr := "could not delete genesis or finalized state"
	if err := db.DeleteState(ctx, finalizedBlockRoot); err.Error() != wantedErr {
		t.Error("Did not receive wanted error")
	}
}
