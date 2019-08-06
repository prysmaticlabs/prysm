package blockchain

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/prysmaticlabs/go-ssz"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

// Ensure ChainService implements interfaces.
var _ = BlockProcessor(&ChainService{})

func init() {
	// TODO(2993): remove this after ssz is optimized for mainnet.
	c := params.BeaconConfig()
	c.HistoricalRootsLimit = 8192
	params.OverrideBeaconConfig(c)
}

func initBlockStateRoot(t *testing.T, block *ethpb.BeaconBlock, chainService *ChainService) (*ethpb.BeaconBlock, error) {
	parentRoot := bytesutil.ToBytes32(block.ParentRoot)
	beaconState, err := chainService.beaconDB.ForkChoiceState(context.Background(), parentRoot[:])
	if err != nil {
		return nil, err
	}

	computedState, err := state.ExecuteStateTransitionNoVerify(context.Background(), beaconState, block)
	if err != nil {
		t.Fatal(err)
	}

	stateRoot, err := ssz.HashTreeRoot(computedState)
	if err != nil {
		return nil, err
	}

	block.StateRoot = stateRoot[:]
	t.Logf("state root after block: %#x", stateRoot)
	return block, nil
}

func TestReceiveBlock_FaultyPOWChain(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	chainService := setupBeaconChain(t, db)
	unixTime := uint64(time.Now().Unix())
	deposits, _ := testutil.SetupInitialDeposits(t, 100)
	if err := db.InitializeState(context.Background(), unixTime, deposits, &ethpb.Eth1Data{}); err != nil {
		t.Fatalf("Could not initialize beacon state to disk: %v", err)
	}

	if err := SetSlotInState(chainService, 1); err != nil {
		t.Fatal(err)
	}

	parentBlock := &ethpb.BeaconBlock{
		Slot: 1,
	}

	parentRoot, err := ssz.SigningRoot(parentBlock)
	if err != nil {
		t.Fatalf("Unable to tree hash block %v", err)
	}

	if err := chainService.beaconDB.SaveBlock(parentBlock); err != nil {
		t.Fatalf("Unable to save block %v", err)
	}

	block := &ethpb.BeaconBlock{
		Slot:       2,
		ParentRoot: parentRoot[:],
		Body: &ethpb.BeaconBlockBody{
			Eth1Data: &ethpb.Eth1Data{
				DepositRoot: []byte("a"),
				BlockHash:   []byte("b"),
			},
		},
	}

	if err := chainService.beaconDB.SaveBlock(block); err != nil {
		t.Fatal(err)
	}
	if err := chainService.ReceiveBlock(context.Background(), block); err == nil {
		t.Errorf("Expected receive block to fail, received nil: %v", err)
	}
}

func TestReceiveBlock_ProcessCorrectly(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	chainService := setupBeaconChain(t, db)
	deposits, privKeys := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, 0, &ethpb.Eth1Data{})
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
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
	if err := chainService.forkChoiceStore.GensisStore(beaconState); err != nil {
		t.Fatal(err)
	}

	beaconState.LatestBlockHeader = &ethpb.BeaconBlockHeader{
		Slot:       genesis.Slot,
		ParentRoot: genesis.ParentRoot,
		BodyRoot:   bodyRoot[:],
		StateRoot:  genesis.StateRoot,
	}
	if err := chainService.beaconDB.SaveBlock(genesis); err != nil {
		t.Fatalf("Could not save block to db: %v", err)
	}
	parentRoot, err := ssz.SigningRoot(genesis)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.SaveForkChoiceState(ctx, beaconState, parentRoot[:]); err != nil {
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

	if err := chainService.beaconDB.SaveBlock(block); err != nil {
		t.Fatal(err)
	}
	if err := chainService.ReceiveBlock(context.Background(), block); err != nil {
		t.Errorf("Block failed processing: %v", err)
	}
	testutil.AssertLogsContain(t, hook, "Finished state transition and updated store for block")
}

