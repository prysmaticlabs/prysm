package blockchain

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
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
		balance := params.BeaconConfig().MaxDeposit
		depositData, err := b.EncodeDepositData(depositInput, balance, time.Now().Unix())
		if err != nil {
			t.Fatalf("Could not encode deposit: %v", err)
		}
		deposits[i] = &pb.Deposit{DepositData: depositData}
	}
	genesisTime := uint64(params.BeaconConfig().GenesisTime.Unix())
	beaconState, err := state.InitialBeaconState(deposits, genesisTime, nil)
	if err != nil {
		t.Fatal(err)
	}

	stateEnc, err := proto.Marshal(beaconState)
	if err != nil {
		t.Fatal(err)
	}
	if err := beaconDB.SaveState(beaconState); err != nil {
		t.Fatal(err)
	}
	stateHash := hashutil.Hash(stateEnc)
	genesisBlock := b.NewGenesisBlock(stateHash[:])
	if err := beaconDB.SaveBlock(genesisBlock); err != nil {
		t.Fatal(err)
	}
	genesisHash, err := hashutil.HashBeaconBlock(genesisBlock)
	if err != nil {
		t.Fatal(err)
	}
	return beaconState, genesisBlock, stateHash, genesisHash
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

	voteTargets := make(map[[32]byte]*pb.BeaconBlock)
	key := bytesutil.ToBytes32(beaconState.ValidatorRegistry[0].Pubkey)
	voteTargets[key] = potentialHead

	head, err := LMDGhost(genesisBlock, voteTargets, observedBlocks, beaconDB)
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

	voteTargets := make(map[[32]byte]*pb.BeaconBlock)
	key := bytesutil.ToBytes32(beaconState.ValidatorRegistry[0].Pubkey)
	voteTargets[key] = candidate2

	// We then test LMD Ghost was applied as the fork-choice rule.
	observedBlocks := []*pb.BeaconBlock{candidate1, candidate2}
	head, err := LMDGhost(genesisBlock, voteTargets, observedBlocks, beaconDB)
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
	candidate1, candidate2 := setupConflictingBlocks(t, beaconDB, genesisHash, stateHash)

	activeIndices := helpers.ActiveValidatorIndices(beaconState.ValidatorRegistry, 0)
	// We store some simulated latest attestation target for every active validator in a map.
	voteTargets := make(map[[32]byte]*pb.BeaconBlock, len(activeIndices))
	for i := 0; i < len(activeIndices); i++ {
		key := bytesutil.ToBytes32([]byte(fmt.Sprintf("%d", i)))
		voteTargets[key] = candidate2
	}

	// We then test LMD Ghost was applied as the fork-choice rule.
	observedBlocks := []*pb.BeaconBlock{candidate1, candidate2}
	head, err := LMDGhost(genesisBlock, voteTargets, observedBlocks, beaconDB)
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
	genesisBlock := b.NewGenesisBlock([]byte{})
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

	voteTargets := make(map[[32]byte]*pb.BeaconBlock)
	voteTargets[[32]byte{}] = potentialHead
	want := "parent block does not exist"
	if _, err := VoteCount(genesisBlock, voteTargets, beaconDB); !strings.Contains(err.Error(), want) {
		t.Fatalf("Expected %s, received %v", want, err)
	}
}

func TestVoteCount_IncreaseCountCorrectly(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	genesisBlock := b.NewGenesisBlock([]byte{})
	genesisHash, err := hashutil.HashBeaconBlock(genesisBlock)
	if err != nil {
		t.Fatal(err)
	}
	if err := beaconDB.SaveBlock(genesisBlock); err != nil {
		t.Fatal(err)
	}

	potentialHead := &pb.BeaconBlock{
		Slot:             5,
		ParentRootHash32: genesisHash[:],
	}
	potentialHead2 := &pb.BeaconBlock{
		Slot:             6,
		ParentRootHash32: genesisHash[:],
	}
	// We store these potential heads in the DB.
	if err := beaconDB.SaveBlock(potentialHead); err != nil {
		t.Fatal(err)
	}
	if err := beaconDB.SaveBlock(potentialHead2); err != nil {
		t.Fatal(err)
	}

	voteTargets := make(map[[32]byte]*pb.BeaconBlock)
	voteTargets[[32]byte{0}] = potentialHead
	voteTargets[[32]byte{1}] = potentialHead2
	count, err := VoteCount(genesisBlock, voteTargets, beaconDB)
	if err != nil {
		t.Fatalf("Could not fetch vote count: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 1 vote, received %d", count)
	}
}
