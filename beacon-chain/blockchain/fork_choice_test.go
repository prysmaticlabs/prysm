package blockchain

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared/ssz"

	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
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
		depositData, err := b.EncodeDepositData(depositInput, balance, time.Now().Unix())
		if err != nil {
			t.Fatalf("Could not encode deposit: %v", err)
		}
		deposits[i] = &pb.Deposit{DepositData: depositData}
	}
	genesisTime := uint64(time.Unix(0, 0).Unix())
	beaconState, err := state.InitialBeaconState(deposits, genesisTime, nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := beaconDB.SaveState(beaconState); err != nil {
		t.Fatal(err)
	}
	stateRoot, err := ssz.TreeHash(beaconState)
	if err != nil {
		t.Fatal(err)
	}
	genesisBlock := b.NewGenesisBlock(stateRoot[:])
	if err := beaconDB.SaveBlock(genesisBlock); err != nil {
		t.Fatal(err)
	}
	genesisRoot, err := ssz.TreeHash(genesisBlock)
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

func TestLMDGhost_TrivialHeadUpdate(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	beaconState, genesisBlock, stateHash, genesisHash := generateTestGenesisStateAndBlock(
		t,
		100,
		beaconDB,
	)
	potentialHead := &pb.BeaconBlock{
		Slot:             5,
		ParentRootHash32: genesisHash[:],
		StateRootHash32:  stateHash[:],
	}

	// We store these potential heads in the DB.
	if err := beaconDB.SaveBlock(potentialHead); err != nil {
		t.Fatal(err)
	}

	// We then test LMD Ghost was applied as the fork-choice rule with a single observed block.
	observedBlocks := []*pb.BeaconBlock{potentialHead}

	voteTargets := make(map[uint64]*pb.BeaconBlock)
	voteTargets[0] = potentialHead

	head, err := LMDGhost(genesisBlock, beaconState, voteTargets, observedBlocks, beaconDB)
	if err != nil {
		t.Fatalf("Could not run LMD GHOST: %v", err)
	}
	if !reflect.DeepEqual(potentialHead, head) {
		t.Errorf("Expected head to equal %v, received %v", potentialHead, head)
	}
}

func TestLMDGhost_TrivialHigherVoteCountWins(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	beaconState, genesisBlock, stateHash, genesisHash := generateTestGenesisStateAndBlock(
		t,
		100,
		beaconDB,
	)

	candidate1, candidate2 := setupConflictingBlocks(t, beaconDB, genesisHash, stateHash)

	voteTargets := make(map[uint64]*pb.BeaconBlock)
	voteTargets[0] = candidate2

	// We then test LMD Ghost was applied as the fork-choice rule.
	observedBlocks := []*pb.BeaconBlock{candidate1, candidate2}
	head, err := LMDGhost(genesisBlock, beaconState, voteTargets, observedBlocks, beaconDB)
	if err != nil {
		t.Fatalf("Could not run LMD GHOST: %v", err)
	}

	// We expect that higherVoteBlock has more votes than lowerVoteBlock, allowing it to be
	// selected by the fork-choice rule.
	if !reflect.DeepEqual(candidate2, head) {
		t.Errorf("Expected head to equal %v, received %v", candidate2, head)
	}
}

func TestLMDGhost_EveryActiveValidatorHasLatestAttestation(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	beaconState, genesisBlock, stateHash, genesisHash := generateTestGenesisStateAndBlock(
		t,
		params.BeaconConfig().DepositsForChainStart,
		beaconDB,
	)
	beaconState.ValidatorBalances[0] = 32e9
	candidate1, candidate2 := setupConflictingBlocks(t, beaconDB, genesisHash, stateHash)

	activeIndices := helpers.ActiveValidatorIndices(beaconState.ValidatorRegistry, 0)
	// We store some simulated latest attestation target for every active validator in a map.
	voteTargets := make(map[uint64]*pb.BeaconBlock, len(activeIndices))
	for i := 0; i < len(activeIndices); i++ {
		voteTargets[uint64(i)] = candidate2
	}

	// We then test LMD Ghost was applied as the fork-choice rule.
	observedBlocks := []*pb.BeaconBlock{candidate1, candidate2}
	head, err := LMDGhost(genesisBlock, beaconState, voteTargets, observedBlocks, beaconDB)
	if err != nil {
		t.Fatalf("Could not run LMD GHOST: %v", err)
	}

	// We expect that higherVoteBlock to have overwhelmingly more votes
	// than lowerVoteBlock, allowing it to be selected by the fork-choice rule.
	if !reflect.DeepEqual(candidate2, head) {
		t.Errorf("Expected head to equal %v, received %v", candidate2, head)
	}
}

func TestVoteCount_ParentDoesNotExist(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	genesisBlock := b.NewGenesisBlock([]byte("stateroot"))
	if err := beaconDB.SaveBlock(genesisBlock); err != nil {
		t.Fatal(err)
	}
	potentialHead := &pb.BeaconBlock{
		Slot:             5,
		ParentRootHash32: []byte{}, // We give a bogus parent root hash.
	}
	if err := beaconDB.SaveBlock(potentialHead); err != nil {
		t.Fatal(err)
	}

	voteTargets := make(map[uint64]*pb.BeaconBlock)
	voteTargets[0] = potentialHead
	want := "parent block does not exist"
	if _, err := VoteCount(genesisBlock, &pb.BeaconState{}, voteTargets, beaconDB); !strings.Contains(err.Error(), want) {
		t.Fatalf("Expected %s, received %v", want, err)
	}
}

func TestVoteCount_IncreaseCountCorrectly(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	genesisBlock := b.NewGenesisBlock([]byte("stateroot"))
	genesisRoot, err := ssz.TreeHash(genesisBlock)
	if err != nil {
		t.Fatal(err)
	}
	if err := beaconDB.SaveBlock(genesisBlock); err != nil {
		t.Fatal(err)
	}

	potentialHead := &pb.BeaconBlock{
		Slot:             5,
		ParentRootHash32: genesisRoot[:],
	}
	potentialHead2 := &pb.BeaconBlock{
		Slot:             6,
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