func TestReceiveBlock_UsesParentBlockState(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	chainService := setupBeaconChain(t, db)
	deposits, privKeys := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, 0, &ethpb.Eth1Data{})
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}
	stateRoot, err := ssz.HashTreeRoot(beaconState)
	if err != nil {
		t.Fatal(err)
	}
	if err := chainService.forkChoiceStore.GensisStore(beaconState); err != nil {
		t.Fatal(err)
	}

	genesis := b.NewGenesisBlock(stateRoot[:])
	bodyRoot, err := ssz.HashTreeRoot(genesis.Body)
	if err != nil {
		t.Fatal(err)
	}
	parentRoot, err := ssz.SigningRoot(genesis)
	if err != nil {
		t.Fatal(err)
	}

	beaconState.LatestBlockHeader = &ethpb.BeaconBlockHeader{
		Slot:       genesis.Slot,
		ParentRoot: genesis.ParentRoot,
		BodyRoot:   bodyRoot[:],
		StateRoot:  genesis.StateRoot,
	}
	beaconState.Eth1DepositIndex = 100

	if err := chainService.beaconDB.SaveForkChoiceState(ctx, beaconState, parentRoot[:]); err != nil {
		t.Fatal(err)
	}

	// We ensure the block uses the right state parent if its ancestor is not block.Slot-1.
	beaconState, err = state.ProcessSlots(ctx, beaconState, beaconState.Slot+3)
	beaconState.Slot++
	epoch := helpers.SlotToEpoch(beaconState.Slot)
	randaoReveal, err := testutil.CreateRandaoReveal(beaconState, epoch, privKeys)
	if err != nil {
		t.Error(err)
	}
	beaconState.Slot--
	block := &ethpb.BeaconBlock{
		Slot:       beaconState.Slot + 1,
		StateRoot:  []byte{},
		ParentRoot: parentRoot[:],
		Body: &ethpb.BeaconBlockBody{
			Eth1Data: &ethpb.Eth1Data{
				DepositRoot: []byte("a"),
				BlockHash:   []byte("b"),
			},
			RandaoReveal: randaoReveal,
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

	if err := chainService.ReceiveBlock(context.Background(), block); err != nil {
		t.Errorf("Block failed processing: %v", err)
	}
	testutil.AssertLogsContain(t, hook, "Finished state transition and updated store for block")
}

func TestReceiveBlock_DeletesBadBlock(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	chainService := setupBeaconChain(t, db)
	deposits, _ := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, 0, &ethpb.Eth1Data{})
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}
	beaconState.StateRoots = make([][]byte, params.BeaconConfig().HistoricalRootsLimit)
	genesis := b.NewGenesisBlock([]byte{})
	bodyRoot, err := ssz.HashTreeRoot(genesis.Body)
	if err != nil {
		t.Fatal(err)
	}
	beaconState.LatestBlockHeader = &ethpb.BeaconBlockHeader{
		Slot:       genesis.Slot,
		ParentRoot: genesis.ParentRoot,
		BodyRoot:   bodyRoot[:],
	}

	parentHash, _ := setupGenesisBlock(t, chainService)

	if err := chainService.beaconDB.SaveHistoricalState(ctx, beaconState, parentHash); err != nil {
		t.Fatal(err)
	}

	beaconState.Slot++

	parentRoot, err := ssz.SigningRoot(beaconState.LatestBlockHeader)
	if err != nil {
		t.Fatal(err)
	}
	block := &ethpb.BeaconBlock{
		Slot:       beaconState.Slot,
		StateRoot:  []byte{},
		ParentRoot: parentRoot[:],
		Body: &ethpb.BeaconBlockBody{
			Eth1Data: &ethpb.Eth1Data{
				DepositRoot: []byte("a"),
				BlockHash:   []byte("b"),
			},
			RandaoReveal: []byte{},
			Attestations: []*ethpb.Attestation{{
				Data: &ethpb.AttestationData{
					Target: &ethpb.Checkpoint{Epoch: 5},
				},
			}},
		},
	}

	blockRoot, err := ssz.SigningRoot(block)
	if err != nil {
		t.Fatal(err)
	}

	err = chainService.ReceiveBlock(context.Background(), block)
	wanted := "failed to process block from fork choice service"
	if !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected %v, received %v", wanted, err)
	}

	savedBlock, err := db.Block(blockRoot)
	if err != nil {
		t.Fatal(err)
	}
	if savedBlock != nil {
		t.Errorf("Expected bad block to have been deleted, received: %v", savedBlock)
	}
	// We also verify the block has been blacklisted.
	if !db.IsEvilBlockHash(blockRoot) {
		t.Error("Expected block root to have been blacklisted")
	}
}

