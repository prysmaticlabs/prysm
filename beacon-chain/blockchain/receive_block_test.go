package blockchain

import (
	"context"
	"encoding/binary"
	"math/big"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/attestation"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	db2 "github.com/prysmaticlabs/prysm/beacon-chain/db"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

// Ensure ChainService implements interfaces.
var _ = BlockProcessor(&ChainService{})

func initBlockStateRoot(t *testing.T, block *ethpb.BeaconBlock, chainService *ChainService) (*ethpb.BeaconBlock, error) {
	parentRoot := bytesutil.ToBytes32(block.ParentRoot)
	parent, err := chainService.beaconDB.Block(context.Background(), parentRoot)
	if err != nil {
		return nil, err
	}
	beaconState, err := chainService.beaconDB.(*db2.BeaconDB).HistoricalStateFromSlot(context.Background(), parent.Slot, parentRoot)
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

func TestReceiveBlock_ProcessCorrectly(t *testing.T) {
	hook := logTest.NewGlobal()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	ctx := context.Background()

	chainService := setupBeaconChain(t, db, nil)
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
	if err := chainService.forkChoiceStore.GenesisStore(ctx, beaconState); err != nil {
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
	testutil.AssertLogsContain(t, hook, "Finished fork choice")
}

func TestReceiveBlockNoPubsubForkchoice_ProcessCorrectly(t *testing.T) {
	hook := logTest.NewGlobal()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	ctx := context.Background()

	chainService := setupBeaconChain(t, db, nil)
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
	if err := chainService.forkChoiceStore.GenesisStore(ctx, beaconState); err != nil {
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

func TestReceiveBlockDeprecated_FaultyPOWChain(t *testing.T) {
	db := internal.SetupDBDeprecated(t)
	defer internal.TeardownDBDeprecated(t, db)
	ctx := context.Background()

	chainService := setupBeaconChain(t, db, nil)
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

	if err := chainService.beaconDB.SaveBlock(ctx, parentBlock); err != nil {
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

	if err := chainService.beaconDB.SaveBlock(ctx, block); err != nil {
		t.Fatal(err)
	}
	if _, err := chainService.ReceiveBlockDeprecated(context.Background(), block); err == nil {
		t.Errorf("Expected receive block to fail, received nil: %v", err)
	}
}

func TestReceiveBlockDeprecated_ProcessCorrectly(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDBDeprecated(t)
	defer internal.TeardownDBDeprecated(t, db)
	ctx := context.Background()

	chainService := setupBeaconChain(t, db, nil)
	deposits, privKeys := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, 0, &ethpb.Eth1Data{})
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}
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
	beaconState.Eth1DepositIndex = 100
	if err := chainService.beaconDB.SaveBlock(ctx, genesis); err != nil {
		t.Fatalf("Could not save block to db: %v", err)
	}
	parentRoot, err := ssz.SigningRoot(genesis)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.SaveHistoricalState(ctx, beaconState, parentRoot); err != nil {
		t.Fatal(err)
	}

	if err := chainService.beaconDB.(*db2.BeaconDB).UpdateChainHead(ctx, genesis, beaconState); err != nil {
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

	stateRoot, err := ssz.HashTreeRoot(stateRootCandidate)
	if err != nil {
		t.Fatal(err)
	}
	block.StateRoot = stateRoot[:]

	block, err = testutil.SignBlock(beaconState, block, privKeys)
	if err != nil {
		t.Error(err)
	}

	if err := chainService.beaconDB.(*db2.BeaconDB).SaveJustifiedBlock(block); err != nil {
		t.Fatal(err)
	}
	if err := chainService.beaconDB.(*db2.BeaconDB).SaveFinalizedBlock(block); err != nil {
		t.Fatal(err)
	}
	if err := chainService.beaconDB.(*db2.BeaconDB).SaveBlockDeprecated(block); err != nil {
		t.Fatal(err)
	}
	if _, err := chainService.ReceiveBlockDeprecated(context.Background(), block); err != nil {
		t.Errorf("BlockDeprecated failed processing: %v", err)
	}
	testutil.AssertLogsContain(t, hook, "Finished processing beacon block")
}

func TestReceiveBlockDeprecated_UsesParentBlockState(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDBDeprecated(t)
	defer internal.TeardownDBDeprecated(t, db)
	ctx := context.Background()

	chainService := setupBeaconChain(t, db, nil)
	deposits, privKeys := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, 0, &ethpb.Eth1Data{})
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}
	beaconState.StateRoots = make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot)
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
	beaconState.Eth1DepositIndex = 100

	parentHash, genesisBlock := setupGenesisBlock(t, chainService)
	if err := chainService.beaconDB.(*db2.BeaconDB).UpdateChainHead(ctx, genesisBlock, beaconState); err != nil {
		t.Fatal(err)
	}
	if err := chainService.beaconDB.(*db2.BeaconDB).SaveHistoricalState(ctx, beaconState, parentHash); err != nil {
		t.Fatal(err)
	}
	parentRoot, err := ssz.SigningRoot(beaconState.LatestBlockHeader)
	if err != nil {
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

	stateRoot, err := ssz.HashTreeRoot(stateRootCandidate)
	if err != nil {
		t.Fatal(err)
	}
	block.StateRoot = stateRoot[:]

	block, err = testutil.SignBlock(beaconState, block, privKeys)
	if err != nil {
		t.Error(err)
	}

	if err := chainService.beaconDB.(*db2.BeaconDB).SaveBlockDeprecated(block); err != nil {
		t.Fatal(err)
	}
	if _, err := chainService.ReceiveBlockDeprecated(context.Background(), block); err != nil {
		t.Errorf("Block failed processing: %v", err)
	}
	testutil.AssertLogsContain(t, hook, "Finished processing beacon block")
}

func TestReceiveBlockDeprecated_DeletesBadBlock(t *testing.T) {
	db := internal.SetupDBDeprecated(t)
	defer internal.TeardownDBDeprecated(t, db)
	ctx := context.Background()

	attsService := attestation.NewAttestationService(
		context.Background(),
		&attestation.Config{BeaconDB: db})
	chainService := setupBeaconChain(t, db, attsService)
	deposits, _ := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, 0, &ethpb.Eth1Data{})
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}
	beaconState.StateRoots = make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot)
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

	parentHash, genesisBlock := setupGenesisBlock(t, chainService)
	if err := chainService.beaconDB.(*db2.BeaconDB).UpdateChainHead(ctx, genesisBlock, beaconState); err != nil {
		t.Fatal(err)
	}
	if err := chainService.beaconDB.(*db2.BeaconDB).SaveHistoricalState(ctx, beaconState, parentHash); err != nil {
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

	_, err = chainService.ReceiveBlockDeprecated(context.Background(), block)
	switch err.(type) {
	case *BlockFailedProcessingErr:
		t.Log("Block failed processing as expected")
	default:
		t.Errorf("Expected block processing to fail, received: %v", err)
	}

	savedBlock, err := db.BlockDeprecated(blockRoot)
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

func TestReceiveBlockDeprecated_CheckBlockStateRoot_GoodState(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDBDeprecated(t)
	defer internal.TeardownDBDeprecated(t, db)
	ctx := context.Background()

	attsService := attestation.NewAttestationService(
		context.Background(),
		&attestation.Config{BeaconDB: db})
	chainService := setupBeaconChain(t, db, attsService)
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
	beaconState.StateRoots = make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot)
	beaconState.LatestBlockHeader = &ethpb.BeaconBlockHeader{
		Slot:       genesis.Slot,
		ParentRoot: genesis.ParentRoot,
		BodyRoot:   bodyRoot[:],
	}
	parentHash, genesisBlock := setupGenesisBlock(t, chainService)
	if err := chainService.beaconDB.(*db2.BeaconDB).SaveHistoricalState(ctx, beaconState, parentHash); err != nil {
		t.Fatal(err)
	}
	beaconState.Slot++
	if err := chainService.beaconDB.(*db2.BeaconDB).UpdateChainHead(ctx, genesisBlock, beaconState); err != nil {
		t.Fatal(err)
	}

	beaconState.Slot++
	parentRoot, err := ssz.SigningRoot(genesisBlock)
	if err != nil {
		t.Fatal(err)
	}
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
	goodStateBlock, err = testutil.SignBlock(beaconState, goodStateBlock, privKeys)
	if err != nil {
		t.Error(err)
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

	if err := chainService.beaconDB.(*db2.BeaconDB).SaveBlockDeprecated(goodStateBlock); err != nil {
		t.Fatal(err)
	}

	_, err = chainService.ReceiveBlockDeprecated(context.Background(), goodStateBlock)
	if err != nil {
		t.Fatalf("error exists for good block %v", err)
	}
	testutil.AssertLogsContain(t, hook, "Executing state transition")
}

func TestReceiveBlockDeprecated_CheckBlockStateRoot_BadState(t *testing.T) {
	db := internal.SetupDBDeprecated(t)
	defer internal.TeardownDBDeprecated(t, db)
	ctx := context.Background()
	chainService := setupBeaconChain(t, db, nil)
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
	beaconState.StateRoots = make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot)
	beaconState.LatestBlockHeader = &ethpb.BeaconBlockHeader{
		Slot:       genesis.Slot,
		ParentRoot: genesis.ParentRoot,
		BodyRoot:   bodyRoot[:],
	}
	parentHash, genesisBlock := setupGenesisBlock(t, chainService)
	if err := chainService.beaconDB.(*db2.BeaconDB).SaveHistoricalState(ctx, beaconState, parentHash); err != nil {
		t.Fatal(err)
	}
	beaconState.Slot++
	if err := chainService.beaconDB.(*db2.BeaconDB).UpdateChainHead(ctx, genesisBlock, beaconState); err != nil {
		t.Fatal(err)
	}

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

	_, err = chainService.ReceiveBlockDeprecated(context.Background(), invalidStateBlock)
	if err == nil {
		t.Fatal("no error for wrong block state root")
	}
	if !strings.Contains(err.Error(), "block failed processing: validate state root failed") {
		t.Fatal(err)
	}
}

func TestReceiveBlockDeprecated_RemovesPendingDeposits(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDBDeprecated(t)
	defer internal.TeardownDBDeprecated(t, db)
	ctx := context.Background()

	attsService := attestation.NewAttestationService(
		context.Background(),
		&attestation.Config{BeaconDB: db})
	chainService := setupBeaconChain(t, db, attsService)
	deposits, privKeys := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, 0, &ethpb.Eth1Data{})
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}
	genesis := b.NewGenesisBlock([]byte{})
	bodyRoot, err := ssz.HashTreeRoot(genesis.Body)
	if err != nil {
		t.Fatal(err)
	}
	beaconState.StateRoots = make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot)
	beaconState.LatestBlockHeader = &ethpb.BeaconBlockHeader{
		Slot:       genesis.Slot,
		ParentRoot: genesis.ParentRoot,
		BodyRoot:   bodyRoot[:],
	}
	beaconState.Eth1Data.DepositCount = 1
	beaconState.Eth1DepositIndex = 0
	if err := chainService.beaconDB.(*db2.BeaconDB).SaveJustifiedState(beaconState); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveFinalizedState(beaconState); err != nil {
		t.Fatal(err)
	}

	stateRoot, err := ssz.HashTreeRoot(beaconState)
	if err != nil {
		t.Fatalf("Could not tree hash state: %v", err)
	}
	parentHash, genesisBlock := setupGenesisBlock(t, chainService)
	beaconState.Slot++
	if err := chainService.beaconDB.(*db2.BeaconDB).UpdateChainHead(ctx, genesisBlock, beaconState); err != nil {
		t.Fatal(err)
	}

	currentSlot := uint64(0)

	epoch := helpers.CurrentEpoch(beaconState)
	randaoReveal, err := testutil.CreateRandaoReveal(beaconState, epoch, privKeys)
	if err != nil {
		t.Fatal(err)
	}

	pendingDeposits := make([]*ethpb.Deposit, 1)
	for i := range pendingDeposits {
		var withdrawalCreds [32]byte
		copy(withdrawalCreds[:], []byte("testing"))
		depositData := &ethpb.Deposit_Data{
			Amount:                params.BeaconConfig().MaxEffectiveBalance,
			WithdrawalCredentials: withdrawalCreds[:],
		}
		depositData.PublicKey = privKeys[0].PublicKey().Marshal()
		domain := helpers.Domain(beaconState, helpers.CurrentEpoch(beaconState), params.BeaconConfig().DomainDeposit)
		root, err := ssz.SigningRoot(depositData)
		if err != nil {
			t.Fatalf("could not get signing root of deposit data %v", err)
		}
		depositData.Signature = privKeys[0].Sign(root[:], domain).Marshal()
		deposit := &ethpb.Deposit{
			Data: depositData,
		}
		pendingDeposits[i] = deposit
	}
	pendingDeposits, root := testutil.GenerateDepositProof(t, pendingDeposits)
	beaconState.Eth1Data.DepositRoot = root[:]
	if err := db.SaveHistoricalState(context.Background(), beaconState, parentHash); err != nil {
		t.Fatal(err)
	}

	parentRoot, err := ssz.SigningRoot(genesis)
	if err != nil {
		t.Fatal(err)
	}
	block := &ethpb.BeaconBlock{
		Slot:       currentSlot + 1,
		StateRoot:  stateRoot[:],
		ParentRoot: parentRoot[:],
		Body: &ethpb.BeaconBlockBody{
			Eth1Data: &ethpb.Eth1Data{
				DepositRoot: []byte("a"),
				BlockHash:   []byte("b"),
			},
			RandaoReveal: randaoReveal,
			Deposits:     pendingDeposits,
		},
	}
	block, err = testutil.SignBlock(beaconState, block, privKeys)
	if err != nil {
		t.Error(err)
	}

	beaconState.Slot--

	beaconState.Eth1DepositIndex = 0
	if err := chainService.beaconDB.(*db2.BeaconDB).SaveStateDeprecated(ctx, beaconState); err != nil {
		t.Fatal(err)
	}
	block, err = initBlockStateRoot(t, block, chainService)
	if err != nil {
		t.Error(err)
	}

	blockRoot, err := ssz.SigningRoot(block)
	if err != nil {
		log.Fatalf("could not hash block: %v", err)
	}
	block, err = testutil.SignBlock(beaconState, block, privKeys)
	if err != nil {
		t.Error(err)
	}

	if err := chainService.beaconDB.(*db2.BeaconDB).SaveJustifiedBlock(block); err != nil {
		t.Fatal(err)
	}
	if err := chainService.beaconDB.(*db2.BeaconDB).SaveFinalizedBlock(block); err != nil {
		t.Fatal(err)
	}

	for _, dep := range pendingDeposits {
		chainService.depositCache.InsertPendingDeposit(chainService.ctx, dep, big.NewInt(0), 0, [32]byte{})
	}

	if len(chainService.depositCache.PendingDeposits(chainService.ctx, nil)) != len(pendingDeposits) || len(pendingDeposits) == 0 {
		t.Fatalf("Expected %d pending deposits", len(pendingDeposits))
	}

	beaconState.Slot--
	if err := chainService.beaconDB.(*db2.BeaconDB).SaveStateDeprecated(ctx, beaconState); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveHistoricalState(context.Background(), beaconState, blockRoot); err != nil {
		t.Fatal(err)
	}
	computedState, err := chainService.ReceiveBlockDeprecated(context.Background(), block)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(beaconState.Validators); i++ {
		pubKey := bytesutil.ToBytes48(beaconState.Validators[i].PublicKey)
		attsService.InsertAttestationIntoStore(pubKey, &ethpb.Attestation{
			Data: &ethpb.AttestationData{
				BeaconBlockRoot: blockRoot[:],
			}},
		)
	}
	if err := chainService.ApplyForkChoiceRuleDeprecated(context.Background(), block, computedState); err != nil {
		t.Fatal(err)
	}

	if len(chainService.depositCache.PendingDeposits(chainService.ctx, nil)) != 0 {
		t.Fatalf("Expected 0 pending deposits, but there are %+v", chainService.depositCache.PendingDeposits(chainService.ctx, nil))
	}
	testutil.AssertLogsContain(t, hook, "Executing state transition")
}

