package blockchain

import (
	"context"
	"encoding/binary"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/attestation"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

// Ensure ChainService implements interfaces.
var _ = BlockProcessor(&ChainService{})

func initBlockStateRoot(t *testing.T, block *pb.BeaconBlock, chainService *ChainService) {
	parentRoot := bytesutil.ToBytes32(block.ParentRootHash32)
	parent, err := chainService.beaconDB.Block(parentRoot)
	if err != nil {
		t.Fatal(err)
	}
	beaconState, err := chainService.beaconDB.HistoricalStateFromSlot(context.Background(), parent.Slot)
	if err != nil {
		t.Fatalf("Unable to retrieve state %v", err)
	}
	saveLatestBlock := beaconState.LatestBlock

	computedState, err := chainService.ApplyBlockStateTransition(context.Background(), block, beaconState)
	if err != nil {
		t.Fatalf("could not apply block state transition: %v", err)
	}

	computedState.LatestBlock = saveLatestBlock
	stateRoot, err := hashutil.HashProto(computedState)
	if err != nil {
		t.Fatalf("could not tree hash state: %v", err)
	}
	block.StateRootHash32 = stateRoot[:]
	t.Logf("state root after block: %#x", stateRoot)
}

func TestReceiveBlock_FaultyPOWChain(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	chainService := setupBeaconChain(t, db, nil)
	unixTime := uint64(time.Now().Unix())
	deposits, _ := setupInitialDeposits(t, 100)
	if err := db.InitializeState(context.Background(), unixTime, deposits, &pb.Eth1Data{}); err != nil {
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
	ctx := context.Background()

	chainService := setupBeaconChain(t, db, nil)
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
	if err := db.SaveHistoricalState(ctx, beaconState); err != nil {
		t.Fatal(err)
	}

	genesis := b.NewGenesisBlock([]byte{})
	if err := chainService.beaconDB.SaveBlock(genesis); err != nil {
		t.Fatalf("Could not save block to db: %v", err)
	}
	parentHash, err := hashutil.HashBeaconBlock(genesis)
	if err != nil {
		t.Fatalf("Unable to get tree hash root of canonical head: %v", err)
	}
	if err := chainService.beaconDB.UpdateChainHead(ctx, genesis, beaconState); err != nil {
		t.Fatal(err)
	}

	beaconState.Slot++
	randaoReveal := createRandaoReveal(t, beaconState, privKeys)

	block := &pb.BeaconBlock{
		Slot:             beaconState.Slot,
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

	initBlockStateRoot(t, block, chainService)

	if err := chainService.beaconDB.SaveJustifiedBlock(block); err != nil {
		t.Fatal(err)
	}
	if err := chainService.beaconDB.SaveFinalizedBlock(block); err != nil {
		t.Fatal(err)
	}
	if err := chainService.beaconDB.SaveBlock(block); err != nil {
		t.Fatal(err)
	}
	if _, err := chainService.ReceiveBlock(context.Background(), block); err != nil {
		t.Errorf("Block failed processing: %v", err)
	}

	testutil.AssertLogsContain(t, hook, "Finished processing beacon block")
}

func TestReceiveBlock_UsesParentBlockState(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	chainService := setupBeaconChain(t, db, nil)
	deposits, _ := setupInitialDeposits(t, 100)
	eth1Data := &pb.Eth1Data{
		DepositRootHash32: []byte{},
		BlockHash32:       []byte{},
	}
	beaconState, err := state.GenesisBeaconState(deposits, 0, eth1Data)
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}
	if err := chainService.beaconDB.SaveHistoricalState(ctx, beaconState); err != nil {
		t.Fatal(err)
	}
	stateRoot, err := hashutil.HashProto(beaconState)
	if err != nil {
		t.Fatalf("Could not tree hash state: %v", err)
	}

	parentHash, genesisBlock := setupGenesisBlock(t, chainService)
	if err := chainService.beaconDB.UpdateChainHead(ctx, genesisBlock, beaconState); err != nil {
		t.Fatal(err)
	}

	// We ensure the block uses the right state parent if its ancestor is not block.Slot-1.
	block := &pb.BeaconBlock{
		Slot:             beaconState.Slot + 4,
		StateRootHash32:  stateRoot[:],
		ParentRootHash32: parentHash[:],
		RandaoReveal:     []byte{},
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte("a"),
			BlockHash32:       []byte("b"),
		},
		Body: &pb.BeaconBlockBody{
			Attestations: nil,
		},
	}
	initBlockStateRoot(t, block, chainService)
	if err := chainService.beaconDB.SaveBlock(block); err != nil {
		t.Fatal(err)
	}
	if _, err := chainService.ReceiveBlock(context.Background(), block); err != nil {
		t.Errorf("Block failed processing: %v", err)
	}
	testutil.AssertLogsContain(t, hook, "Finished processing beacon block")
}

func TestReceiveBlock_DeletesBadBlock(t *testing.T) {
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{
		EnableCheckBlockStateRoot: false,
	})
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	chainService := setupBeaconChain(t, db, nil)
	deposits, _ := setupInitialDeposits(t, 100)
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
	if err := chainService.beaconDB.SaveHistoricalState(ctx, beaconState); err != nil {
		t.Fatal(err)
	}

	parentHash, genesisBlock := setupGenesisBlock(t, chainService)
	if err := chainService.beaconDB.UpdateChainHead(ctx, genesisBlock, beaconState); err != nil {
		t.Fatal(err)
	}

	beaconState.Slot++

	block := &pb.BeaconBlock{
		Slot:             beaconState.Slot,
		StateRootHash32:  stateRoot[:],
		ParentRootHash32: parentHash[:],
		RandaoReveal:     []byte{},
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte("a"),
			BlockHash32:       []byte("b"),
		},
		Body: &pb.BeaconBlockBody{
			Attestations: []*pb.Attestation{
				{
					Data: &pb.AttestationData{
						JustifiedEpoch: params.BeaconConfig().GenesisSlot * 100,
					},
				},
			},
		},
	}

	blockRoot, err := hashutil.HashBeaconBlock(block)
	if err != nil {
		t.Fatal(err)
	}

	_, err = chainService.ReceiveBlock(context.Background(), block)
	switch err.(type) {
	case *BlockFailedProcessingErr:
		t.Log("Block failed processing as expected")
	default:
		t.Errorf("Unexpected block processing error: %v", err)
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
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{
		EnableCheckBlockStateRoot: true,
	})
}