func TestReceiveBlock_CheckBlockStateRoot_GoodState(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	chainService := setupBeaconChain(t, db)
	deposits, privKeys := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, 0, &ethpb.Eth1Data{})
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}
	stateRoot, err := ssz.HashTreeRoot(beaconState)
	if err != nil {
		t.Fatal(err)
	}

	beaconState.Eth1DepositIndex = 100
	genesis := b.NewGenesisBlock(stateRoot[:])
	bodyRoot, err := ssz.HashTreeRoot(genesis.Body)
	if err != nil {
		t.Fatal(err)
	}

	if err := chainService.forkChoiceStore.GensisStore(beaconState); err != nil {
		t.Fatal(err)
	}

	beaconState.LatestBlockHeader = &ethpb.BeaconBlockHeader{
		Slot:       genesis.Slot,
		ParentRoot: genesis.ParentRoot,
		BodyRoot:   bodyRoot[:],
		StateRoot:  genesis.StateRoot,
	}
	parentRoot, err := ssz.SigningRoot(genesis)
	if err != nil {
		t.Fatal(err)
	}

	if err := chainService.beaconDB.SaveForkChoiceState(ctx, beaconState, parentRoot[:]); err != nil {
		t.Fatal(err)
	}
	beaconState.Slot++

	beaconState.Slot++

	epoch := helpers.CurrentEpoch(beaconState)
	randaoReveal, err := testutil.CreateRandaoReveal(beaconState, epoch, privKeys)
	if err != nil {
		t.Fatal(err)
	}

	goodStateBlock := &ethpb.BeaconBlock{
		Slot:       beaconState.Slot,
		ParentRoot: parentRoot[:],
		Body: &ethpb.BeaconBlockBody{
			Eth1Data:     &ethpb.Eth1Data{},
			RandaoReveal: randaoReveal,
		},
	}

	beaconState.Slot--
	goodStateBlock, err = initBlockStateRoot(t, goodStateBlock, chainService)
	if err != nil {
		t.Error(err)
	}
	goodStateBlock, err = testutil.SignBlock(beaconState, goodStateBlock, privKeys)
	if err != nil {
		t.Error(err)
	}

	if err := chainService.beaconDB.SaveBlock(goodStateBlock); err != nil {
		t.Fatal(err)
	}

	if err = chainService.ReceiveBlock(context.Background(), goodStateBlock); err != nil {
		t.Fatalf("error exists for good block %v", err)
	}
	testutil.AssertLogsContain(t, hook, "Finished state transition and updated store for block")
}

func TestReceiveBlock_CheckBlockStateRoot_BadState(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()
	chainService := setupBeaconChain(t, db)
	deposits, privKeys := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, 0, &ethpb.Eth1Data{})
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}
	beaconState.Eth1DepositIndex = 100
	genesis := b.NewGenesisBlock([]byte{})
	bodyRoot, err := ssz.HashTreeRoot(genesis.Body)
	if err != nil {
		t.Fatal(err)
	}
	beaconState.StateRoots = make([][]byte, params.BeaconConfig().HistoricalRootsLimit)
	beaconState.LatestBlockHeader = &ethpb.BeaconBlockHeader{
		Slot:       genesis.Slot,
		ParentRoot: genesis.ParentRoot,
		BodyRoot:   bodyRoot[:],
	}
	parentHash, _ := setupGenesisBlock(t, chainService)
	if err := chainService.beaconDB.SaveHistoricalState(ctx, beaconState, parentHash); err != nil {
		t.Fatal(err)
	}
	beaconState.Slot++
	beaconState.Slot++
	parentRoot, err := ssz.SigningRoot(genesis)
	if err != nil {
		t.Fatal(err)
	}
	epoch := helpers.CurrentEpoch(beaconState)
	randaoReveal, err := testutil.CreateRandaoReveal(beaconState, epoch, privKeys)
	if err != nil {
		t.Fatal(err)
	}

	invalidStateBlock := &ethpb.BeaconBlock{
		Slot:       beaconState.Slot,
		StateRoot:  []byte{'b', 'a', 'd', ' ', 'h', 'a', 's', 'h'},
		ParentRoot: parentRoot[:],
		Body: &ethpb.BeaconBlockBody{
			Eth1Data:     &ethpb.Eth1Data{},
			RandaoReveal: randaoReveal,
		},
	}
	invalidStateBlock, err = testutil.SignBlock(beaconState, invalidStateBlock, privKeys)
	if err != nil {
		t.Error(err)
	}
	beaconState.Slot--

	err = chainService.ReceiveBlock(context.Background(), invalidStateBlock)
	if err == nil {
		t.Fatal("no error for wrong block state root")
	}
	if !strings.Contains(err.Error(), "failed to process block from fork choice service") {
		t.Fatal(err)
	}
}

func TestReceiveBlock_RemovesPendingDeposits(t *testing.T) {
	// TODO: need to reimplement, this test is outdated and has becoming a mess
}