// Scenario graph: http://bit.ly/2K1k2KZ
//
//digraph G {
//    rankdir=LR;
//    node [shape="none"];
//
//    subgraph blocks {
//        rankdir=LR;
//        node [shape="box"];
//        a->b;
//        b->c;
//        c->e;
//        c->f;
//        f->g;
//        e->h;
//    }
//
//    { rank=same; 1; a;}
//    { rank=same; 2; b;}
//    { rank=same; 3; c;}
//    { rank=same; 5; e;}
//    { rank=same; 6; f;}
//    { rank=same; 7; g;}
//    { rank=same; 8; h;}
//
//    1->2->3->4->5->6->7->8->9[arrowhead=none];
//}
func TestReceiveBlockDeprecated_OnChainSplit(t *testing.T) {
	// The scenario to test is that we think that the canonical head is block H
	// and then we receive block G. We don't have block F, so we request it. Then
	// we process F, the G. The expected behavior is that we load the historical
	// state from slot 3 where the common ancestor block C is present.

	db := internal.SetupDBDeprecated(t)
	defer internal.TeardownDBDeprecated(t, db)
	ctx := context.Background()

	chainService := setupBeaconChain(t, db, nil)
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
	beaconState.StateRoots = make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot)
	beaconState.LatestBlockHeader = &ethpb.BeaconBlockHeader{
		Slot:       genesis.Slot,
		ParentRoot: genesis.ParentRoot,
		BodyRoot:   bodyRoot[:],
	}
	stateRoot, err := ssz.HashTreeRoot(beaconState)
	if err != nil {
		t.Fatalf("Could not tree hash state: %v", err)
	}
	_, genesisBlock := setupGenesisBlock(t, chainService)
	if err := db.UpdateChainHead(ctx, genesisBlock, beaconState); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveFinalizedState(beaconState); err != nil {
		t.Fatal(err)
	}
	genesisSlot := uint64(0)

	parentRoot, err := ssz.SigningRoot(genesisBlock)
	if err != nil {
		t.Fatal(err)
	}

	// Top chain slots (see graph)
	blockSlots := []uint64{1, 2, 3, 5, 8}
	for _, slot := range blockSlots {
		epoch := helpers.SlotToEpoch(genesisSlot + slot)
		beaconSlot := beaconState.Slot
		beaconState.Slot = genesisSlot + slot
		randaoReveal, err := testutil.CreateRandaoReveal(beaconState, epoch, privKeys)
		if err != nil {
			t.Fatal(err)
		}
		beaconState.Slot = beaconSlot
		block := &ethpb.BeaconBlock{
			Slot:       genesisSlot + slot,
			StateRoot:  stateRoot[:],
			ParentRoot: parentRoot[:],
			Body: &ethpb.BeaconBlockBody{
				Eth1Data:     &ethpb.Eth1Data{},
				RandaoReveal: randaoReveal,
			},
		}
		block, err = testutil.SignBlock(beaconState, block, privKeys)
		if err != nil {
			t.Error(err)
		}
		block, err = initBlockStateRoot(t, block, chainService)
		if err != nil {
			t.Error(err)
		}
		block, err = testutil.SignBlock(beaconState, block, privKeys)
		if err != nil {
			t.Error(err)
		}
		computedState, err := chainService.ReceiveBlockDeprecated(ctx, block)
		if err != nil {
			t.Fatal(err)
		}
		stateRoot, err = ssz.HashTreeRoot(computedState)
		if err != nil {
			t.Fatal(err)
		}
		if err = db.SaveBlockDeprecated(block); err != nil {
			t.Fatal(err)
		}
		if err = db.UpdateChainHead(ctx, block, computedState); err != nil {
			t.Fatal(err)
		}
		parentRoot, err = ssz.SigningRoot(block)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Common ancestor is block at slot 3
	commonAncestor, err := db.CanonicalBlockBySlot(ctx, genesisSlot+3)
	if err != nil {
		t.Fatal(err)
	}

	parentRoot, err = ssz.SigningRoot(commonAncestor)
	if err != nil {
		t.Fatal(err)
	}

	beaconState, err = db.HistoricalStateFromSlot(ctx, commonAncestor.Slot, parentRoot)
	if err != nil {
		t.Fatal(err)
	}
	stateRoot, err = ssz.HashTreeRoot(beaconState)
	if err != nil {
		t.Fatal(err)
	}

	beaconSlot := beaconState.Slot
	beaconState.Slot = genesisSlot + 6
	epoch := helpers.CurrentEpoch(beaconState)
	randaoReveal, err := testutil.CreateRandaoReveal(beaconState, epoch, privKeys)
	if err != nil {
		t.Fatal(err)
	}
	beaconState.Slot = beaconSlot
	// Then we receive the block `f` from slot 6
	blockF := &ethpb.BeaconBlock{
		Slot:       genesisSlot + 6,
		ParentRoot: parentRoot[:],
		StateRoot:  stateRoot[:],
		Body: &ethpb.BeaconBlockBody{
			Eth1Data:     &ethpb.Eth1Data{},
			RandaoReveal: randaoReveal,
		},
	}
	rootF, _ := ssz.SigningRoot(blockF)
	if err := db.SaveHistoricalState(ctx, beaconState, rootF); err != nil {
		t.Fatal(err)
	}

	blockF, err = testutil.SignBlock(beaconState, blockF, privKeys)
	if err != nil {
		t.Error(err)
	}
	blockF, err = initBlockStateRoot(t, blockF, chainService)
	if err != nil {
		t.Error(err)
	}
	blockF, err = testutil.SignBlock(beaconState, blockF, privKeys)
	if err != nil {
		t.Error(err)
	}

	computedState, err := chainService.ReceiveBlockDeprecated(ctx, blockF)
	if err != nil {
		t.Fatal(err)
	}

	stateRoot, err = ssz.HashTreeRoot(computedState)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.SaveBlockDeprecated(blockF); err != nil {
		t.Fatal(err)
	}

	parentRoot, err = ssz.SigningRoot(blockF)
	if err != nil {
		t.Fatal(err)
	}

	beaconSlot = beaconState.Slot
	beaconState.Slot = genesisSlot + 7
	epoch = helpers.CurrentEpoch(beaconState)
	randaoReveal, err = testutil.CreateRandaoReveal(beaconState, epoch, privKeys)
	if err != nil {
		t.Fatal(err)
	}
	beaconState.Slot = beaconSlot

	// Then we apply block `g` from slot 7
	blockG := &ethpb.BeaconBlock{
		Slot:       genesisSlot + 7,
		ParentRoot: parentRoot[:],
		StateRoot:  stateRoot[:],
		Body: &ethpb.BeaconBlockBody{
			Eth1Data:     &ethpb.Eth1Data{},
			RandaoReveal: randaoReveal,
		},
	}
	blockG, err = testutil.SignBlock(beaconState, blockG, privKeys)
	if err != nil {
		t.Error(err)
	}
	blockG, err = initBlockStateRoot(t, blockG, chainService)
	if err != nil {
		t.Error(err)
	}
	blockG, err = testutil.SignBlock(beaconState, blockG, privKeys)
	if err != nil {
		t.Error(err)
	}

	computedState, err = chainService.ReceiveBlockDeprecated(ctx, blockG)
	if err != nil {
		t.Fatal(err)
	}

	if computedState.Slot != blockG.Slot {
		t.Errorf("Unexpect state slot %d, wanted %d", computedState.Slot, blockG.Slot)
	}
}

