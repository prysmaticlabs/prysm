package rpc

import (
	"context"
	"fmt"
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
	res, err := validatorServer.ValidatorCommitteeAtSlot(context.Background(), req)
	if err != nil {
		t.Fatalf("Unable to fetch committee at slot: %v", err)
	}
	if res.Shard != 0 {
		t.Errorf("Shard for validator at index 31 should be 2, received %d", res.Shard)
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
	if res.Shard != 1 {
		t.Errorf("Wanted shard 5, got: %d", res.Shard)
	}
	if res.Slot != 9223372036854775872 {
		t.Errorf("Wanted slot 9223372036854775872, got: %d", res.Slot)
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
	if res.Shard != 126 {
		t.Errorf("Wanted shard 126, got: %d", res.Shard)
	}
	if res.Slot != 9223372036854775935 {
		t.Errorf("Wanted slot 9223372036854775935, got: %d", res.Slot)
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
		depositData, err := b.EncodeDepositData(
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