func TestReceiveBlock_CheckBlockStateRoot_GoodState(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	attsService := attestation.NewAttestationService(
		context.Background(),
		&attestation.Config{BeaconDB: db})
	chainService := setupBeaconChain(t, db, attsService)
	deposits, privKeys := setupInitialDeposits(t, 100)
	eth1Data := &pb.Eth1Data{
		DepositRootHash32: []byte{},
		BlockHash32:       []byte{},
	}
	beaconState, err := state.GenesisBeaconState(deposits, 0, eth1Data)
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}
	if err := chainService.beaconDB.SaveHistoricalState(ctx, beaconState); err != nil {
		t.Fatal(err)
	}
	parentHash, genesisBlock := setupGenesisBlock(t, chainService)
	beaconState.Slot++
	if err := chainService.beaconDB.UpdateChainHead(ctx, genesisBlock, beaconState); err != nil {
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
	initBlockStateRoot(t, goodStateBlock, chainService)

	if err := chainService.beaconDB.SaveBlock(goodStateBlock); err != nil {
		t.Fatal(err)
	}

	_, err = chainService.ReceiveBlock(context.Background(), goodStateBlock)
	if err != nil {
		t.Fatalf("error exists for good block %v", err)
	}
	testutil.AssertLogsContain(t, hook, "Executing state transition")
}

