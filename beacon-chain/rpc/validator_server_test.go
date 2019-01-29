package rpc

import (
	"context"
	"testing"
	"time"

	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestValidatorIndex(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	genesis := b.NewGenesisBlock([]byte{})
	if err := db.SaveBlock(genesis); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}

	depositData, err := b.EncodeDepositData(
		&pbp2p.DepositInput{
			Pubkey: []byte{'A'},
		},
		params.BeaconConfig().MaxDepositInGwei,
		time.Now().Unix(),
	)
	if err != nil {
		t.Fatalf("Could not encode deposit input: %v", err)
	}
	deposits := []*pbp2p.Deposit{
		{DepositData: depositData},
	}
	beaconState, err := state.InitialBeaconState(deposits, 0, nil)
	if err != nil {
		t.Fatalf("Could not instantiate initial state: %v", err)
	}

	if err := db.UpdateChainHead(genesis, beaconState); err != nil {
		t.Fatalf("Could not save genesis state: %v", err)
	}

	validatorServer := &ValidatorServer{
		beaconDB: db,
	}
	req := &pb.ValidatorIndexRequest{
		PublicKey: []byte{'A'},
	}
	if _, err := validatorServer.ValidatorIndex(context.Background(), req); err != nil {
		t.Errorf("Could not get validator index: %v", err)
	}
}

func TestValidatorEpochAssignments(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	genesis := b.NewGenesisBlock([]byte{})
	if err := db.SaveBlock(genesis); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}

	depositData, err := b.EncodeDepositData(
		&pbp2p.DepositInput{
			Pubkey: []byte{'A'},
		},
		params.BeaconConfig().MaxDepositInGwei,
		time.Now().Unix(),
	)
	if err != nil {
		t.Fatalf("Could not encode deposit input: %v", err)
	}
	deposits := []*pbp2p.Deposit{
		{DepositData: depositData},
	}
	beaconState, err := state.InitialBeaconState(deposits, 0, nil)
	if err != nil {
		t.Fatalf("Could not instantiate initial state: %v", err)
	}

	if err := db.UpdateChainHead(genesis, beaconState); err != nil {
		t.Fatalf("Could not save genesis state: %v", err)
	}

	validatorServer := &ValidatorServer{
		beaconDB: db,
	}
	req := &pb.ValidatorEpochAssignmentsRequest{
		EpochStart: 0,
		PublicKey:  []byte{'A'},
	}
	res, err := validatorServer.ValidatorEpochAssignments(context.Background(), req)
	if err != nil {
		t.Errorf("Could not get validator index: %v", err)
	}
	t.Log(res)
}
