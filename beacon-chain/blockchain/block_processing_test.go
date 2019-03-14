package blockchain

import (
	"context"
	"encoding/binary"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func initBlockStateRoot(t *testing.T, block *pb.BeaconBlock, beaconStateArg *pb.BeaconState, chainService *ChainService) {
	var beaconState *pb.BeaconState
	if beaconStateArg == nil {
		beaconState, _ = chainService.beaconDB.State(context.TODO())
	} else {
		beaconState = beaconStateArg
	}

	//proto.Clone(state).(*pb.BeaconState)

	savedBeaconState, err := proto.Marshal(beaconState)
	if err != nil {
		t.Fatalf("failed to marshal beaconState: %v", err)
	}

	chainHeadRoot, err := chainService.ChainHeadRoot()
	if err != nil {
		t.Fatalf("could not retrieve chain head root: %v", err)
	}
	computedState, err := state.ExecuteStateTransition(
		chainService.ctx,
		beaconState,
		block,
		chainHeadRoot,
		false,
	)
	if err != nil {
		t.Fatalf("could not execute state transition: %v", err)
	}
	stateRoot, err := hashutil.HashProto(computedState)
	if err != nil {
		t.Fatalf("could not tree hash state: %v", err)
	}
	block.StateRootHash32 = stateRoot[:]
	t.Logf("state root after block: %#x", stateRoot)

	if err := proto.Unmarshal(savedBeaconState, beaconState); err != nil {
		t.Fatalf("failed to unmarshal saved beaconState: %v", err)
	}
}

func TestReceiveBlock_FaultyPOWChain(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	chainService := setupBeaconChain(t, true, db, true, nil)
	unixTime := uint64(time.Now().Unix())
	deposits, _ := setupInitialDeposits(t, 100)
	if err := db.InitializeState(unixTime, deposits, &pb.Eth1Data{}); err != nil {
		t.Fatalf("Could not initialize beacon state to disk: %v", err)
	}

	if err := SetSlotInState(chainService, 1); err != nil {
		t.Fatal(err)
	}

	parentBlock := &pb.BeaconBlock{
		Slot: 1,
	}

	parentRoot, err := hashutil.HashBeaconBlock(parentBlock)
	if err != nil {
		t.Fatalf("Unable to tree hash block %v", err)
	}

	if err := chainService.beaconDB.SaveBlock(parentBlock); err != nil {
		t.Fatalf("Unable to save block %v", err)
	}

	block := &pb.BeaconBlock{
		Slot:             2,
		ParentRootHash32: parentRoot[:],
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte("a"),
			BlockHash32:       []byte("b"),
		},
	}

	if err := chainService.beaconDB.SaveBlock(block); err != nil {
		t.Fatal(err)
	}
	if _, err := chainService.ReceiveBlock(context.Background(), block); err == nil {
		t.Errorf("Expected receive block to fail, received nil: %v", err)
	}
}

func TestReceiveBlock_ProcessCorrectly(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	chainService := setupBeaconChain(t, false, db, true, nil)
	deposits, privKeys := setupInitialDeposits(t, 100)
	eth1Data := &pb.Eth1Data{
		DepositRootHash32: []byte{},
		BlockHash32:       []byte{},
	}
	beaconState, err := state.GenesisBeaconState(deposits, 0, eth1Data)
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}
	stateRoot, err := hashutil.HashProto(beaconState)
	if err != nil {
		t.Fatalf("Could not tree hash state: %v", err)
	}
	parentHash, genesisBlock := setupGenesisBlock(t, chainService, beaconState)
	if err := chainService.beaconDB.UpdateChainHead(genesisBlock, beaconState); err != nil {
		t.Fatal(err)
	}

	currentSlot := params.BeaconConfig().GenesisSlot
	beaconState.Slot++
	randaoReveal := createRandaoReveal(t, beaconState, privKeys)

	block := &pb.BeaconBlock{
		Slot:             currentSlot + 1,
		StateRootHash32:  stateRoot[:],
		ParentRootHash32: parentHash[:],
		RandaoReveal:     randaoReveal,
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte("a"),
			BlockHash32:       []byte("b"),
		},
		Body: &pb.BeaconBlockBody{
			Attestations: nil,
		},
	}

	beaconState.Slot--
	initBlockStateRoot(t, block, beaconState, chainService)

	if err := chainService.beaconDB.SaveBlock(block); err != nil {
		t.Fatal(err)
	}
	if _, err := chainService.ReceiveBlock(context.Background(), block); err != nil {
		t.Errorf("Block failed processing: %v", err)
	}

	testutil.AssertLogsContain(t, hook, "Processed beacon block")
}

