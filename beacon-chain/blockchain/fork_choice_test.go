package blockchain

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestLMDGhost_TrivialHeadUpdate(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	genesisValidatorRegistry := validators.InitialValidatorRegistry()
	deposits := make([]*pb.Deposit, len(genesisValidatorRegistry))
	for i := 0; i < len(deposits); i++ {
		depositInput := &pb.DepositInput{
			Pubkey: genesisValidatorRegistry[i].Pubkey,
		}
		balance := params.BeaconConfig().MaxDeposit * params.BeaconConfig().Gwei
		depositData, err := b.EncodeDepositData(depositInput, balance, time.Now().Unix())
		if err != nil {
			t.Fatal(err)
		}
		deposits[i] = &pb.Deposit{DepositData: depositData}
	}
	genesisTime := uint64(params.BeaconConfig().GenesisTime.Unix())
	beaconState, err := state.InitialBeaconState(deposits, genesisTime, nil)
	if err != nil {
		t.Fatal(err)
	}

	// #nosec G104
	stateEnc, _ := proto.Marshal(beaconState)
	if err := db.SaveState(beaconState); err != nil {
		t.Fatal(err)
	}
	stateHash := hashutil.Hash(stateEnc)
	genesisBlock := b.NewGenesisBlock(stateHash[:])
	if err := db.SaveBlock(genesisBlock); err != nil {
		t.Fatal(err)
	}
	genesisEnc, _ := proto.Marshal(genesisBlock)
	genesisHash := hashutil.Hash(genesisEnc)
	potentialHead := &pb.BeaconBlock{
		Slot:             5,
		ParentRootHash32: genesisHash[:],
		StateRootHash32:  stateHash[:],
	}
	potentialHeadEnc, _ := proto.Marshal(potentialHead)
	potentialHeadHash := hashutil.Hash(potentialHeadEnc)

	// We store these potential heads in the DB.
	if err := db.SaveBlock(potentialHead); err != nil {
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
	if err := db.SaveLatestAttestationForValidator(0, latestAtt); err != nil {
		t.Fatal(err)
	}

	// We then test LMD Ghost was applied as the fork-choice rule with a single observed block.
	observedBlocks := []*pb.BeaconBlock{potentialHead}
	head, err := LMDGhost(beaconState, genesisBlock, observedBlocks, db)
	if err != nil {
		t.Fatalf("Could not run LMD GHOST: %v", err)
	}
	if !reflect.DeepEqual(potentialHead, head) {
		t.Errorf("Expected head to equal %v, received %v", potentialHead, head)
	}
}

func TestLMDGhost_TrivialHigherVoteCountWins(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	genesisValidatorRegistry := validators.InitialValidatorRegistry()
	deposits := make([]*pb.Deposit, len(genesisValidatorRegistry))
	for i := 0; i < len(deposits); i++ {
		depositInput := &pb.DepositInput{
			Pubkey: genesisValidatorRegistry[i].Pubkey,
		}
		balance := params.BeaconConfig().MaxDeposit * params.BeaconConfig().Gwei
		depositData, err := b.EncodeDepositData(depositInput, balance, time.Now().Unix())
		if err != nil {
			t.Fatal(err)
		}
		deposits[i] = &pb.Deposit{DepositData: depositData}
	}
	genesisTime := uint64(params.BeaconConfig().GenesisTime.Unix())
	beaconState, err := state.InitialBeaconState(deposits, genesisTime, nil)
	if err != nil {
		t.Fatal(err)
	}

	// #nosec G104
	stateEnc, _ := proto.Marshal(beaconState)
	if err := db.SaveState(beaconState); err != nil {
		t.Fatal(err)
	}
	stateHash := hashutil.Hash(stateEnc)
	genesisBlock := b.NewGenesisBlock(stateHash[:])
	if err := db.SaveBlock(genesisBlock); err != nil {
		t.Fatal(err)
	}
	genesisEnc, _ := proto.Marshal(genesisBlock)
	genesisHash := hashutil.Hash(genesisEnc)
	lowerVoteBlock := &pb.BeaconBlock{
		Slot:             5,
		ParentRootHash32: genesisHash[:],
		StateRootHash32:  stateHash[:],
	}
	higherVoteBlock := &pb.BeaconBlock{
		Slot:             5,
		ParentRootHash32: genesisHash[:],
		StateRootHash32:  []byte("some-other-head"),
	}
	higherVoteBlockEnc, _ := proto.Marshal(higherVoteBlock)
	higherVoteBlockHash := hashutil.Hash(higherVoteBlockEnc)

	// We store these potential heads in the DB.
	if err := db.SaveBlock(lowerVoteBlock); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(higherVoteBlock); err != nil {
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
	if err := db.SaveLatestAttestationForValidator(0, latestAtt); err != nil {
		t.Fatal(err)
	}

	// We then test LMD Ghost was applied as the fork-choice rule.
	observedBlocks := []*pb.BeaconBlock{lowerVoteBlock, higherVoteBlock}
	head, err := LMDGhost(beaconState, genesisBlock, observedBlocks, db)
	if err != nil {
		t.Fatalf("Could not run LMD GHOST: %v", err)
	}

	// We expect that higherVoteBlock has more votes than lowerVoteBlock, allowing it to be
	// selected by the fork-choice rule.
	if !reflect.DeepEqual(higherVoteBlock, head) {
		t.Errorf("Expected head to equal %v, received %v", higherVoteBlock, head)
	}
}

func TestVoteCount_ParentDoesNotExist(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	genesisBlock := b.NewGenesisBlock([]byte{})
	if err := db.SaveBlock(genesisBlock); err != nil {
		t.Fatal(err)
	}
	potentialHead := &pb.BeaconBlock{
		Slot:             5,
		ParentRootHash32: []byte{}, // We give a bogus parent root hash.
	}
	if err := db.SaveBlock(potentialHead); err != nil {
		t.Fatal(err)
	}
	targets := []*pb.BeaconBlock{potentialHead}
	want := "parent block does not exist"
	if _, err := VoteCount(genesisBlock, targets, db); !strings.Contains(err.Error(), want) {
		t.Fatalf("Expected %s, received %v", want, err)
	}
}

func TestVoteCount_IncreaseCountCorrectly(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	genesisBlock := b.NewGenesisBlock([]byte{})
	genesisEnc, _ := proto.Marshal(genesisBlock)
	genesisHash := hashutil.Hash(genesisEnc)
	if err := db.SaveBlock(genesisBlock); err != nil {
		t.Fatal(err)
	}

	potentialHead := &pb.BeaconBlock{
		Slot:             5,
		ParentRootHash32: genesisHash[:],
	}
	potentialHead2 := &pb.BeaconBlock{
		Slot:             5,
		ParentRootHash32: genesisHash[:],
	}
	// We store these potential heads in the DB.
	if err := db.SaveBlock(potentialHead); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(potentialHead2); err != nil {
		t.Fatal(err)
	}
	targets := []*pb.BeaconBlock{potentialHead, potentialHead2}
	count, err := VoteCount(genesisBlock, targets, db)
	if err != nil {
		t.Fatalf("Could not fetch vote count: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 votes, received %d", count)
	}
}

func TestBlockChildren(t *testing.T) {
	genesisBlock := b.NewGenesisBlock([]byte{})
	genesisEnc, _ := proto.Marshal(genesisBlock)
	genesisHash := hashutil.Hash(genesisEnc)
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
