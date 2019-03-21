package blockchain

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/beacon-chain/attestation"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/forkutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

// Ensure ChainService implements interfaces.
var _ = ForkChoice(&ChainService{})

// Generates an initial genesis block and state using a custom number of initial
// deposits as a helper function for LMD Ghost fork-choice testing.
func generateTestGenesisStateAndBlock(
	t testing.TB,
	numDeposits uint64,
	beaconDB *db.BeaconDB,
) (*pb.BeaconState, *pb.BeaconBlock, [32]byte, [32]byte) {
	deposits := make([]*pb.Deposit, numDeposits)
	for i := 0; i < len(deposits); i++ {
		pubkey := []byte{byte(i)}
		depositInput := &pb.DepositInput{
			Pubkey: pubkey,
		}

		balance := params.BeaconConfig().MaxDepositAmount
		depositData, err := helpers.EncodeDepositData(depositInput, balance, time.Now().Unix())
		if err != nil {
			t.Fatalf("Could not encode deposit: %v", err)
		}
		deposits[i] = &pb.Deposit{DepositData: depositData}
	}
	genesisTime := uint64(time.Unix(0, 0).Unix())
	beaconState, err := state.GenesisBeaconState(deposits, genesisTime, nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := beaconDB.SaveState(beaconState); err != nil {
		t.Fatal(err)
	}
	stateRoot, err := hashutil.HashProto(beaconState)
	if err != nil {
		t.Fatal(err)
	}
	genesisBlock := b.NewGenesisBlock(stateRoot[:])
	if err := beaconDB.SaveBlock(genesisBlock); err != nil {
		t.Fatal(err)
	}
	genesisRoot, err := hashutil.HashBeaconBlock(genesisBlock)
	if err != nil {
		t.Fatal(err)
	}
	return beaconState, genesisBlock, stateRoot, genesisRoot
}

func TestApplyForkChoice_SetsCanonicalHead(t *testing.T) {
	beaconState, err := state.GenesisBeaconState(nil, 0, nil)
	if err != nil {
		t.Fatalf("Cannot create genesis beacon state: %v", err)
	}
	stateRoot, err := hashutil.HashProto(beaconState)
	if err != nil {
		t.Fatalf("Could not tree hash state: %v", err)
	}

	genesis := b.NewGenesisBlock(stateRoot[:])
	genesisRoot, err := hashutil.HashProto(genesis)
	if err != nil {
		t.Fatalf("Could not get genesis block root: %v", err)
	}
	// Table driven tests for various fork choice scenarios.
	tests := []struct {
		blockSlot uint64
		state     *pb.BeaconState
		logAssert string
	}{
		// Higher slot but same state should trigger chain update.
		{
			blockSlot: 64,
			state:     beaconState,
			logAssert: "Chain head block and state updated",
		},
		// Higher slot, different state, but higher last finalized slot.
		{
			blockSlot: 64,
			state:     &pb.BeaconState{FinalizedEpoch: 2},
			logAssert: "Chain head block and state updated",
		},
		// Higher slot, different state, same last finalized slot,
		// but last justified slot.
		{
			blockSlot: 64,
			state: &pb.BeaconState{
				FinalizedEpoch: 0,
				JustifiedEpoch: 2,
			},
			logAssert: "Chain head block and state updated",
		},
	}
	for _, tt := range tests {
		hook := logTest.NewGlobal()
		db := internal.SetupDB(t)
		defer internal.TeardownDB(t, db)
		chainService := setupBeaconChain(t, false, db, true, nil)
		unixTime := uint64(time.Now().Unix())
		deposits, _ := setupInitialDeposits(t, 100)
		if err := db.InitializeState(unixTime, deposits, &pb.Eth1Data{}); err != nil {
			t.Fatalf("Could not initialize beacon state to disk: %v", err)
		}

		stateRoot, err := hashutil.HashProto(tt.state)
		if err != nil {
			t.Fatalf("Could not tree hash state: %v", err)
		}
		block := &pb.BeaconBlock{
			Slot:             tt.blockSlot,
			StateRootHash32:  stateRoot[:],
			ParentRootHash32: genesisRoot[:],
			Eth1Data: &pb.Eth1Data{
				DepositRootHash32: []byte("a"),
				BlockHash32:       []byte("b"),
			},
		}
		if err := chainService.beaconDB.SaveBlock(block); err != nil {
			t.Fatal(err)
		}
		if err := chainService.ApplyForkChoiceRule(context.Background(), block, tt.state); err != nil {
			t.Errorf("Expected head to update, received %v", err)
		}

		if err := chainService.beaconDB.SaveBlock(block); err != nil {
			t.Fatal(err)
		}
		chainService.cancel()
		testutil.AssertLogsContain(t, hook, tt.logAssert)
	}
}

func TestVoteCount_ParentDoesNotExistNoVoteCount(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	genesisBlock := b.NewGenesisBlock([]byte("stateroot"))
	if err := beaconDB.SaveBlock(genesisBlock); err != nil {
		t.Fatal(err)
	}
	potentialHead := &pb.BeaconBlock{
		ParentRootHash32: []byte{'A'}, // We give a bogus parent root hash.
	}
	if err := beaconDB.SaveBlock(potentialHead); err != nil {
		t.Fatal(err)
	}

	voteTargets := make(map[uint64]*pb.BeaconBlock)
	voteTargets[0] = potentialHead
	count, err := VoteCount(genesisBlock, &pb.BeaconState{}, voteTargets, beaconDB)
	if err != nil {
		t.Fatalf("Could not get vote count: %v", err)
	}
	if count != 0 {
		t.Errorf("Wanted vote count 0, got: %d", count)
	}
}

func TestVoteCount_IncreaseCountCorrectly(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	genesisBlock := b.NewGenesisBlock([]byte("stateroot"))
	genesisRoot, err := hashutil.HashBeaconBlock(genesisBlock)
	if err != nil {
		t.Fatal(err)
	}
	if err := beaconDB.SaveBlock(genesisBlock); err != nil {
		t.Fatal(err)
	}

	potentialHead := &pb.BeaconBlock{
		Slot:             params.BeaconConfig().GenesisSlot + 5,
		ParentRootHash32: genesisRoot[:],
	}
	potentialHead2 := &pb.BeaconBlock{
		Slot:             params.BeaconConfig().GenesisSlot + 6,
		ParentRootHash32: genesisRoot[:],
	}
	// We store these potential heads in the DB.
	if err := beaconDB.SaveBlock(potentialHead); err != nil {
		t.Fatal(err)
	}
	if err := beaconDB.SaveBlock(potentialHead2); err != nil {
		t.Fatal(err)
	}
	beaconState := &pb.BeaconState{ValidatorBalances: []uint64{1e9, 1e9}}
	voteTargets := make(map[uint64]*pb.BeaconBlock)
	voteTargets[0] = potentialHead
	voteTargets[1] = potentialHead2
	count, err := VoteCount(genesisBlock, beaconState, voteTargets, beaconDB)
	if err != nil {
		t.Fatalf("Could not fetch vote balances: %v", err)
	}
	if count != 2e9 {
		t.Errorf("Expected total balances 2e9, received %d", count)
	}
}

func TestAttestationTargets_RetrieveWorks(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)

	pubKey := []byte{'A'}
	state := &pb.BeaconState{
		ValidatorRegistry: []*pb.Validator{{
			Pubkey:    pubKey,
			ExitEpoch: params.BeaconConfig().FarFutureEpoch}},
	}

	if err := beaconDB.SaveState(state); err != nil {
		t.Fatalf("could not save state: %v", err)
	}

	block := &pb.BeaconBlock{Slot: 100}
	if err := beaconDB.SaveBlock(block); err != nil {
		t.Fatalf("could not save block: %v", err)
	}
	blockRoot, err := hashutil.HashBeaconBlock(block)
	if err != nil {
		log.Fatalf("could not hash block: %v", err)
	}

	attsService := attestation.NewAttestationService(
		context.Background(),
		&attestation.Config{BeaconDB: beaconDB})

	atts := &pb.Attestation{
		Data: &pb.AttestationData{
			BeaconBlockRootHash32: blockRoot[:],
		}}
	pubKey48 := bytesutil.ToBytes48(pubKey)
	attsService.Store[pubKey48] = atts

	chainService := setupBeaconChain(t, false, beaconDB, true, attsService)
	attestationTargets, err := chainService.attestationTargets(state)
	if err != nil {
		t.Fatalf("Could not get attestation targets: %v", err)
	}
	if attestationTargets[0].validatorIndex != 0 {
		t.Errorf("Wanted validator index 0, got %d", attestationTargets[0].validatorIndex)
	}
	if attestationTargets[0].block.Slot != block.Slot {
		t.Errorf("Wanted attested slot %d, got %d", block.Slot, attestationTargets[0].block.Slot)
	}
}