func TestIsBlockReadyForProcessing_ValidBlock(t *testing.T) {
	db := internal.SetupDBDeprecated(t)
	defer internal.TeardownDBDeprecated(t, db)
	ctx := context.Background()

	chainService := setupBeaconChain(t, db, nil)
	unixTime := uint64(time.Now().Unix())
	deposits, privKeys := testutil.SetupInitialDeposits(t, 100)
	if err := db.InitializeState(context.Background(), unixTime, deposits, &ethpb.Eth1Data{}); err != nil {
		t.Fatalf("Could not initialize beacon state to disk: %v", err)
	}
	beaconState, err := db.HeadState(ctx)
	if err != nil {
		t.Fatalf("Can't get genesis state: %v", err)
	}
	genesis := b.NewGenesisBlock([]byte{})
	bodyRoot, err := ssz.HashTreeRoot(genesis.Body)
	if err != nil {
		t.Fatal(err)
	}
	beaconState.StateRoots = make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot)
	beaconState.LatestBlockHeader = &ethpb.BeaconBlockHeader{
		Slot:       genesis.Slot,
		ParentRoot: genesis.ParentRoot,
		BodyRoot:   bodyRoot[:],
	}
	block := &ethpb.BeaconBlock{
		ParentRoot: []byte{'a'},
	}

	if err := chainService.VerifyBlockValidity(ctx, block, beaconState); err == nil {
		t.Fatal("block processing succeeded despite block having no parent saved")
	}

	beaconState.Slot = 10

	stateRoot, err := ssz.HashTreeRoot(beaconState)
	if err != nil {
		t.Fatalf("Could not tree hash state: %v", err)
	}
	if err := chainService.beaconDB.SaveBlock(ctx, genesis); err != nil {
		t.Fatalf("cannot save block: %v", err)
	}
	parentRoot, err := ssz.SigningRoot(genesis)
	if err != nil {
		t.Fatalf("unable to get root of canonical head: %v", err)
	}

	beaconState.Eth1Data = &ethpb.Eth1Data{
		DepositRoot: []byte{2},
		BlockHash:   []byte{3},
	}
	beaconState.Slot = 0

	currentSlot := uint64(1)

	epoch := helpers.CurrentEpoch(beaconState)
	randaoReveal, err := testutil.CreateRandaoReveal(beaconState, epoch, privKeys)
	if err != nil {
		t.Fatal(err)
	}

	block2 := &ethpb.BeaconBlock{
		Slot:       currentSlot,
		StateRoot:  stateRoot[:],
		ParentRoot: parentRoot[:],
		Body: &ethpb.BeaconBlockBody{
			Eth1Data: &ethpb.Eth1Data{
				DepositRoot: []byte("a"),
				BlockHash:   []byte("b"),
			},
			RandaoReveal: randaoReveal,
			Attestations: []*ethpb.Attestation{{
				AggregationBits: []byte{128, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
					0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{Root: parentRoot[:]},
					Crosslink: &ethpb.Crosslink{
						Shard: 960,
					},
				},
			}},
		},
	}

	if err := chainService.VerifyBlockValidity(ctx, block2, beaconState); err != nil {
		t.Fatalf("block processing failed despite being a valid block: %v", err)
	}
}