func TestReceiveBlock_CheckBlockStateRoot_BadState(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	chainService := setupBeaconChain(t, db, nil)
	deposits, privKeys := setupInitialDeposits(t, 100)
	ctx := context.Background()
	eth1Data := &pb.Eth1Data{
		DepositRootHash32: []byte{},
		BlockHash32:       []byte{},
	}
	beaconState, err := state.GenesisBeaconState(deposits, 0, eth1Data)
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}
	if err := chainService.beaconDB.SaveHistoricalState(ctx, beaconState); err != nil {
		t.Fatal(err)
	}
	parentHash, genesisBlock := setupGenesisBlock(t, chainService)
	beaconState.Slot++
	if err := chainService.beaconDB.UpdateChainHead(ctx, genesisBlock, beaconState); err != nil {
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
	ctx := context.Background()

	attsService := attestation.NewAttestationService(
		context.Background(),
		&attestation.Config{BeaconDB: db})
	chainService := setupBeaconChain(t, db, attsService)
	deposits, privKeys := setupInitialDeposits(t, 100)
	eth1Data := &pb.Eth1Data{
		DepositRootHash32: []byte{},
		BlockHash32:       []byte{},
	}
	beaconState, err := state.GenesisBeaconState(deposits, 0, eth1Data)
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}
	if err := chainService.beaconDB.SaveJustifiedState(beaconState); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveFinalizedState(beaconState); err != nil {
		t.Fatal(err)
	}

	stateRoot, err := hashutil.HashProto(beaconState)
	if err != nil {
		t.Fatalf("Could not tree hash state: %v", err)
	}
	parentHash, genesisBlock := setupGenesisBlock(t, chainService)
	beaconState.Slot++
	if err := chainService.beaconDB.UpdateChainHead(ctx, genesisBlock, beaconState); err != nil {
		t.Fatal(err)
	}

	currentSlot := params.BeaconConfig().GenesisSlot
	randaoReveal := createRandaoReveal(t, beaconState, privKeys)

	pendingDeposits := []*pb.Deposit{
		createPreChainStartDeposit(t, []byte{'F'}, beaconState.DepositIndex),
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
		pendingDeposits[i].MerkleProofHash32S = proof
	}
	depositRoot := depositTrie.Root()
	beaconState.LatestEth1Data.DepositRootHash32 = depositRoot[:]
	if err := db.SaveHistoricalState(context.Background(), beaconState); err != nil {
		t.Fatal(err)
	}

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
			Deposits: pendingDeposits,
		},
	}

	beaconState.Slot--
	beaconState.DepositIndex = 0
	if err := chainService.beaconDB.SaveState(ctx, beaconState); err != nil {
		t.Fatal(err)
	}
	initBlockStateRoot(t, block, chainService)

	blockRoot, err := hashutil.HashBeaconBlock(block)
	if err != nil {
		log.Fatalf("could not hash block: %v", err)
	}

	if err := chainService.beaconDB.SaveJustifiedBlock(block); err != nil {
		t.Fatal(err)
	}
	if err := chainService.beaconDB.SaveFinalizedBlock(block); err != nil {
		t.Fatal(err)
	}

	for _, dep := range pendingDeposits {
		db.InsertPendingDeposit(chainService.ctx, dep, big.NewInt(0))
	}

	if len(db.PendingDeposits(chainService.ctx, nil)) != len(pendingDeposits) || len(pendingDeposits) == 0 {
		t.Fatalf("Expected %d pending deposits", len(pendingDeposits))
	}

	beaconState.Slot--
	if err := chainService.beaconDB.SaveState(ctx, beaconState); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveHistoricalState(context.Background(), beaconState); err != nil {
		t.Fatal(err)
	}
	computedState, err := chainService.ReceiveBlock(context.Background(), block)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(beaconState.ValidatorRegistry); i++ {
		pubKey := bytesutil.ToBytes48(beaconState.ValidatorRegistry[i].Pubkey)
		attsService.InsertAttestationIntoStore(pubKey, &pb.Attestation{
			Data: &pb.AttestationData{
				BeaconBlockRootHash32: blockRoot[:],
			}},
		)
	}
	if err := chainService.ApplyForkChoiceRule(context.Background(), block, computedState); err != nil {
		t.Fatal(err)
	}

	if len(db.PendingDeposits(chainService.ctx, nil)) != 0 {
		t.Fatalf("Expected 0 pending deposits, but there are %+v", db.PendingDeposits(chainService.ctx, nil))
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
func TestReceiveBlock_OnChainSplit(t *testing.T) {
	// The scenario to test is that we think that the canonical head is block H
	// and then we receive block G. We don't have block F, so we request it. Then
	// we process F, the G. The expected behavior is that we load the historical
	// state from slot 3 where the common ancestor block C is present.

	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	chainService := setupBeaconChain(t, db, nil)
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
	parentHash, genesisBlock := setupGenesisBlock(t, chainService)
	if err := db.UpdateChainHead(ctx, genesisBlock, beaconState); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveFinalizedState(beaconState); err != nil {
		t.Fatal(err)
	}
	genesisSlot := params.BeaconConfig().GenesisSlot

	// Top chain slots (see graph)
	blockSlots := []uint64{1, 2, 3, 5, 8}
	for _, slot := range blockSlots {
		block := &pb.BeaconBlock{
			Slot:             genesisSlot + slot,
			StateRootHash32:  stateRoot[:],
			ParentRootHash32: parentHash[:],
			RandaoReveal:     createRandaoReveal(t, beaconState, privKeys),
			Body:             &pb.BeaconBlockBody{},
		}
		initBlockStateRoot(t, block, chainService)
		computedState, err := chainService.ReceiveBlock(ctx, block)
		if err != nil {
			t.Fatal(err)
		}
		stateRoot, err = hashutil.HashProto(computedState)
		if err != nil {
			t.Fatal(err)
		}
		if err = db.SaveBlock(block); err != nil {
			t.Fatal(err)
		}
		if err = db.UpdateChainHead(ctx, block, computedState); err != nil {
			t.Fatal(err)
		}
		parentHash, err = hashutil.HashBeaconBlock(block)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Common ancestor is block at slot 3
	commonAncestor, err := db.BlockBySlot(ctx, genesisSlot+3)
	if err != nil {
		t.Fatal(err)
	}

	parentHash, err = hashutil.HashBeaconBlock(commonAncestor)
	if err != nil {
		t.Fatal(err)
	}

	beaconState, err = db.HistoricalStateFromSlot(ctx, commonAncestor.Slot)
	if err != nil {
		t.Fatal(err)
	}
	stateRoot, err = hashutil.HashProto(beaconState)
	if err != nil {
		t.Fatal(err)
	}
	// Then we receive the block `f` from slot 6
	blockF := &pb.BeaconBlock{
		Slot:             genesisSlot + 6,
		ParentRootHash32: parentHash[:],
		StateRootHash32:  stateRoot[:],
		RandaoReveal:     createRandaoReveal(t, beaconState, privKeys),
		Body:             &pb.BeaconBlockBody{},
	}
	initBlockStateRoot(t, blockF, chainService)

	computedState, err := chainService.ReceiveBlock(ctx, blockF)
	if err != nil {
		t.Fatal(err)
	}

	stateRoot, err = hashutil.HashProto(computedState)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.SaveBlock(blockF); err != nil {
		t.Fatal(err)
	}

	parentHash, err = hashutil.HashBeaconBlock(blockF)
	if err != nil {
		t.Fatal(err)
	}

	// Then we apply block `g` from slot 7
	blockG := &pb.BeaconBlock{
		Slot:             genesisSlot + 7,
		ParentRootHash32: parentHash[:],
		StateRootHash32:  stateRoot[:],
		RandaoReveal:     createRandaoReveal(t, computedState, privKeys),
		Body:             &pb.BeaconBlockBody{},
	}
	initBlockStateRoot(t, blockG, chainService)

	computedState, err = chainService.ReceiveBlock(ctx, blockG)
	if err != nil {
		t.Fatal(err)
	}

	if computedState.Slot != blockG.Slot {
		t.Errorf("Unexpect state slot %d, wanted %d", computedState.Slot, blockG.Slot)
	}
}

func TestIsBlockReadyForProcessing_ValidBlock(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	chainService := setupBeaconChain(t, db, nil)
	unixTime := uint64(time.Now().Unix())
	deposits, privKeys := setupInitialDeposits(t, 100)
	if err := db.InitializeState(context.Background(), unixTime, deposits, &pb.Eth1Data{}); err != nil {
		t.Fatalf("Could not initialize beacon state to disk: %v", err)
	}
	beaconState, err := db.HeadState(ctx)
	if err != nil {
		t.Fatalf("Can't get genesis state: %v", err)
	}
	block := &pb.BeaconBlock{
		ParentRootHash32: []byte{'a'},
	}

	if err := chainService.VerifyBlockValidity(ctx, block, beaconState); err == nil {
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

	if err := chainService.VerifyBlockValidity(ctx, block2, beaconState); err != nil {
		t.Fatalf("block processing failed despite being a valid block: %v", err)
	}
}

func TestDeleteValidatorIdx_DeleteWorks(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	epoch := uint64(2)
	v.InsertActivatedVal(epoch+1, []uint64{0, 1, 2})
	v.InsertExitedVal(epoch+1, []uint64{0, 2})
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
	chainService := setupBeaconChain(t, db, nil)
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
	if v.ExitedValFromEpoch(epoch) != nil {
		t.Errorf("Activated validators mapping for epoch %d still there", epoch)
	}
}

func TestSaveValidatorIdx_SaveRetrieveWorks(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	epoch := uint64(1)
	v.InsertActivatedVal(epoch+1, []uint64{0, 1, 2})
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
	chainService := setupBeaconChain(t, db, nil)
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

	if v.ActivatedValFromEpoch(epoch) != nil {
		t.Errorf("Activated validators mapping for epoch %d still there", epoch)
	}
}
