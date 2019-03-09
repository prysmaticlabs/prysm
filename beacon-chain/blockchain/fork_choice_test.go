package blockchain

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/attestation"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

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

func setupConflictingBlocks(
	t *testing.T,
	beaconDB *db.BeaconDB,
	genesisHash [32]byte,
	stateRoot [32]byte,
) (candidate1 *pb.BeaconBlock, candidate2 *pb.BeaconBlock) {
	candidate1 = &pb.BeaconBlock{
		Slot:             5,
		ParentRootHash32: genesisHash[:],
		StateRootHash32:  stateRoot[:],
	}
	candidate2 = &pb.BeaconBlock{
		Slot:             5,
		ParentRootHash32: genesisHash[:],
		StateRootHash32:  []byte("some-other-state"),
	}
	// We store these potential heads in the DB.
	if err := beaconDB.SaveBlock(candidate1); err != nil {
		t.Fatal(err)
	}
	if err := beaconDB.SaveBlock(candidate2); err != nil {
		t.Fatal(err)
	}
	return candidate1, candidate2
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