func TestReceiveBlock_CheckBlockStateRoot_GoodState(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	chainService := setupBeaconChain(t, false, db, false, nil)
	deposits, privKeys := setupInitialDeposits(t, 100)
	eth1Data := &pb.Eth1Data{
		DepositRootHash32: []byte{},
		BlockHash32:       []byte{},
	}
	beaconState, err := state.GenesisBeaconState(deposits, 0, eth1Data)
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}
	parentHash, genesisBlock := setupGenesisBlock(t, chainService, beaconState)
	beaconState.Slot++
	if err := chainService.beaconDB.UpdateChainHead(genesisBlock, beaconState); err != nil {
		t.Fatal(err)
	}

	beaconState.Slot++
	goodStateBlock := &pb.BeaconBlock{
		Slot:             beaconState.Slot,
		ParentRootHash32: parentHash[:],
		RandaoReveal:     createRandaoReveal(t, beaconState, privKeys),
		Body:             &pb.BeaconBlockBody{},
	}
	beaconState.Slot--
	initBlockStateRoot(t, goodStateBlock, beaconState, chainService)

	_, err = chainService.ReceiveBlock(context.Background(), goodStateBlock)
	if err != nil {
		t.Fatalf("error exists for good block %v", err)
	}
	testutil.AssertLogsContain(t, hook, "Executing state transition")
}

func TestReceiveBlock_CheckBlockStateRoot_BadState(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	chainService := setupBeaconChain(t, false, db, false, nil)
	deposits, privKeys := setupInitialDeposits(t, 100)
	eth1Data := &pb.Eth1Data{
		DepositRootHash32: []byte{},
		BlockHash32:       []byte{},
	}
	beaconState, err := state.GenesisBeaconState(deposits, 0, eth1Data)
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}
	parentHash, genesisBlock := setupGenesisBlock(t, chainService, beaconState)
	beaconState.Slot++
	if err := chainService.beaconDB.UpdateChainHead(genesisBlock, beaconState); err != nil {
		t.Fatal(err)
	}

	beaconState.Slot++
	invalidStateBlock := &pb.BeaconBlock{
		Slot:             beaconState.Slot,
		StateRootHash32:  []byte{'b', 'a', 'd', ' ', 'h', 'a', 's', 'h'},
		ParentRootHash32: parentHash[:],
		RandaoReveal:     createRandaoReveal(t, beaconState, privKeys),
		Body:             &pb.BeaconBlockBody{},
	}
	beaconState.Slot--

	_, err = chainService.ReceiveBlock(context.Background(), invalidStateBlock)
	if err == nil {
		t.Fatal("no error for wrong block state root")
	}
	if !strings.Contains(err.Error(), "beacon state root is not equal to block state root: ") {
		t.Fatal(err)
	}
}

