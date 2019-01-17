package blockchain

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"

	"github.com/gogo/protobuf/proto"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	bytesutil "github.com/prysmaticlabs/prysm/shared/bytes"
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
		balance := params.BeaconConfig().MaxDeposit * params.BeaconConfig().Gwei
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
	potentialHeadEnc, _ := proto.Marshal(potentialHead)
	potentialHeadHash := hashutil.Hash(potentialHeadEnc)

	// We store these potential heads in the DB.
	if err := beaconDB.SaveBlock(potentialHead); err != nil {
		t.Fatal(err)
	}

	// We store some simulated latest attestation for an active validator.
	latestAtt := &pb.Attestation{
		Data: &pb.AttestationData{
			Slot:                  3,
			BeaconBlockRootHash32: potentialHeadHash[:],
		},
	}
	// We ensure the block target of potentialHead has 1 vote corresponding to validator
	// at index 0.
	if err := beaconDB.SaveLatestAttestationForValidator(0, latestAtt); err != nil {
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
	lowerVoteBlock := &pb.BeaconBlock{
		Slot:             5,
		ParentRootHash32: genesisHash[:],
		StateRootHash32:  stateHash[:],
	}
	higherVoteBlock := &pb.BeaconBlock{
		Slot:             5,
		ParentRootHash32: genesisHash[:],
		StateRootHash32:  []byte("some-other-state"),
	}
	higherVoteBlockEnc, _ := proto.Marshal(higherVoteBlock)
	higherVoteBlockHash := hashutil.Hash(higherVoteBlockEnc)

	// We store these potential heads in the DB.
	if err := beaconDB.SaveBlock(lowerVoteBlock); err != nil {
		t.Fatal(err)
	}
	if err := beaconDB.SaveBlock(higherVoteBlock); err != nil {
		t.Fatal(err)
	}

	// We store some simulated latest attestation for an active validator.
	latestAtt := &pb.Attestation{
		Data: &pb.AttestationData{
			Slot:                  3,
			BeaconBlockRootHash32: higherVoteBlockHash[:],
		},
	}
	// We ensure the block target of potentialHead has 1 vote corresponding to validator
	// at index 0.
	if err := beaconDB.SaveLatestAttestationForValidator(0, latestAtt); err != nil {
		t.Fatal(err)
	}

	voteTargets := make(map[[32]byte]*pb.BeaconBlock)
	key := bytesutil.ToBytes32(beaconState.ValidatorRegistry[0].Pubkey)
	voteTargets[key] = higherVoteBlock

	// We then test LMD Ghost was applied as the fork-choice rule.
	observedBlocks := []*pb.BeaconBlock{lowerVoteBlock, higherVoteBlock}
	head, err := LMDGhost(genesisBlock, voteTargets, observedBlocks, beaconDB)
	if err != nil {
		t.Fatalf("Could not run LMD GHOST: %v", err)
	}

	// We expect that higherVoteBlock has more votes than lowerVoteBlock, allowing it to be
	// selected by the fork-choice rule.
	if !reflect.DeepEqual(higherVoteBlock, head) {
		t.Errorf("Expected head to equal %v, received %v", higherVoteBlock, head)
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
	lowerVoteBlock := &pb.BeaconBlock{
		Slot:             5,
		ParentRootHash32: genesisHash[:],
		StateRootHash32:  stateHash[:],
	}
	higherVoteBlock := &pb.BeaconBlock{
		Slot:             5,
		ParentRootHash32: genesisHash[:],
		StateRootHash32:  []byte("some-other-state"),
	}

	// We store these potential heads in the DB.
	if err := beaconDB.SaveBlock(lowerVoteBlock); err != nil {
		t.Fatal(err)
	}
	if err := beaconDB.SaveBlock(higherVoteBlock); err != nil {
		t.Fatal(err)
	}

	activeIndices := validators.ActiveValidatorIndices(beaconState.ValidatorRegistry, 0)
	// We store some simulated latest attestation target for every active validator in a map.
	voteTargets := make(map[[32]byte]*pb.BeaconBlock, len(activeIndices))
	for i := 0; i < len(activeIndices); i++ {
		key := bytesutil.ToBytes32([]byte(fmt.Sprintf("%d", i)))
		voteTargets[key] = higherVoteBlock
	}

	// We then test LMD Ghost was applied as the fork-choice rule.
	observedBlocks := []*pb.BeaconBlock{lowerVoteBlock, higherVoteBlock}
	head, err := LMDGhost(genesisBlock, voteTargets, observedBlocks, beaconDB)
	if err != nil {
		t.Fatalf("Could not run LMD GHOST: %v", err)
	}

	// We expect that higherVoteBlock to have overwhelmingly more votes
	// than lowerVoteBlock, allowing it to be selected by the fork-choice rule.
	if !reflect.DeepEqual(higherVoteBlock, head) {
		t.Errorf("Expected head to equal %v, received %v", higherVoteBlock, head)
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

func TestBlockChildren(t *testing.T) {
	genesisBlock := b.NewGenesisBlock([]byte{})
	genesisHash, err := hashutil.HashBeaconBlock(genesisBlock)
	if err != nil {
		t.Fatal(err)
	}
	targets := []*pb.BeaconBlock{
		{
			Slot:             9,
			ParentRootHash32: genesisHash[:],
		},
		{
			Slot:             5,
			ParentRootHash32: []byte{},
		},
		{
			Slot:             8,
			ParentRootHash32: genesisHash[:],
		},
	}
	children, err := BlockChildren(genesisBlock, targets)
	if err != nil {
		t.Fatalf("Could not fetch block children: %v", err)
	}
	if len(children) != 2 {
		t.Errorf("Expected %d children, received %d", 2, len(children))
	}
}