func TestDeleteValidatorIdx_DeleteWorks(t *testing.T) {
	db := internal.SetupDBDeprecated(t)
	defer internal.TeardownDBDeprecated(t, db)
	ctx := context.Background()
	epoch := uint64(2)
	v.InsertActivatedIndices(epoch+1, []uint64{0, 1, 2})
	v.InsertExitedVal(epoch+1, []uint64{0, 2})
	var validators []*ethpb.Validator
	for i := 0; i < 3; i++ {
		pubKeyBuf := make([]byte, params.BeaconConfig().BLSPubkeyLength)
		binary.PutUvarint(pubKeyBuf, uint64(i))
		validators = append(validators, &ethpb.Validator{
			PublicKey: pubKeyBuf,
		})
	}
	state := &pb.BeaconState{
		Validators: validators,
		Slot:       epoch * params.BeaconConfig().SlotsPerEpoch,
	}
	chainService := setupBeaconChain(t, db, nil)
	if err := chainService.saveValidatorIdx(ctx, state); err != nil {
		t.Fatalf("Could not save validator idx: %v", err)
	}
	if err := chainService.deleteValidatorIdx(ctx, state); err != nil {
		t.Fatalf("Could not delete validator idx: %v", err)
	}
	wantedIdx := uint64(1)
	idx, _, err := chainService.beaconDB.ValidatorIndex(ctx, bytesutil.ToBytes48(validators[wantedIdx].PublicKey))
	if err != nil {
		t.Fatalf("Could not get validator index: %v", err)
	}
	if wantedIdx != idx {
		t.Errorf("Wanted: %d, got: %d", wantedIdx, idx)
	}

	wantedIdx = uint64(2)
	if chainService.beaconDB.(*db2.BeaconDB).HasValidator(validators[wantedIdx].PublicKey) {
		t.Errorf("Validator index %d should have been deleted", wantedIdx)
	}
	if v.ExitedValFromEpoch(epoch) != nil {
		t.Errorf("Activated validators mapping for epoch %d still there", epoch)
	}
}

