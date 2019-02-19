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

	state, err := genesisState(1)
	if err != nil {
		t.Fatalf("Could not setup genesis state: %v", err)
	}

	if err := db.UpdateChainHead(genesis, state); err != nil {
		t.Fatalf("Could not save genesis state: %v", err)
	}

	validatorServer := &ValidatorServer{
		beaconDB: db,
	}
	var pubKey [96]byte
	copy(pubKey[:], []byte(strconv.Itoa(0)))
	req := &pb.ValidatorIndexRequest{
		PublicKey: pubKey[:],
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

	state, err := genesisState(params.BeaconConfig().DepositsForChainStart)
	if err != nil {
		t.Fatalf("Could not setup genesis state: %v", err)
	}

	if err := db.UpdateChainHead(genesis, state); err != nil {
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

	state, err := genesisState(params.BeaconConfig().DepositsForChainStart)
	if err != nil {
		t.Fatalf("Could not setup genesis state: %v", err)
	}

	if err := db.UpdateChainHead(genesis, state); err != nil {
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

	state, err := genesisState(params.BeaconConfig().DepositsForChainStart)
	if err != nil {
		t.Fatalf("Could not setup genesis state: %v", err)
	}

	if err := db.UpdateChainHead(genesis, state); err != nil {
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

func TestNextEpochCommitteeAssignment_CantFindValidatorIdx(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	if err := db.SaveState(&pbp2p.BeaconState{ValidatorRegistry: []*pbp2p.Validator{}}); err != nil {
		t.Fatalf("could not save state: %v", err)
	}
	vs := &ValidatorServer{
		beaconDB: db,
	}
	req := &pb.ValidatorIndexRequest{
		PublicKey: []byte{'A'},
	}
	want := fmt.Sprintf("can't find validator index for public key %#x", req.PublicKey)
	if _, err := vs.NextEpochCommitteeAssignment(context.Background(), req); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestNextEpochCommitteeAssignment_OK(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	genesis := b.NewGenesisBlock([]byte{})
	if err := db.SaveBlock(genesis); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}
	state, err := genesisState(params.BeaconConfig().DepositsForChainStart)
	if err != nil {
		t.Fatalf("Could not setup genesis state: %v", err)
	}
	if err := db.UpdateChainHead(genesis, state); err != nil {
		t.Fatalf("Could not save genesis state: %v", err)
	}
	vs := &ValidatorServer{
		beaconDB: db,
	}

	// Test the first validator in registry.
	var pubKey [96]byte
	copy(pubKey[:], []byte(strconv.Itoa(0)))
	req := &pb.ValidatorIndexRequest{
		PublicKey: pubKey[:],
	}
	res, err := vs.NextEpochCommitteeAssignment(context.Background(), req)
	if err != nil {
		t.Errorf("Could not call next epoch committee assignment %v", err)
	}
	if res.Shard >= params.BeaconConfig().ShardCount {
		t.Errorf("Assigned shard %d can't be higher than %d",
			res.Shard, params.BeaconConfig().ShardCount)
	}
	if res.Slot < state.Slot+params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Assigned slot %d can't be higher than %d",
			res.Slot, state.Slot+params.BeaconConfig().SlotsPerEpoch)
	}

	// Test the last validator in registry.
	lastValidatorIndex := int(params.BeaconConfig().DepositsForChainStart - 1)
	copy(pubKey[:], []byte(strconv.Itoa(lastValidatorIndex)))
	req = &pb.ValidatorIndexRequest{
		PublicKey: pubKey[:],
	}
	res, err = vs.NextEpochCommitteeAssignment(context.Background(), req)
	if err != nil {
		t.Errorf("Could not call next epoch committee assignment %v", err)
	}
	if res.Shard >= params.BeaconConfig().ShardCount {
		t.Errorf("Assigned shard %d can't be higher than %d",
			res.Shard, params.BeaconConfig().ShardCount)
	}
	if res.Slot < state.Slot+params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Assigned slot %d can't be higher than %d",
			res.Slot, state.Slot+params.BeaconConfig().SlotsPerEpoch)
	}
}

func genesisState(validators uint64) (*pbp2p.BeaconState, error) {
	genesisTime := time.Unix(0, 0).Unix()
	deposits := make([]*pbp2p.Deposit, validators)
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
			return nil, err
		}
		deposits[i] = &pbp2p.Deposit{DepositData: depositData}
	}
	return state.GenesisBeaconState(deposits, uint64(genesisTime), nil)
}
