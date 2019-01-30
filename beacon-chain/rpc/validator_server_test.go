package rpc

import (
	"context"
	"strconv"
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
		params.BeaconConfig().MaxDeposit,
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

	genesisTime := params.BeaconConfig().GenesisTime.Unix()
	deposits := make([]*pbp2p.Deposit, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(deposits); i++ {
		var pubKey [48]byte
		copy(pubKey[:], []byte(strconv.Itoa(i)))
		depositInput := &pbp2p.DepositInput{
			Pubkey:                 pubKey[:],
			RandaoCommitmentHash32: []byte{0},
		}
		depositData, err := b.EncodeDepositData(
			depositInput,
			params.BeaconConfig().MaxDeposit,
			genesisTime,
		)
		if err != nil {
			t.Fatalf("Could not encode initial block deposits: %v", err)
		}
		deposits[i] = &pbp2p.Deposit{DepositData: depositData}
	}
	beaconState, err := state.InitialBeaconState(deposits, uint64(genesisTime), nil)
	if err != nil {
		t.Fatalf("Could not instantiate initial state: %v", err)
	}

	if err := db.UpdateChainHead(genesis, beaconState); err != nil {
		t.Fatalf("Could not save genesis state: %v", err)
	}

	validatorServer := &ValidatorServer{
		beaconDB: db,
	}
	var pubKey [48]byte
	copy(pubKey[:], []byte("0"))
	req := &pb.ValidatorEpochAssignmentsRequest{
		EpochStart: 0,
		PublicKey:  pubKey[:],
	}
	res, err := validatorServer.ValidatorEpochAssignments(context.Background(), req)
	if err != nil {
		t.Errorf("Could not get validator index: %v", err)
	}
	// With initial shuffling of default 16384 validators, the validator corresponding to
	// public key 0 should correspond to an attester slot of 2 at shard 5.
	if res.Assignment.Shard != 5 {
		t.Errorf(
			"Expected validator with pubkey %#x to be assigned to shard 5, received %d",
			req.PublicKey,
			res.Assignment.Shard,
		)
	}
	if res.Assignment.AttesterSlot != 2 {
		t.Errorf(
			"Expected validator with pubkey %#x to be assigned as attester of slot 2, received %d",
			req.PublicKey,
			res.Assignment.AttesterSlot,
		)
	}
}
