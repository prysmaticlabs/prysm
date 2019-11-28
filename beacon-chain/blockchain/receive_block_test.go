package blockchain

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestReceiveBlock_ProcessCorrectly(t *testing.T) {
	hook := logTest.NewGlobal()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	ctx := context.Background()

	chainService := setupBeaconChain(t, db)

	beaconState, privKeys, _ := testutil.DeterministicGenesisState(100)
	genesis, _ := testutil.GenerateFullBlock(beaconState, privKeys, nil, beaconState.Slot+1)
	beaconState, err := state.ExecuteStateTransition(ctx, beaconState, genesis)
	if err != nil {
		t.Fatal(err)
	}
	genesisBlkRoot, err := ssz.SigningRoot(genesis)
	if err != nil {
		t.Fatal(err)
	}
	cp := &ethpb.Checkpoint{Root: genesisBlkRoot[:]}
	if err := chainService.forkChoiceStore.GenesisStore(ctx, cp, cp); err != nil {
		t.Fatal(err)
	}

	if err := chainService.beaconDB.SaveBlock(ctx, genesis); err != nil {
		t.Fatalf("Could not save block to db: %v", err)
	}

	if err := db.SaveState(ctx, beaconState, genesisBlkRoot); err != nil {
		t.Fatal(err)
	}

	slot := beaconState.Slot + 1
	block, err := testutil.GenerateFullBlock(beaconState, privKeys, nil, slot)
	if err != nil {
		t.Fatal(err)
	}

	if err := chainService.beaconDB.SaveBlock(ctx, block); err != nil {
		t.Fatal(err)
	}
	if err := chainService.ReceiveBlock(context.Background(), block); err != nil {
		t.Errorf("Block failed processing: %v", err)
	}
	testutil.AssertLogsContain(t, hook, "Finished applying state transition")
}

func TestReceiveReceiveBlockNoPubsub_CanSaveHeadInfo(t *testing.T) {
	hook := logTest.NewGlobal()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	ctx := context.Background()

	chainService := setupBeaconChain(t, db)

	headBlk := &ethpb.BeaconBlock{Slot: 100}
	if err := db.SaveBlock(ctx, headBlk); err != nil {
		t.Fatal(err)
	}
	r, err := ssz.SigningRoot(headBlk)
	if err != nil {
		t.Fatal(err)
	}
	chainService.forkChoiceStore = &store{headRoot: r[:]}

	if err := chainService.ReceiveBlockNoPubsub(ctx, &ethpb.BeaconBlock{
		Slot: 1,
		Body: &ethpb.BeaconBlockBody{}}); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(r[:], chainService.HeadRoot()) {
		t.Error("Incorrect head root saved")
	}

	if !reflect.DeepEqual(headBlk, chainService.HeadBlock()) {
		t.Error("Incorrect head block saved")
	}

	testutil.AssertLogsContain(t, hook, "Saved new head info")
}

func TestReceiveReceiveBlockNoPubsub_SameHead(t *testing.T) {
	hook := logTest.NewGlobal()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	ctx := context.Background()

	chainService := setupBeaconChain(t, db)

	headBlk := &ethpb.BeaconBlock{}
	if err := db.SaveBlock(ctx, headBlk); err != nil {
		t.Fatal(err)
	}
	newBlk := &ethpb.BeaconBlock{
		Slot: 1,
		Body: &ethpb.BeaconBlockBody{}}
	newRoot, _ := ssz.SigningRoot(newBlk)
	if err := db.SaveBlock(ctx, newBlk); err != nil {
		t.Fatal(err)
	}

	chainService.forkChoiceStore = &store{headRoot: newRoot[:]}
	chainService.canonicalRoots[0] = newRoot[:]

	if err := chainService.ReceiveBlockNoPubsub(ctx, newBlk); err != nil {
		t.Fatal(err)
	}

	testutil.AssertLogsDoNotContain(t, hook, "Saved new head info")
}

func TestReceiveBlockNoPubsubForkchoice_ProcessCorrectly(t *testing.T) {
	hook := logTest.NewGlobal()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	ctx := context.Background()

	chainService := setupBeaconChain(t, db)
	beaconState, privKeys, err := testutil.DeterministicGenesisState(100)
	if err != nil {
		t.Fatal(err)
	}

	block, err := testutil.GenerateFullBlock(beaconState, privKeys, nil, beaconState.Slot)
	if err != nil {
		t.Fatal(err)
	}

	if err := chainService.forkChoiceStore.GenesisStore(ctx, &ethpb.Checkpoint{}, &ethpb.Checkpoint{}); err != nil {
		t.Fatal(err)
	}

	if err := chainService.beaconDB.SaveBlock(ctx, block); err != nil {
		t.Fatalf("Could not save block to db: %v", err)
	}

	block, err = testutil.GenerateFullBlock(beaconState, privKeys, nil, beaconState.Slot)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, beaconState, bytesutil.ToBytes32(block.ParentRoot)); err != nil {
		t.Fatal(err)
	}

	if err := chainService.beaconDB.SaveBlock(ctx, block); err != nil {
		t.Fatal(err)
	}
	if err := chainService.ReceiveBlockNoPubsubForkchoice(context.Background(), block); err != nil {
		t.Errorf("Block failed processing: %v", err)
	}
	testutil.AssertLogsContain(t, hook, "Finished applying state transition")
	testutil.AssertLogsDoNotContain(t, hook, "Finished fork choice")
}
