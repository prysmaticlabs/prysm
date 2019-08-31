package blockchain

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestReceiveBlock_ProcessCorrectly(t *testing.T) {
	hook := logTest.NewGlobal()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	ctx := context.Background()

	chainService := setupBeaconChain(t, db)
	deposits, privKeys := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, 0, &ethpb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	beaconState.Eth1DepositIndex = 100
	stateRoot, err := ssz.HashTreeRoot(beaconState)
	if err != nil {
		t.Fatal(err)
	}

	genesis := b.NewGenesisBlock(stateRoot[:])
	bodyRoot, err := ssz.HashTreeRoot(genesis.Body)
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

	beaconState.LatestBlockHeader = &ethpb.BeaconBlockHeader{
		Slot:       genesis.Slot,
		ParentRoot: genesis.ParentRoot,
		BodyRoot:   bodyRoot[:],
		StateRoot:  genesis.StateRoot,
	}
	if err := chainService.beaconDB.SaveBlock(ctx, genesis); err != nil {
		t.Fatalf("Could not save block to db: %v", err)
	}
	parentRoot, err := ssz.SigningRoot(genesis)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.SaveState(ctx, beaconState, parentRoot); err != nil {
		t.Fatal(err)
	}

	slot := beaconState.Slot + 1
	epoch := helpers.SlotToEpoch(slot)
	beaconState.Slot++
	randaoReveal, err := testutil.CreateRandaoReveal(beaconState, epoch, privKeys)
	if err != nil {
		t.Fatal(err)
	}
	beaconState.Slot--

	block := &ethpb.BeaconBlock{
		Slot:       slot,
		ParentRoot: parentRoot[:],
		Body: &ethpb.BeaconBlockBody{
			Eth1Data: &ethpb.Eth1Data{
				DepositCount: uint64(len(deposits)),
				DepositRoot:  []byte("a"),
				BlockHash:    []byte("b"),
			},
			RandaoReveal: randaoReveal[:],
			Attestations: nil,
		},
	}

	stateRootCandidate, err := state.ExecuteStateTransitionNoVerify(context.Background(), beaconState, block)
	if err != nil {
		t.Fatal(err)
	}

	stateRoot, err = ssz.HashTreeRoot(stateRootCandidate)
	if err != nil {
		t.Fatal(err)
	}
	block.StateRoot = stateRoot[:]

	block, err = testutil.SignBlock(beaconState, block, privKeys)
	if err != nil {
		t.Error(err)
	}

	if err := chainService.beaconDB.SaveBlock(ctx, block); err != nil {
		t.Fatal(err)
	}
	if err := chainService.ReceiveBlock(context.Background(), block); err != nil {
		t.Errorf("Block failed processing: %v", err)
	}
	testutil.AssertLogsContain(t, hook, "Finished state transition and updated fork choice store for block")
	testutil.AssertLogsContain(t, hook, "Finished applying fork choice for block")
}

func TestReceiveReceiveBlockNoPubsub_CanSaveHeadInfo(t *testing.T) {
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
}

