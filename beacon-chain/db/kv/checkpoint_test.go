package kv

import (
	"context"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestStore_JustifiedCheckpoint_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	root := bytesutil.ToBytes32([]byte{'A'})
	cp := &ethpb.Checkpoint{
		Epoch: 10,
		Root:  root[:],
	}
	st := testutil.NewBeaconState()
	if err := st.SetSlot(1); err != nil {
		t.Fatal(err)
	}

	if err := db.SaveState(ctx, st, root); err != nil {
		t.Fatal(err)
	}

	if err := db.SaveJustifiedCheckpoint(ctx, cp); err != nil {
		t.Fatal(err)
	}

	retrieved, err := db.JustifiedCheckpoint(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(cp, retrieved) {
		t.Errorf("Wanted %v, received %v", cp, retrieved)
	}
}

func TestStore_FinalizedCheckpoint_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	genesis := bytesutil.ToBytes32([]byte{'G', 'E', 'N', 'E', 'S', 'I', 'S'})
	if err := db.SaveGenesisBlockRoot(ctx, genesis); err != nil {
		t.Fatal(err)
	}

	blk := testutil.NewBeaconBlock()
	blk.Block.ParentRoot = genesis[:]
	blk.Block.Slot = 40

	root, err := stateutil.BlockRoot(blk.Block)
	if err != nil {
		t.Fatal(err)
	}

	cp := &ethpb.Checkpoint{
		Epoch: 5,
		Root:  root[:],
	}

	// a valid chain is required to save finalized checkpoint.
	if err := db.SaveBlock(ctx, blk); err != nil {
		t.Fatal(err)
	}
	st := testutil.NewBeaconState()
	if err := st.SetSlot(1); err != nil {
		t.Fatal(err)
	}
	// a state is required to save checkpoint
	if err := db.SaveState(ctx, st, root); err != nil {
		t.Fatal(err)
	}

	if err := db.SaveFinalizedCheckpoint(ctx, cp); err != nil {
		t.Fatal(err)
	}

	retrieved, err := db.FinalizedCheckpoint(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(cp, retrieved) {
		t.Errorf("Wanted %v, received %v", cp, retrieved)
	}
}

func TestStore_JustifiedCheckpoint_DefaultCantBeNil(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	genesisRoot := [32]byte{'A'}
	if err := db.SaveGenesisBlockRoot(ctx, genesisRoot); err != nil {
		t.Fatal(err)
	}

	cp := &ethpb.Checkpoint{Root: genesisRoot[:]}
	retrieved, err := db.JustifiedCheckpoint(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(cp, retrieved) {
		t.Errorf("Wanted %v, received %v", cp, retrieved)
	}
}

func TestStore_FinalizedCheckpoint_DefaultCantBeNil(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	genesisRoot := [32]byte{'B'}
	if err := db.SaveGenesisBlockRoot(ctx, genesisRoot); err != nil {
		t.Fatal(err)
	}

	cp := &ethpb.Checkpoint{Root: genesisRoot[:]}
	retrieved, err := db.FinalizedCheckpoint(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(cp, retrieved) {
		t.Errorf("Wanted %v, received %v", cp, retrieved)
	}
}

func TestStore_FinalizedCheckpoint_StateMustExist(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	cp := &ethpb.Checkpoint{
		Epoch: 5,
		Root:  []byte{'B'},
	}

	if err := db.SaveFinalizedCheckpoint(ctx, cp); !strings.Contains(err.Error(), errMissingStateForCheckpoint.Error()) {
		t.Fatalf("wanted err %v, got %v", errMissingStateForCheckpoint, err)
	}
}
