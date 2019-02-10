package db

import (
	"bytes"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func setupInitialDeposits(t *testing.T) []*pb.Deposit {
	genesisValidatorRegistry := validators.InitialValidatorRegistry()
	deposits := make([]*pb.Deposit, len(genesisValidatorRegistry))
	for i := 0; i < len(deposits); i++ {
		depositInput := &pb.DepositInput{
			Pubkey: genesisValidatorRegistry[i].Pubkey,
		}
		balance := params.BeaconConfig().MaxDepositAmount
		depositData, err := blocks.EncodeDepositData(depositInput, balance, time.Now().Unix())
		if err != nil {
			t.Fatalf("Cannot encode data: %v", err)
		}
		deposits[i] = &pb.Deposit{DepositData: depositData}
	}
	return deposits
}

func TestInitializeState(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	genesisTime := uint64(time.Now().Unix())
	deposits := setupInitialDeposits(t)
	if err := db.InitializeState(genesisTime, deposits); err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}
	b, err := db.ChainHead()
	if err != nil {
		t.Fatalf("Failed to get chain head: %v", err)
	}
	if b.GetSlot() != params.BeaconConfig().GenesisSlot {
		t.Fatalf("Expected block height to equal 1. Got %d", b.GetSlot())
	}

	beaconState, err := db.State()
	if err != nil {
		t.Fatalf("Failed to get state: %v", err)
	}
	if beaconState == nil {
		t.Fatalf("Failed to retrieve state: %v", beaconState)
	}
	beaconStateEnc, err := proto.Marshal(beaconState)
	if err != nil {
		t.Fatalf("Failed to encode state: %v", err)
	}

	statePrime, err := db.State()
	if err != nil {
		t.Fatalf("Failed to get state: %v", err)
	}
	statePrimeEnc, err := proto.Marshal(statePrime)
	if err != nil {
		t.Fatalf("Failed to encode state: %v", err)
	}

	if !bytes.Equal(beaconStateEnc, statePrimeEnc) {
		t.Fatalf("Expected %#x and %#x to be equal", beaconStateEnc, statePrimeEnc)
	}
}

func TestGenesisTime(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	genesisTime, err := db.GenesisTime()
	if err == nil {
		t.Fatal("expected GenesisTime to fail")
	}

	deposits := setupInitialDeposits(t)
	if err := db.InitializeState(uint64(genesisTime.Unix()), deposits); err != nil {
		t.Fatalf("failed to initialize state: %v", err)
	}

	time1, err := db.GenesisTime()
	if err != nil {
		t.Fatalf("GenesisTime failed on second attempt: %v", err)
	}
	time2, err := db.GenesisTime()
	if err != nil {
		t.Fatalf("GenesisTime failed on second attempt: %v", err)
	}

	if time1 != time2 {
		t.Fatalf("Expected %v and %v to be equal", time1, time2)
	}
}