func TestReceiveBlock_RemovesPendingDeposits(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	chainService := setupBeaconChain(t, false, db, true, nil)
	deposits, privKeys := setupInitialDeposits(t, 100)
	eth1Data := &pb.Eth1Data{
		DepositRootHash32: []byte{},
		BlockHash32:       []byte{},
	}
	beaconState, err := state.GenesisBeaconState(deposits, 0, eth1Data)
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}
	parentHash, genesisBlock := setupGenesisBlock(t, chainService, beaconState)

	beaconState.Slot++
	randaoReveal := createRandaoReveal(t, beaconState, privKeys)
	beaconState.Slot--

	if err := chainService.beaconDB.UpdateChainHead(genesisBlock, beaconState); err != nil {
		t.Fatal(err)
	}

	pendingDeposits := []*pb.Deposit{
		createPreChainStartDeposit(t, []byte{'F'}),
	}
	pendingDepositsData := make([][]byte, len(pendingDeposits))
	for i, pd := range pendingDeposits {
		pendingDepositsData[i] = pd.DepositData
	}
	depositTrie, err := trieutil.GenerateTrieFromItems(pendingDepositsData, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("Could not generate deposit trie: %v", err)
	}
	for i := range pendingDeposits {
		pendingDeposits[i].MerkleTreeIndex = 0
		proof, err := depositTrie.MerkleProof(int(pendingDeposits[i].MerkleTreeIndex))
		if err != nil {
			t.Fatalf("Could not generate proof: %v", err)
		}
		pendingDeposits[i].MerkleBranchHash32S = proof
	}
	depositRoot := depositTrie.Root()
	beaconState.LatestEth1Data.DepositRootHash32 = depositRoot[:]

	block := &pb.BeaconBlock{
		Slot:             beaconState.Slot + 1,
		ParentRootHash32: parentHash[:],
		RandaoReveal:     randaoReveal,
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte("a"),
			BlockHash32:       []byte("b"),
		},
		Body: &pb.BeaconBlockBody{
			Deposits: pendingDeposits,
		},
	}

	for _, dep := range pendingDeposits {
		db.InsertPendingDeposit(chainService.ctx, dep, big.NewInt(0))
	}

	if len(db.PendingDeposits(chainService.ctx, nil)) != len(pendingDeposits) || len(pendingDeposits) == 0 {
		t.Fatalf("Expected %d pending deposits", len(pendingDeposits))
	}

	initBlockStateRoot(t, block, beaconState, chainService)
	if err := chainService.beaconDB.SaveState(beaconState); err != nil {
		t.Fatal(err)
	}

	computedState, err := chainService.ReceiveBlock(context.Background(), block)
	if err != nil {
		t.Fatal(err)
	}
	if err := chainService.ApplyForkChoiceRule(context.Background(), block, computedState); err != nil {
		t.Fatal(err)
	}

	if len(db.PendingDeposits(chainService.ctx, nil)) != 0 {
		t.Fatalf("Expected 0 pending deposits, but there are %+v", db.PendingDeposits(chainService.ctx, nil))
	}
	testutil.AssertLogsContain(t, hook, "Executing state transition")
}

func TestIsBlockReadyForProcessing_ValidBlock(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	chainService := setupBeaconChain(t, false, db, true, nil)
	unixTime := uint64(time.Now().Unix())
	deposits, privKeys := setupInitialDeposits(t, 100)
	if err := db.InitializeState(unixTime, deposits, &pb.Eth1Data{}); err != nil {
		t.Fatalf("Could not initialize beacon state to disk: %v", err)
	}
	beaconState, err := db.State(ctx)
	if err != nil {
		t.Fatalf("Can't get genesis state: %v", err)
	}
	block := &pb.BeaconBlock{
		ParentRootHash32: []byte{'a'},
	}

	if err := chainService.isBlockReadyForProcessing(block, beaconState); err == nil {
		t.Fatal("block processing succeeded despite block having no parent saved")
	}

	beaconState.Slot = params.BeaconConfig().GenesisSlot + 10

	stateRoot, err := hashutil.HashProto(beaconState)
	if err != nil {
		t.Fatalf("Could not tree hash state: %v", err)
	}
	genesis := b.NewGenesisBlock([]byte{})
	if err := chainService.beaconDB.SaveBlock(genesis); err != nil {
		t.Fatalf("cannot save block: %v", err)
	}
	parentRoot, err := hashutil.HashBeaconBlock(genesis)
	if err != nil {
		t.Fatalf("unable to get root of canonical head: %v", err)
	}

	beaconState.LatestEth1Data = &pb.Eth1Data{
		DepositRootHash32: []byte{2},
		BlockHash32:       []byte{3},
	}
	beaconState.Slot = params.BeaconConfig().GenesisSlot

	currentSlot := params.BeaconConfig().GenesisSlot + 1
	attestationSlot := params.BeaconConfig().GenesisSlot

	randaoReveal := createRandaoReveal(t, beaconState, privKeys)
	block2 := &pb.BeaconBlock{
		Slot:             currentSlot,
		StateRootHash32:  stateRoot[:],
		ParentRootHash32: parentRoot[:],
		RandaoReveal:     randaoReveal,
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte("a"),
			BlockHash32:       []byte("b"),
		},
		Body: &pb.BeaconBlockBody{
			Attestations: []*pb.Attestation{{
				AggregationBitfield: []byte{128, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
					0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				Data: &pb.AttestationData{
					Slot:                     attestationSlot,
					JustifiedBlockRootHash32: parentRoot[:],
				},
			}},
		},
	}

	chainService.enablePOWChain = true

	if err := chainService.isBlockReadyForProcessing(block2, beaconState); err != nil {
		t.Fatalf("block processing failed despite being a valid block: %v", err)
	}
}

