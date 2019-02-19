package rpc

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"

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

	depositData, err := helpers.EncodeDepositData(
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
	beaconState, err := state.GenesisBeaconState(deposits, 0, nil)
	if err != nil {
		t.Fatalf("Could not instantiate genesis state: %v", err)
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

	genesisTime := time.Unix(0, 0).Unix()
	deposits := make([]*pbp2p.Deposit, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(deposits); i++ {
		var pubKey [96]byte
		copy(pubKey[:], []byte(strconv.Itoa(i)))
		depositInput := &pbp2p.DepositInput{
			Pubkey: pubKey[:],
		}
		depositData, err := helpers.EncodeDepositData(
			depositInput,
			params.BeaconConfig().MaxDepositAmount,
			genesisTime,
		)
		if err != nil {
			t.Fatalf("Could not encode genesis block deposits: %v", err)
		}
		deposits[i] = &pbp2p.Deposit{DepositData: depositData}
	}
	beaconState, err := state.GenesisBeaconState(deposits, uint64(genesisTime), nil)
	if err != nil {
		t.Fatalf("Could not instantiate genesis state: %v", err)
	}

	if err := db.UpdateChainHead(genesis, beaconState); err != nil {
		t.Fatalf("Could not save genesis state: %v", err)
	}

	validatorServer := &ValidatorServer{
		beaconDB: db,
	}
	var pubKey [96]byte
	copy(pubKey[:], []byte("0"))
	req := &pb.ValidatorEpochAssignmentsRequest{
		EpochStart: params.BeaconConfig().GenesisSlot,
		PublicKey:  pubKey[:],
	}
	if _, err := validatorServer.ValidatorEpochAssignments(context.Background(), req); err != nil {
		t.Errorf("Could not get validator index: %v", err)
	}
}

func TestValidatorEpochAssignments_WrongPubkeyLength(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	validatorServer := &ValidatorServer{
		beaconDB: db,
	}
	req := &pb.ValidatorEpochAssignmentsRequest{
		EpochStart: params.BeaconConfig().GenesisSlot,
		PublicKey:  []byte{},
	}
	want := fmt.Sprintf("expected public key to have length %d", params.BeaconConfig().BLSPubkeyLength)
	if _, err := validatorServer.ValidatorEpochAssignments(context.Background(), req); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestValidatorCommitteeAtSlot_CrosslinkCommitteesFailure(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	genesis := b.NewGenesisBlock([]byte{})
	if err := db.SaveBlock(genesis); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}

	genesisTime := time.Unix(0, 0).Unix()
	deposits := make([]*pbp2p.Deposit, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(deposits); i++ {
		var pubKey [96]byte
		copy(pubKey[:], []byte(strconv.Itoa(i)))
		depositInput := &pbp2p.DepositInput{
			Pubkey: pubKey[:],
		}
		depositData, err := helpers.EncodeDepositData(
			depositInput,
			params.BeaconConfig().MaxDepositAmount,
			genesisTime,
		)
		if err != nil {
			t.Fatalf("Could not encode genesis block deposits: %v", err)
		}
		deposits[i] = &pbp2p.Deposit{DepositData: depositData}
	}
	beaconState, err := state.GenesisBeaconState(deposits, uint64(genesisTime), nil)
	if err != nil {
		t.Fatalf("Could not instantiate genesis state: %v", err)
	}

	if err := db.UpdateChainHead(genesis, beaconState); err != nil {
		t.Fatalf("Could not save genesis state: %v", err)
	}
	validatorServer := &ValidatorServer{
		beaconDB: db,
	}
	req := &pb.CommitteeRequest{
		Slot: params.BeaconConfig().SlotsPerEpoch * 10,
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

	genesisTime := time.Unix(0, 0).Unix()
	deposits := make([]*pbp2p.Deposit, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(deposits); i++ {
		var pubKey [96]byte
		copy(pubKey[:], []byte(strconv.Itoa(i)))
		depositInput := &pbp2p.DepositInput{
			Pubkey: pubKey[:],
		}
		depositData, err := helpers.EncodeDepositData(
			depositInput,
			params.BeaconConfig().MaxDepositAmount,
			genesisTime,
		)
		if err != nil {
			t.Fatalf("Could not encode genesis block deposits: %v", err)
		}
		deposits[i] = &pbp2p.Deposit{DepositData: depositData}
	}
	beaconState, err := state.GenesisBeaconState(deposits, uint64(genesisTime), nil)
	if err != nil {
		t.Fatalf("Could not instantiate genesis state: %v", err)
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
	if _, err := validatorServer.ValidatorCommitteeAtSlot(context.Background(), req); err != nil {
		t.Errorf("Unable to fetch committee at slot: %v", err)
	}
}