func TestBlockChildren_2InARow(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)

	chainService := setupBeaconChain(t, false, beaconDB, true, nil)

	state := &pb.BeaconState{
		Slot: 3,
	}

	// Construct the following chain:
	// B1 <- B2 <- B3  (State is slot 3)
	block1 := &pb.BeaconBlock{
		Slot:             1,
		ParentRootHash32: []byte{'A'},
	}
	root1, err := hashutil.HashBeaconBlock(block1)
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}
	if err = chainService.beaconDB.SaveBlock(block1); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(block1, state); err != nil {
		t.Fatalf("Could update chain head: %v", err)
	}

	block2 := &pb.BeaconBlock{
		Slot:             2,
		ParentRootHash32: root1[:],
	}
	root2, err := hashutil.HashBeaconBlock(block2)
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}
	if err = chainService.beaconDB.SaveBlock(block2); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(block2, state); err != nil {
		t.Fatalf("Could update chain head: %v", err)
	}

	block3 := &pb.BeaconBlock{
		Slot:             3,
		ParentRootHash32: root2[:],
	}
	if err = chainService.beaconDB.SaveBlock(block3); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(block3, state); err != nil {
		t.Fatalf("Could update chain head: %v", err)
	}

	childrenBlock, err := chainService.blockChildren(block1, state)
	if err != nil {
		t.Fatalf("Could not get block children: %v", err)
	}

	// When we input block B1, we should get B2 back.
	wanted := []*pb.BeaconBlock{block2}
	if !reflect.DeepEqual(wanted, childrenBlock) {
		t.Errorf("Wrong children block received")
	}
}