func TestDeleteValidatorIdx_DeleteWorks(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	epoch := uint64(2)
	v.ActivatedValidators[epoch] = []uint64{0, 1, 2}
	v.ExitedValidators[epoch] = []uint64{0, 2}
	var validators []*pb.Validator
	for i := 0; i < 3; i++ {
		pubKeyBuf := make([]byte, params.BeaconConfig().BLSPubkeyLength)
		binary.PutUvarint(pubKeyBuf, uint64(i))
		validators = append(validators, &pb.Validator{
			Pubkey: pubKeyBuf,
		})
	}
	state := &pb.BeaconState{
		ValidatorRegistry: validators,
		Slot:              epoch * params.BeaconConfig().SlotsPerEpoch,
	}
	chainService := setupBeaconChain(t, false, db, true, nil)
	if err := chainService.saveValidatorIdx(state); err != nil {
		t.Fatalf("Could not save validator idx: %v", err)
	}
	if err := chainService.deleteValidatorIdx(state); err != nil {
		t.Fatalf("Could not delete validator idx: %v", err)
	}
	wantedIdx := uint64(1)
	idx, err := chainService.beaconDB.ValidatorIndex(validators[wantedIdx].Pubkey)
	if err != nil {
		t.Fatalf("Could not get validator index: %v", err)
	}
	if wantedIdx != idx {
		t.Errorf("Wanted: %d, got: %d", wantedIdx, idx)
	}

	wantedIdx = uint64(2)
	if chainService.beaconDB.HasValidator(validators[wantedIdx].Pubkey) {
		t.Errorf("Validator index %d should have been deleted", wantedIdx)
	}

	if _, ok := v.ExitedValidators[epoch]; ok {
		t.Errorf("Activated validators mapping for epoch %d still there", epoch)
	}
}

func TestSaveValidatorIdx_SaveRetrieveWorks(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	epoch := uint64(1)
	v.ActivatedValidators[epoch] = []uint64{0, 1, 2}
	var validators []*pb.Validator
	for i := 0; i < 3; i++ {
		pubKeyBuf := make([]byte, params.BeaconConfig().BLSPubkeyLength)
		binary.PutUvarint(pubKeyBuf, uint64(i))
		validators = append(validators, &pb.Validator{
			Pubkey: pubKeyBuf,
		})
	}
	state := &pb.BeaconState{
		ValidatorRegistry: validators,
		Slot:              epoch * params.BeaconConfig().SlotsPerEpoch,
	}
	chainService := setupBeaconChain(t, false, db, true, nil)
	if err := chainService.saveValidatorIdx(state); err != nil {
		t.Fatalf("Could not save validator idx: %v", err)
	}

	wantedIdx := uint64(2)
	idx, err := chainService.beaconDB.ValidatorIndex(validators[wantedIdx].Pubkey)
	if err != nil {
		t.Fatalf("Could not get validator index: %v", err)
	}
	if wantedIdx != idx {
		t.Errorf("Wanted: %d, got: %d", wantedIdx, idx)
	}

	if _, ok := v.ActivatedValidators[epoch]; ok {
		t.Errorf("Activated validators mapping for epoch %d still there", epoch)
	}
}