func TestReceiveBlockNoPubsubForkchoice_ProcessCorrectly(t *testing.T) {
	hook := logTest.NewGlobal()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	ctx := context.Background()

	chainService := setupBeaconChain(t, db)
	deposits, privKeys := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, 0, &ethpb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	beaconState.Eth1DepositIndex = 100
	stateRoot, err := ssz.HashTreeRoot(beaconState)
	if err != nil {
		t.Fatal(err)
	}

	genesis := b.NewGenesisBlock(stateRoot[:])
	bodyRoot, err := ssz.HashTreeRoot(genesis.Body)
	if err != nil {
		t.Fatal(err)
	}
	if err := chainService.forkChoiceStore.GenesisStore(ctx, &ethpb.Checkpoint{}, &ethpb.Checkpoint{}); err != nil {
		t.Fatal(err)
	}

	beaconState.LatestBlockHeader = &ethpb.BeaconBlockHeader{
		Slot:       genesis.Slot,
		ParentRoot: genesis.ParentRoot,
		BodyRoot:   bodyRoot[:],
		StateRoot:  genesis.StateRoot,
	}
	if err := chainService.beaconDB.SaveBlock(ctx, genesis); err != nil {
		t.Fatalf("Could not save block to db: %v", err)
	}
	parentRoot, err := ssz.SigningRoot(genesis)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.SaveState(ctx, beaconState, parentRoot); err != nil {
		t.Fatal(err)
	}

	slot := beaconState.Slot + 1
	epoch := helpers.SlotToEpoch(slot)
	beaconState.Slot++
	randaoReveal, err := testutil.CreateRandaoReveal(beaconState, epoch, privKeys)
	if err != nil {
		t.Fatal(err)
	}
	beaconState.Slot--

	block := &ethpb.BeaconBlock{
		Slot:       slot,
		ParentRoot: parentRoot[:],
		Body: &ethpb.BeaconBlockBody{
			Eth1Data: &ethpb.Eth1Data{
				DepositCount: uint64(len(deposits)),
				DepositRoot:  []byte("a"),
				BlockHash:    []byte("b"),
			},
			RandaoReveal: randaoReveal[:],
			Attestations: nil,
		},
	}

	stateRootCandidate, err := state.ExecuteStateTransitionNoVerify(context.Background(), beaconState, block)
	if err != nil {
		t.Fatal(err)
	}

	stateRoot, err = ssz.HashTreeRoot(stateRootCandidate)
	if err != nil {
		t.Fatal(err)
	}
	block.StateRoot = stateRoot[:]

	block, err = testutil.SignBlock(beaconState, block, privKeys)
	if err != nil {
		t.Error(err)
	}

	if err := chainService.beaconDB.SaveBlock(ctx, block); err != nil {
		t.Fatal(err)
	}
	if err := chainService.ReceiveBlockNoPubsubForkchoice(context.Background(), block); err != nil {
		t.Errorf("Block failed processing: %v", err)
	}
	testutil.AssertLogsContain(t, hook, "Finished state transition and updated fork choice store for block")
	testutil.AssertLogsDoNotContain(t, hook, "Finished fork choice")
}

func TestReceiveBlockNoPubsubForkchoice_CanUpdateValidatorDB(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	ctx := context.Background()

	chainService := setupBeaconChain(t, db)

	b := &ethpb.BeaconBlock{
		Slot: params.BeaconConfig().SlotsPerEpoch,
		Body: &ethpb.BeaconBlockBody{}}
	bRoot, err := ssz.SigningRoot(b)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, &pb.BeaconState{
		Validators: []*ethpb.Validator{
			{PublicKey: []byte{'X'}},
			{PublicKey: []byte{'Y'}},
			{PublicKey: []byte{'Z'}},
		},
	}, bRoot); err != nil {
		t.Fatal(err)
	}

	headBlk := &ethpb.BeaconBlock{Slot: 100}
	if err := db.SaveBlock(ctx, headBlk); err != nil {
		t.Fatal(err)
	}
	r, err := ssz.SigningRoot(headBlk)
	if err != nil {
		t.Fatal(err)
	}

	chainService.forkChoiceStore = &store{headRoot: r[:]}

	v.DeleteActivatedVal(1)
	v.InsertActivatedIndices(1, []uint64{0})

	if err := chainService.ReceiveBlockNoPubsub(ctx, b); err != nil {
		t.Fatal(err)
	}

	index, _, _ := db.ValidatorIndex(ctx, bytesutil.ToBytes48([]byte{'X'}))
	if index != 0 {
		t.Errorf("Wanted: %d, got: %d", 1, index)
	}
	_, e, _ := db.ValidatorIndex(ctx, bytesutil.ToBytes48([]byte{'Y'}))
	if e == true {
		t.Error("Index should not exist in DB")
	}
	_, e, _ = db.ValidatorIndex(ctx, bytesutil.ToBytes48([]byte{'Z'}))
	if e == true {
		t.Error("Index should not exist in DB")
	}
}