func TestSaveValidatorIdx_SaveRetrieveWorks(t *testing.T) {
	db := internal.SetupDBDeprecated(t)
	defer internal.TeardownDBDeprecated(t, db)
	ctx := context.Background()
	epoch := uint64(1)
	v.InsertActivatedIndices(epoch+1, []uint64{0, 1, 2})
	var validators []*ethpb.Validator
	for i := 0; i < 3; i++ {
		pubKeyBuf := make([]byte, params.BeaconConfig().BLSPubkeyLength)
		binary.PutUvarint(pubKeyBuf, uint64(i))
		validators = append(validators, &ethpb.Validator{
			PublicKey: pubKeyBuf,
		})
	}
	state := &pb.BeaconState{
		Validators: validators,
		Slot:       epoch * params.BeaconConfig().SlotsPerEpoch,
	}
	chainService := setupBeaconChain(t, db, nil)
	if err := chainService.saveValidatorIdx(ctx, state); err != nil {
		t.Fatalf("Could not save validator idx: %v", err)
	}

	wantedIdx := uint64(2)
	idx, _, err := chainService.beaconDB.ValidatorIndex(ctx, bytesutil.ToBytes48(validators[wantedIdx].PublicKey))
	if err != nil {
		t.Fatalf("Could not get validator index: %v", err)
	}
	if wantedIdx != idx {
		t.Errorf("Wanted: %d, got: %d", wantedIdx, idx)
	}

	if v.ActivatedValFromEpoch(epoch) != nil {
		t.Errorf("Activated validators mapping for epoch %d still there", epoch)
	}
}