func TestBlockChildren_ChainSplits(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)

	chainService := setupBeaconChain(t, false, beaconDB, true, nil)

	state := &pb.BeaconState{
		Slot: 10,
	}

	// Construct the following chain:
	//     /- B2
	// B1 <- B3 (State is slot 10)
	//      \- B4
	block1 := &pb.BeaconBlock{
		Slot:             1,
		ParentRootHash32: []byte{'A'},
	}
	root1, err := hashutil.HashBeaconBlock(block1)
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}
	if err = chainService.beaconDB.SaveBlock(block1); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(block1, state); err != nil {
		t.Fatalf("Could update chain head: %v", err)
	}

	block2 := &pb.BeaconBlock{
		Slot:             2,
		ParentRootHash32: root1[:],
	}
	if err = chainService.beaconDB.SaveBlock(block2); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(block2, state); err != nil {
		t.Fatalf("Could update chain head: %v", err)
	}

	block3 := &pb.BeaconBlock{
		Slot:             3,
		ParentRootHash32: root1[:],
	}
	if err = chainService.beaconDB.SaveBlock(block3); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(block3, state); err != nil {
		t.Fatalf("Could update chain head: %v", err)
	}

	block4 := &pb.BeaconBlock{
		Slot:             4,
		ParentRootHash32: root1[:],
	}
	if err = chainService.beaconDB.SaveBlock(block4); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(block4, state); err != nil {
		t.Fatalf("Could update chain head: %v", err)
	}

	childrenBlock, err := chainService.blockChildren(block1, state)
	if err != nil {
		t.Fatalf("Could not get block children: %v", err)
	}

	// When we input block B1, we should get B2, B3 and B4 back.
	wanted := []*pb.BeaconBlock{block2, block3, block4}
	if !reflect.DeepEqual(wanted, childrenBlock) {
		t.Errorf("Wrong children block received")
	}
}

func TestBlockChildren_SkipSlots(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)

	chainService := setupBeaconChain(t, false, beaconDB, true, nil)

	state := &pb.BeaconState{
		Slot: 10,
	}

	// Construct the following chain:
	// B1 <- B5 <- B9 (State is slot 10)
	block1 := &pb.BeaconBlock{
		Slot:             1,
		ParentRootHash32: []byte{'A'},
	}
	root1, err := hashutil.HashBeaconBlock(block1)
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}
	if err = chainService.beaconDB.SaveBlock(block1); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(block1, state); err != nil {
		t.Fatalf("Could update chain head: %v", err)
	}

	block5 := &pb.BeaconBlock{
		Slot:             5,
		ParentRootHash32: root1[:],
	}
	root2, err := hashutil.HashBeaconBlock(block5)
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}
	if err = chainService.beaconDB.SaveBlock(block5); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(block5, state); err != nil {
		t.Fatalf("Could update chain head: %v", err)
	}

	block9 := &pb.BeaconBlock{
		Slot:             9,
		ParentRootHash32: root2[:],
	}
	if err = chainService.beaconDB.SaveBlock(block9); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(block9, state); err != nil {
		t.Fatalf("Could update chain head: %v", err)
	}

	childrenBlock, err := chainService.blockChildren(block1, state)
	if err != nil {
		t.Fatalf("Could not get block children: %v", err)
	}

	// When we input block B1, we should get B5.
	wanted := []*pb.BeaconBlock{block5}
	if !reflect.DeepEqual(wanted, childrenBlock) {
		t.Errorf("Wrong children block received")
	}
}

func TestLMDGhost_TrivialHeadUpdate(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)

	state := &pb.BeaconState{
		Slot:              10,
		ValidatorBalances: []uint64{params.BeaconConfig().MaxDepositAmount},
	}

	chainService := setupBeaconChain(t, false, beaconDB, true, nil)

	// Construct the following chain:
	// B1 - B2 (State is slot 2)
	block1 := &pb.BeaconBlock{
		Slot:             1,
		ParentRootHash32: []byte{'A'},
	}
	root1, err := hashutil.HashBeaconBlock(block1)
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}
	if err = chainService.beaconDB.SaveBlock(block1); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(block1, state); err != nil {
		t.Fatalf("Could update chain head: %v", err)
	}

	block2 := &pb.BeaconBlock{
		Slot:             2,
		ParentRootHash32: root1[:],
	}
	if err = chainService.beaconDB.SaveBlock(block2); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(block2, state); err != nil {
		t.Fatalf("Could update chain head: %v", err)
	}

	// The only vote is on block 2.
	voteTargets := make(map[uint64]*pb.BeaconBlock)
	voteTargets[0] = block2

	// LMDGhost should pick block 2.
	head, err := chainService.lmdGhost(block1, state, voteTargets)
	if err != nil {
		t.Fatalf("Could not run LMD GHOST: %v", err)
	}
	if !reflect.DeepEqual(block2, head) {
		t.Errorf("Expected head to equal %v, received %v", block1, head)
	}
}

