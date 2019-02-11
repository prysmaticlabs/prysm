package rpc

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestValidatorIndex_Ok(t *testing.T) {
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
		params.BeaconConfig().MaxDepositAmount,
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

func TestValidatorEpochAssignments_Ok(t *testing.T) {
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
			Pubkey: pubKey[:],
		}
		depositData, err := b.EncodeDepositData(
			depositInput,
			params.BeaconConfig().MaxDepositAmount,
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
		EpochStart: params.BeaconConfig().GenesisSlot,
		PublicKey:  pubKey[:],
	}
	res, err := validatorServer.ValidatorEpochAssignments(context.Background(), req)
	if err != nil {
		t.Errorf("Could not get validator index: %v", err)
	}
	// With initial shuffling of default 16384 validators, the validator corresponding to
	// public key 0 from genesis slot should correspond to an attester slot of 9223372036854775808 at shard 0.
	if res.Assignment.Shard != 1 {
		t.Errorf(
			"Expected validator with pubkey %#x to be assigned to shard 0, received %d",
			req.PublicKey,
			res.Assignment.Shard,
		)
	}
	if res.Assignment.AttesterSlot != 9223372036854775808 {
		t.Errorf(
			"Expected validator with pubkey %#x to be assigned as attester of slot 9223372036854775808, "+
				"received %d",
			req.PublicKey,
			res.Assignment.AttesterSlot,
		)
	}
}

func TestValidatorCommitteeAtSlot_CrosslinkCommitteesFailure(t *testing.T) {
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
			Pubkey: pubKey[:],
		}
		depositData, err := b.EncodeDepositData(
			depositInput,
			params.BeaconConfig().MaxDepositAmount,
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
	req := &pb.CommitteeRequest{
		Slot: params.BeaconConfig().EpochLength * 10,
	}
	want := "could not get crosslink committees at slot"
	if _, err := validatorServer.ValidatorCommitteeAtSlot(context.Background(), req); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestValidatorCommitteeAtSlot_Ok(t *testing.T) {
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
			Pubkey: pubKey[:],
		}
		depositData, err := b.EncodeDepositData(
			depositInput,
			params.BeaconConfig().MaxDepositAmount,
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
	req := &pb.CommitteeRequest{
		Slot:           params.BeaconConfig().GenesisSlot + 1,
		ValidatorIndex: 31,
	}
	res, err := validatorServer.ValidatorCommitteeAtSlot(context.Background(), req)
	if err != nil {
		t.Fatalf("Unable to fetch committee at slot: %v", err)
	}
	if res.Shard != 0 {
		t.Errorf("Shard for validator at index 31 should be 2, received %d", res.Shard)
	}
}