func TestSaveValidatorIdx_IdxNotInState(t *testing.T) {
	db := internal.SetupDBDeprecated(t)
	defer internal.TeardownDBDeprecated(t, db)
	epoch := uint64(100)
	ctx := context.Background()

	// Tried to insert 5 active indices to DB with only 3 validators in state
	v.InsertActivatedIndices(epoch+1, []uint64{0, 1, 2, 3, 4})
	var validators []*ethpb.Validator
	for i := 0; i < 3; i++ {
		pubKeyBuf := make([]byte, params.BeaconConfig().BLSPubkeyLength)
		binary.PutUvarint(pubKeyBuf, uint64(i))
		validators = append(validators, &ethpb.Validator{
			PublicKey: pubKeyBuf,
		})
	}
	state := &pb.BeaconState{
		Validators: validators,
		Slot:       epoch * params.BeaconConfig().SlotsPerEpoch,
	}
	chainService := setupBeaconChain(t, db, nil)
	if err := chainService.saveValidatorIdx(ctx, state); err != nil {
		t.Fatalf("Could not save validator idx: %v", err)
	}

	wantedIdx := uint64(2)
	idx, _, err := chainService.beaconDB.ValidatorIndex(ctx, bytesutil.ToBytes48(validators[wantedIdx].PublicKey))
	if err != nil {
		t.Fatalf("Could not get validator index: %v", err)
	}
	if wantedIdx != idx {
		t.Errorf("Wanted: %d, got: %d", wantedIdx, idx)
	}

	if v.ActivatedValFromEpoch(epoch) != nil {
		t.Errorf("Activated validators mapping for epoch %d still there", epoch)
	}

	// Verify the skipped validators are included in the next epoch
	if !reflect.DeepEqual(v.ActivatedValFromEpoch(epoch+2), []uint64{3, 4}) {
		t.Error("Did not get wanted validator from activation queue")
	}
}