func TestLMDGhost_3WayChainSplitsSameHeight(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)

	state := &pb.BeaconState{
		Slot: 10,
		ValidatorBalances: []uint64{
			params.BeaconConfig().MaxDepositAmount,
			params.BeaconConfig().MaxDepositAmount,
			params.BeaconConfig().MaxDepositAmount,
			params.BeaconConfig().MaxDepositAmount},
	}

	chainService := setupBeaconChain(t, false, beaconDB, true, nil)

	// Construct the following chain:
	//    /- B2
	// B1  - B3 (State is slot 10)
	//    \- B4
	block1 := &pb.BeaconBlock{
		Slot:             1,
		ParentRootHash32: []byte{'A'},
	}
	root1, err := hashutil.HashBeaconBlock(block1)
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}
	if err = chainService.beaconDB.SaveBlock(block1); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(block1, state); err != nil {
		t.Fatalf("Could update chain head: %v", err)
	}

	block2 := &pb.BeaconBlock{
		Slot:             2,
		ParentRootHash32: root1[:],
	}
	if err = chainService.beaconDB.SaveBlock(block2); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(block2, state); err != nil {
		t.Fatalf("Could update chain head: %v", err)
	}

	block3 := &pb.BeaconBlock{
		Slot:             3,
		ParentRootHash32: root1[:],
	}
	if err = chainService.beaconDB.SaveBlock(block3); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(block3, state); err != nil {
		t.Fatalf("Could update chain head: %v", err)
	}

	block4 := &pb.BeaconBlock{
		Slot:             4,
		ParentRootHash32: root1[:],
	}
	if err = chainService.beaconDB.SaveBlock(block4); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(block4, state); err != nil {
		t.Fatalf("Could update chain head: %v", err)
	}

	// Give block 4 the most votes (2).
	voteTargets := make(map[uint64]*pb.BeaconBlock)
	voteTargets[0] = block2
	voteTargets[1] = block3
	voteTargets[2] = block4
	voteTargets[3] = block4
	// LMDGhost should pick block 4.
	head, err := chainService.lmdGhost(block1, state, voteTargets)
	if err != nil {
		t.Fatalf("Could not run LMD GHOST: %v", err)
	}
	if !reflect.DeepEqual(block4, head) {
		t.Errorf("Expected head to equal %v, received %v", block4, head)
	}
}

func TestLMDGhost_2WayChainSplitsDiffHeight(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)

	state := &pb.BeaconState{
		Slot: 10,
		ValidatorBalances: []uint64{
			params.BeaconConfig().MaxDepositAmount,
			params.BeaconConfig().MaxDepositAmount,
			params.BeaconConfig().MaxDepositAmount,
			params.BeaconConfig().MaxDepositAmount},
	}

	chainService := setupBeaconChain(t, false, beaconDB, true, nil)

	// Construct the following chain:
	//    /- B2 - B4 - B6
	// B1  - B3 - B5 (State is slot 10)
	block1 := &pb.BeaconBlock{
		Slot:             1,
		ParentRootHash32: []byte{'A'},
	}
	root1, err := hashutil.HashBeaconBlock(block1)
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}
	if err = chainService.beaconDB.SaveBlock(block1); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(block1, state); err != nil {
		t.Fatalf("Could update chain head: %v", err)
	}

	block2 := &pb.BeaconBlock{
		Slot:             2,
		ParentRootHash32: root1[:],
	}
	root2, err := hashutil.HashBeaconBlock(block2)
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}
	if err = chainService.beaconDB.SaveBlock(block2); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(block2, state); err != nil {
		t.Fatalf("Could update chain head: %v", err)
	}

	block3 := &pb.BeaconBlock{
		Slot:             3,
		ParentRootHash32: root1[:],
	}
	root3, err := hashutil.HashBeaconBlock(block3)
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}
	if err = chainService.beaconDB.SaveBlock(block3); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(block3, state); err != nil {
		t.Fatalf("Could update chain head: %v", err)
	}

	block4 := &pb.BeaconBlock{
		Slot:             4,
		ParentRootHash32: root2[:],
	}
	root4, err := hashutil.HashBeaconBlock(block4)
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}
	if err = chainService.beaconDB.SaveBlock(block4); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(block4, state); err != nil {
		t.Fatalf("Could update chain head: %v", err)
	}

	block5 := &pb.BeaconBlock{
		Slot:             5,
		ParentRootHash32: root3[:],
	}
	if err = chainService.beaconDB.SaveBlock(block5); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(block5, state); err != nil {
		t.Fatalf("Could update chain head: %v", err)
	}

	block6 := &pb.BeaconBlock{
		Slot:             6,
		ParentRootHash32: root4[:],
	}
	if err = chainService.beaconDB.SaveBlock(block6); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(block6, state); err != nil {
		t.Fatalf("Could update chain head: %v", err)
	}

	// Give block 5 the most votes (2).
	voteTargets := make(map[uint64]*pb.BeaconBlock)
	voteTargets[0] = block6
	voteTargets[1] = block5
	voteTargets[2] = block5
	// LMDGhost should pick block 5.
	head, err := chainService.lmdGhost(block1, state, voteTargets)
	if err != nil {
		t.Fatalf("Could not run LMD GHOST: %v", err)
	}
	if !reflect.DeepEqual(block5, head) {
		t.Errorf("Expected head to equal %v, received %v", block5, head)
	}
}

// This benchmarks LMD GHOST fork choice using 8 blocks in a row.
// 8 validators and all validators voted on the last block.
// Ex:
// 	B0 - B1 - B2 - B3 - B4 - B5 - B6 - B7 (8 votes)
func BenchmarkLMDGhost_8Slots_8Validators(b *testing.B) {
	beaconDB := internal.SetupDB(b)
	defer internal.TeardownDB(b, beaconDB)

	validatorCount := 8
	balances := make([]uint64, validatorCount)
	for i := 0; i < validatorCount; i++ {
		balances[i] = params.BeaconConfig().MaxDepositAmount
	}

	chainService := setupBeaconChainBenchmark(b, false, beaconDB, true, nil)

	// Construct 8 blocks. (Epoch length = 8)
	epochLength := uint64(8)
	state := &pb.BeaconState{
		Slot:              epochLength,
		ValidatorBalances: balances,
	}
	genesis := &pb.BeaconBlock{
		Slot:             0,
		ParentRootHash32: []byte{},
	}
	root, err := hashutil.HashBeaconBlock(genesis)
	if err != nil {
		b.Fatalf("Could not hash block: %v", err)
	}
	if err = chainService.beaconDB.SaveBlock(genesis); err != nil {
		b.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(genesis, state); err != nil {
		b.Fatalf("Could update chain head: %v", err)
	}

	var block *pb.BeaconBlock
	for i := 1; i < int(epochLength); i++ {
		block = &pb.BeaconBlock{
			Slot:             uint64(i),
			ParentRootHash32: root[:],
		}
		if err = chainService.beaconDB.SaveBlock(block); err != nil {
			b.Fatalf("Could not save block: %v", err)
		}
		if err = chainService.beaconDB.UpdateChainHead(block, state); err != nil {
			b.Fatalf("Could update chain head: %v", err)
		}
		root, err = hashutil.HashBeaconBlock(block)
		if err != nil {
			b.Fatalf("Could not hash block: %v", err)
		}
	}

	voteTargets := make(map[uint64]*pb.BeaconBlock)
	for i := 0; i < validatorCount; i++ {
		voteTargets[uint64(i)] = block
	}

	for i := 0; i < b.N; i++ {
		_, err := chainService.lmdGhost(genesis, state, voteTargets)
		if err != nil {
			b.Fatalf("Could not run LMD GHOST: %v", err)
		}
	}
}

// This benchmarks LMD GHOST fork choice 32 blocks in a row.
// This is assuming the worst case where no finalization happens
// for 4 epochs in our Sapphire test net. (epoch length is 8 slots)
// 8 validators and all validators voted on the last block.
// Ex:
// 	B0 - B1 - B2 - B3 - B4 - B5 - B6 - B7 (8 votes)
func BenchmarkLMDGhost_32Slots_8Validators(b *testing.B) {
	beaconDB := internal.SetupDB(b)
	defer internal.TeardownDB(b, beaconDB)

	validatorCount := 8
	balances := make([]uint64, validatorCount)
	for i := 0; i < validatorCount; i++ {
		balances[i] = params.BeaconConfig().MaxDepositAmount
	}

	chainService := setupBeaconChainBenchmark(b, false, beaconDB, true, nil)

	// Construct 8 blocks. (Epoch length = 8)
	epochLength := uint64(8)
	state := &pb.BeaconState{
		Slot:              epochLength,
		ValidatorBalances: balances,
	}
	genesis := &pb.BeaconBlock{
		Slot:             0,
		ParentRootHash32: []byte{},
	}
	root, err := hashutil.HashBeaconBlock(genesis)
	if err != nil {
		b.Fatalf("Could not hash block: %v", err)
	}
	if err = chainService.beaconDB.SaveBlock(genesis); err != nil {
		b.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(genesis, state); err != nil {
		b.Fatalf("Could update chain head: %v", err)
	}

	var block *pb.BeaconBlock
	for i := 1; i < int(epochLength); i++ {
		block = &pb.BeaconBlock{
			Slot:             uint64(i),
			ParentRootHash32: root[:],
		}
		if err = chainService.beaconDB.SaveBlock(block); err != nil {
			b.Fatalf("Could not save block: %v", err)
		}
		if err = chainService.beaconDB.UpdateChainHead(block, state); err != nil {
			b.Fatalf("Could update chain head: %v", err)
		}
		root, err = hashutil.HashBeaconBlock(block)
		if err != nil {
			b.Fatalf("Could not hash block: %v", err)
		}
	}

	voteTargets := make(map[uint64]*pb.BeaconBlock)
	for i := 0; i < validatorCount; i++ {
		voteTargets[uint64(i)] = block
	}

	for i := 0; i < b.N; i++ {
		_, err := chainService.lmdGhost(genesis, state, voteTargets)
		if err != nil {
			b.Fatalf("Could not run LMD GHOST: %v", err)
		}
	}
}

// This test benchmarks LMD GHOST fork choice using 32 blocks in a row.
// 64 validators and all validators voted on the last block.
// Ex:
// 	B0 - B1 - B2 - ... - B32 (64 votes)
func BenchmarkLMDGhost_32Slots_64Validators(b *testing.B) {
	beaconDB := internal.SetupDB(b)
	defer internal.TeardownDB(b, beaconDB)

	validatorCount := 64
	balances := make([]uint64, validatorCount)
	for i := 0; i < validatorCount; i++ {
		balances[i] = params.BeaconConfig().MaxDepositAmount
	}

	chainService := setupBeaconChainBenchmark(b, false, beaconDB, true, nil)

	// Construct 64 blocks. (Epoch length = 64)
	epochLength := uint64(32)
	state := &pb.BeaconState{
		Slot:              epochLength,
		ValidatorBalances: balances,
	}
	genesis := &pb.BeaconBlock{
		Slot:             0,
		ParentRootHash32: []byte{},
	}
	root, err := hashutil.HashBeaconBlock(genesis)
	if err != nil {
		b.Fatalf("Could not hash block: %v", err)
	}
	if err = chainService.beaconDB.SaveBlock(genesis); err != nil {
		b.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(genesis, state); err != nil {
		b.Fatalf("Could update chain head: %v", err)
	}

	var block *pb.BeaconBlock
	for i := 1; i < int(epochLength); i++ {
		block = &pb.BeaconBlock{
			Slot:             uint64(i),
			ParentRootHash32: root[:],
		}
		if err = chainService.beaconDB.SaveBlock(block); err != nil {
			b.Fatalf("Could not save block: %v", err)
		}
		if err = chainService.beaconDB.UpdateChainHead(block, state); err != nil {
			b.Fatalf("Could update chain head: %v", err)
		}
		root, err = hashutil.HashBeaconBlock(block)
		if err != nil {
			b.Fatalf("Could not hash block: %v", err)
		}
	}

	voteTargets := make(map[uint64]*pb.BeaconBlock)
	for i := 0; i < validatorCount; i++ {
		voteTargets[uint64(i)] = block
	}

	for i := 0; i < b.N; i++ {
		_, err := chainService.lmdGhost(genesis, state, voteTargets)
		if err != nil {
			b.Fatalf("Could not run LMD GHOST: %v", err)
		}
	}
}

// This test benchmarks LMD GHOST fork choice using 64 blocks in a row.
// 16384 validators and all validators voted on the last block.
// Ex:
// 	B0 - B1 - B2 - ... - B64 (16384 votes)
func BenchmarkLMDGhost_64Slots_16384Validators(b *testing.B) {
	beaconDB := internal.SetupDB(b)
	defer internal.TeardownDB(b, beaconDB)

	validatorCount := 16384
	balances := make([]uint64, validatorCount)
	for i := 0; i < validatorCount; i++ {
		balances[i] = params.BeaconConfig().MaxDepositAmount
	}

	chainService := setupBeaconChainBenchmark(b, false, beaconDB, true, nil)

	// Construct 64 blocks. (Epoch length = 64)
	epochLength := uint64(64)
	state := &pb.BeaconState{
		Slot:              epochLength,
		ValidatorBalances: balances,
	}
	genesis := &pb.BeaconBlock{
		Slot:             0,
		ParentRootHash32: []byte{},
	}
	root, err := hashutil.HashBeaconBlock(genesis)
	if err != nil {
		b.Fatalf("Could not hash block: %v", err)
	}
	if err = chainService.beaconDB.SaveBlock(genesis); err != nil {
		b.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(genesis, state); err != nil {
		b.Fatalf("Could update chain head: %v", err)
	}

	var block *pb.BeaconBlock
	for i := 1; i < int(epochLength); i++ {
		block = &pb.BeaconBlock{
			Slot:             uint64(i),
			ParentRootHash32: root[:],
		}
		if err = chainService.beaconDB.SaveBlock(block); err != nil {
			b.Fatalf("Could not save block: %v", err)
		}
		if err = chainService.beaconDB.UpdateChainHead(block, state); err != nil {
			b.Fatalf("Could update chain head: %v", err)
		}
		root, err = hashutil.HashBeaconBlock(block)
		if err != nil {
			b.Fatalf("Could not hash block: %v", err)
		}
	}

	voteTargets := make(map[uint64]*pb.BeaconBlock)
	for i := 0; i < validatorCount; i++ {
		voteTargets[uint64(i)] = block
	}

	for i := 0; i < b.N; i++ {
		_, err := chainService.lmdGhost(genesis, state, voteTargets)
		if err != nil {
			b.Fatalf("Could not run LMD GHOST: %v", err)
		}
	}
}

func setupBeaconChainBenchmark(b *testing.B, faultyPoWClient bool, beaconDB *db.BeaconDB, enablePOWChain bool, attsService *attestation.Service) *ChainService {
	endpoint := "ws://127.0.0.1"
	ctx := context.Background()
	var web3Service *powchain.Web3Service
	var err error
	if enablePOWChain {
		if faultyPoWClient {
			client := &faultyClient{}
			web3Service, err = powchain.NewWeb3Service(ctx, &powchain.Web3ServiceConfig{
				Endpoint:        endpoint,
				DepositContract: common.Address{},
				Reader:          client,
				Client:          client,
				Logger:          client,
			})
		} else {
			client := &mockClient{}
			web3Service, err = powchain.NewWeb3Service(ctx, &powchain.Web3ServiceConfig{
				Endpoint:        endpoint,
				DepositContract: common.Address{},
				Reader:          client,
				Client:          client,
				Logger:          client,
			})
		}
	}
	if err != nil {
		b.Fatalf("unable to set up web3 service: %v", err)
	}

	cfg := &Config{
		BeaconBlockBuf: 0,
		BeaconDB:       beaconDB,
		Web3Service:    web3Service,
		OpsPoolService: &mockOperationService{},
		EnablePOWChain: enablePOWChain,
		AttsService:    attsService,
	}
	if err != nil {
		b.Fatalf("could not register blockchain service: %v", err)
	}
	chainService, err := NewChainService(ctx, cfg)
	if err != nil {
		b.Fatalf("unable to setup chain service: %v", err)
	}

	return chainService
}

func TestUpdateFFGCheckPts_NewJustifiedSlot(t *testing.T) {
	genesisSlot := params.BeaconConfig().GenesisSlot
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	chainSvc := setupBeaconChain(t, false, db, true, nil)
	gBlockRoot, gBlock, gState, privKeys := setupFFGTest(t)
	if err := chainSvc.beaconDB.SaveBlock(gBlock); err != nil {
		t.Fatal(err)
	}
	if err := chainSvc.beaconDB.UpdateChainHead(gBlock, gState); err != nil {
		t.Fatal(err)
	}
	if err := chainSvc.beaconDB.SaveFinalizedState(gState); err != nil {
		t.Fatal(err)
	}

	// Last justified check point happened at slot 0.
	if err := chainSvc.beaconDB.SaveJustifiedBlock(
		&pb.BeaconBlock{Slot: genesisSlot}); err != nil {
		t.Fatal(err)
	}

	// Also saved finalized block to slot 0 to test justification case only.
	if err := chainSvc.beaconDB.SaveFinalizedBlock(&pb.BeaconBlock{Slot: genesisSlot}); err != nil {
		t.Fatal(err)
	}

	// New justified slot in state is at slot 64.
	offset := uint64(64)
	proposerIdx, err := helpers.BeaconProposerIndex(gState, genesisSlot+offset)
	if err != nil {
		t.Fatal(err)
	}
	gState.JustifiedEpoch = params.BeaconConfig().GenesisEpoch + 1
	gState.Slot = genesisSlot + offset
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, gState.JustifiedEpoch)
	domain := forkutil.DomainVersion(gState.Fork, gState.JustifiedEpoch, params.BeaconConfig().DomainRandao)
	epochSignature := privKeys[proposerIdx].Sign(buf, domain)
	block := &pb.BeaconBlock{
		Slot:             genesisSlot + offset,
		RandaoReveal:     epochSignature.Marshal(),
		ParentRootHash32: gBlockRoot[:],
		Body:             &pb.BeaconBlockBody{}}
	if err := chainSvc.beaconDB.SaveBlock(block); err != nil {
		t.Fatal(err)
	}
	if err := chainSvc.beaconDB.UpdateChainHead(block, gState); err != nil {
		t.Fatal(err)
	}
	if err := chainSvc.updateFFGCheckPts(gState); err != nil {
		t.Fatal(err)
	}

	// Getting latest justification check point from DB and
	// verify they have been updated.
	newJustifiedState, err := chainSvc.beaconDB.JustifiedState()
	if err != nil {
		t.Fatal(err)
	}
	newJustifiedBlock, err := chainSvc.beaconDB.JustifiedBlock()
	if err != nil {
		t.Fatal(err)
	}
	if newJustifiedState.Slot-genesisSlot != offset {
		t.Errorf("Wanted justification state slot: %d, got: %d",
			offset, newJustifiedState.Slot-genesisSlot)
	}
	if newJustifiedBlock.Slot-genesisSlot != offset {
		t.Errorf("Wanted justification block slot: %d, got: %d",
			offset, newJustifiedBlock.Slot-genesisSlot)
	}
}

func TestUpdateFFGCheckPts_NewFinalizedSlot(t *testing.T) {
	genesisSlot := params.BeaconConfig().GenesisSlot
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	chainSvc := setupBeaconChain(t, false, db, true, nil)

	gBlockRoot, gBlock, gState, privKeys := setupFFGTest(t)
	if err := chainSvc.beaconDB.SaveBlock(gBlock); err != nil {
		t.Fatal(err)
	}
	if err := chainSvc.beaconDB.UpdateChainHead(gBlock, gState); err != nil {
		t.Fatal(err)
	}
	if err := chainSvc.beaconDB.SaveFinalizedState(gState); err != nil {
		t.Fatal(err)
	}

	// Last finalized check point happened at slot 0.
	if err := chainSvc.beaconDB.SaveFinalizedBlock(
		gBlock); err != nil {
		t.Fatal(err)
	}
	if err := chainSvc.beaconDB.SaveFinalizedState(
		gState); err != nil {
		t.Fatal(err)
	}

	// New Finalized slot in state is at slot 64.
	offset := uint64(64)
	proposerIdx, err := helpers.BeaconProposerIndex(gState, genesisSlot+offset)
	if err != nil {
		t.Fatal(err)
	}

	// Also saved justified block to slot 0 to test finalized case only.
	if err := chainSvc.beaconDB.SaveJustifiedBlock(
		&pb.BeaconBlock{Slot: genesisSlot}); err != nil {
		t.Fatal(err)
	}

	gState.FinalizedEpoch = params.BeaconConfig().GenesisEpoch + 1
	gState.Slot = genesisSlot + offset
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, gState.FinalizedEpoch)
	domain := forkutil.DomainVersion(gState.Fork, gState.FinalizedEpoch, params.BeaconConfig().DomainRandao)
	epochSignature := privKeys[proposerIdx].Sign(buf, domain)
	block := &pb.BeaconBlock{
		Slot:             genesisSlot + offset,
		RandaoReveal:     epochSignature.Marshal(),
		ParentRootHash32: gBlockRoot[:],
		Body:             &pb.BeaconBlockBody{}}

	if err := chainSvc.beaconDB.SaveBlock(block); err != nil {
		t.Fatal(err)
	}
	if err := chainSvc.beaconDB.UpdateChainHead(block, gState); err != nil {
		t.Fatal(err)
	}
	if err := chainSvc.updateFFGCheckPts(gState); err != nil {
		t.Fatal(err)
	}

	// Getting latest justification check point from DB and
	// verify they have been updated.
	newFinalizedState, err := chainSvc.beaconDB.FinalizedState()
	if err != nil {
		t.Fatal(err)
	}
	newFinalizedBlock, err := chainSvc.beaconDB.FinalizedBlock()
	if err != nil {
		t.Fatal(err)
	}
	if newFinalizedState.Slot-genesisSlot != offset {
		t.Errorf("Wanted finalized state slot: %d, got: %d",
			offset, newFinalizedState.Slot-genesisSlot)
	}
	if newFinalizedBlock.Slot-genesisSlot != offset {
		t.Errorf("Wanted finalized block slot: %d, got: %d",
			offset, newFinalizedBlock.Slot-genesisSlot)
	}
}

func TestUpdateFFGCheckPts_NewJustifiedSkipSlot(t *testing.T) {
	genesisSlot := params.BeaconConfig().GenesisSlot
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	chainSvc := setupBeaconChain(t, false, db, true, nil)
	gBlockRoot, gBlock, gState, privKeys := setupFFGTest(t)
	if err := chainSvc.beaconDB.SaveBlock(gBlock); err != nil {
		t.Fatal(err)
	}
	if err := chainSvc.beaconDB.UpdateChainHead(gBlock, gState); err != nil {
		t.Fatal(err)
	}
	if err := chainSvc.beaconDB.SaveFinalizedState(gState); err != nil {
		t.Fatal(err)
	}

	// Last justified check point happened at slot 0.
	if err := chainSvc.beaconDB.SaveJustifiedBlock(
		&pb.BeaconBlock{Slot: genesisSlot}); err != nil {
		t.Fatal(err)
	}

	// Also saved finalized block to slot 0 to test justification case only.
	if err := chainSvc.beaconDB.SaveFinalizedBlock(
		&pb.BeaconBlock{Slot: genesisSlot}); err != nil {
		t.Fatal(err)
	}

	// New justified slot in state is at slot 64, but it's a skip slot...
	offset := uint64(64)
	lastAvailableSlot := uint64(60)
	proposerIdx, err := helpers.BeaconProposerIndex(gState, genesisSlot+lastAvailableSlot)
	if err != nil {
		t.Fatal(err)
	}
	gState.JustifiedEpoch = params.BeaconConfig().GenesisEpoch + 1
	gState.Slot = genesisSlot + offset
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, params.BeaconConfig().GenesisEpoch)
	domain := forkutil.DomainVersion(gState.Fork, params.BeaconConfig().GenesisEpoch, params.BeaconConfig().DomainRandao)
	epochSignature := privKeys[proposerIdx].Sign(buf, domain)
	block := &pb.BeaconBlock{
		Slot:             genesisSlot + lastAvailableSlot,
		RandaoReveal:     epochSignature.Marshal(),
		ParentRootHash32: gBlockRoot[:],
		Body:             &pb.BeaconBlockBody{}}
	if err := chainSvc.beaconDB.SaveBlock(block); err != nil {
		t.Fatal(err)
	}
	if err := chainSvc.beaconDB.UpdateChainHead(block, gState); err != nil {
		t.Fatal(err)
	}
	if err := chainSvc.updateFFGCheckPts(gState); err != nil {
		t.Fatal(err)
	}

	// Getting latest justification check point from DB and
	// verify they have been updated.
	newJustifiedState, err := chainSvc.beaconDB.JustifiedState()
	if err != nil {
		t.Fatal(err)
	}
	newJustifiedBlock, err := chainSvc.beaconDB.JustifiedBlock()
	if err != nil {
		t.Fatal(err)
	}
	if newJustifiedState.Slot-genesisSlot != offset {
		t.Errorf("Wanted justification state slot: %d, got: %d",
			offset, newJustifiedState.Slot-genesisSlot)
	}
	if newJustifiedBlock.Slot-genesisSlot != lastAvailableSlot {
		t.Errorf("Wanted justification block slot: %d, got: %d",
			offset, newJustifiedBlock.Slot-genesisSlot)
	}
}

func setupFFGTest(t *testing.T) ([32]byte, *pb.BeaconBlock, *pb.BeaconState, []*bls.SecretKey) {
	genesisSlot := params.BeaconConfig().GenesisSlot
	var crosslinks []*pb.Crosslink
	for i := 0; i < int(params.BeaconConfig().ShardCount); i++ {
		crosslinks = append(crosslinks, &pb.Crosslink{Epoch: params.BeaconConfig().GenesisEpoch})
	}
	latestRandaoMixes := make(
		[][]byte,
		params.BeaconConfig().LatestRandaoMixesLength,
	)
	for i := 0; i < len(latestRandaoMixes); i++ {
		latestRandaoMixes[i] = make([]byte, 32)
	}
	var validatorRegistry []*pb.Validator
	var validatorBalances []uint64
	var privKeys []*bls.SecretKey
	for i := uint64(0); i < 64; i++ {
		priv, err := bls.RandKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		privKeys = append(privKeys, priv)
		validatorRegistry = append(validatorRegistry,
			&pb.Validator{
				Pubkey:    priv.PublicKey().Marshal(),
				ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			})
		validatorBalances = append(validatorBalances, params.BeaconConfig().MaxDepositAmount)
	}
	gBlock := &pb.BeaconBlock{Slot: genesisSlot}
	gBlockRoot, err := hashutil.HashBeaconBlock(gBlock)
	if err != nil {
		t.Fatal(err)
	}
	gState := &pb.BeaconState{
		Slot:                   genesisSlot,
		LatestBlockRootHash32S: make([][]byte, params.BeaconConfig().LatestBlockRootsLength),
		LatestRandaoMixes:      latestRandaoMixes,
		LatestIndexRootHash32S: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
		LatestSlashedBalances:  make([]uint64, params.BeaconConfig().LatestSlashedExitLength),
		LatestCrosslinks:       crosslinks,
		ValidatorRegistry:      validatorRegistry,
		ValidatorBalances:      validatorBalances,
		LatestBlock:            gBlock,
		Fork: &pb.Fork{
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			Epoch:           params.BeaconConfig().GenesisEpoch,
		},
	}
	return gBlockRoot, gBlock, gState, privKeys
}
