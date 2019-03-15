package rpc

import (
	"context"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

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

func TestValidatorIndex_OK(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	pubKey := []byte{'A'}
	if err := db.SaveValidatorIndex(pubKey, 0); err != nil {
		t.Fatalf("Could not save validator index: %v", err)
	}

	validatorServer := &ValidatorServer{
		beaconDB: db,
	}

	req := &pb.ValidatorIndexRequest{
		PublicKey: pubKey,
	}
	if _, err := validatorServer.ValidatorIndex(context.Background(), req); err != nil {
		t.Errorf("Could not get validator index: %v", err)
	}
}

func TestNextEpochCommitteeAssignment_WrongPubkeyLength(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	validatorServer := &ValidatorServer{
		beaconDB: db,
	}
	req := &pb.ValidatorEpochAssignmentsRequest{
		PublicKey:  []byte{},
		EpochStart: params.BeaconConfig().GenesisEpoch,
	}
	want := fmt.Sprintf("expected public key to have length %d", params.BeaconConfig().BLSPubkeyLength)
	if _, err := validatorServer.CommitteeAssignment(context.Background(), req); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
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

	pubKey := make([]byte, 96)
	req := &pb.ValidatorEpochAssignmentsRequest{
		PublicKey:  pubKey,
		EpochStart: params.BeaconConfig().GenesisEpoch,
	}
	want := fmt.Sprintf("validator %#x does not exist", req.PublicKey)
	if _, err := vs.CommitteeAssignment(context.Background(), req); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestCommitteeAssignment_OK(t *testing.T) {
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
	var wg sync.WaitGroup
	numOfValidators := int(params.BeaconConfig().DepositsForChainStart)
	errs := make(chan error, numOfValidators)
	for i := 0; i < numOfValidators; i++ {
		pubKeyBuf := make([]byte, params.BeaconConfig().BLSPubkeyLength)
		binary.PutUvarint(pubKeyBuf, uint64(i))
		wg.Add(1)
		go func(index int) {
			errs <- db.SaveValidatorIndexBatch(pubKeyBuf, index)
			wg.Done()
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("Could not save validator index: %v", err)
		}
	}

	vs := &ValidatorServer{
		beaconDB: db,
	}

	pubKeyBuf := make([]byte, params.BeaconConfig().BLSPubkeyLength)
	binary.PutUvarint(pubKeyBuf, 0)
	// Test the first validator in registry.
	req := &pb.ValidatorEpochAssignmentsRequest{
		PublicKey:  pubKeyBuf,
		EpochStart: params.BeaconConfig().GenesisSlot,
	}
	res, err := vs.CommitteeAssignment(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not call epoch committee assignment %v", err)
	}
	if res.Shard >= params.BeaconConfig().ShardCount {
		t.Errorf("Assigned shard %d can't be higher than %d",
			res.Shard, params.BeaconConfig().ShardCount)
	}
	if res.Slot > state.Slot+params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Assigned slot %d can't be higher than %d",
			res.Slot, state.Slot+params.BeaconConfig().SlotsPerEpoch)
	}

	// Test the last validator in registry.
	lastValidatorIndex := params.BeaconConfig().DepositsForChainStart - 1
	pubKeyBuf = make([]byte, params.BeaconConfig().BLSPubkeyLength)
	binary.PutUvarint(pubKeyBuf, lastValidatorIndex)
	req = &pb.ValidatorEpochAssignmentsRequest{
		PublicKey:  pubKeyBuf,
		EpochStart: params.BeaconConfig().GenesisSlot,
	}
	res, err = vs.CommitteeAssignment(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not call epoch committee assignment %v", err)
	}
	if res.Shard >= params.BeaconConfig().ShardCount {
		t.Errorf("Assigned shard %d can't be higher than %d",
			res.Shard, params.BeaconConfig().ShardCount)
	}
	if res.Slot > state.Slot+params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Assigned slot %d can't be higher than %d",
			res.Slot, state.Slot+params.BeaconConfig().SlotsPerEpoch)
	}
}

func TestValidatorStatus_CantFindValidatorIdx(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	if err := db.SaveState(&pbp2p.BeaconState{ValidatorRegistry: []*pbp2p.Validator{}}); err != nil {
		t.Fatalf("could not save state: %v", err)
	}
	vs := &ValidatorServer{
		beaconDB: db,
	}
	req := &pb.ValidatorIndexRequest{
		PublicKey: []byte{'B'},
	}
	want := fmt.Sprintf("validator %#x does not exist", req.PublicKey)
	if _, err := vs.ValidatorStatus(context.Background(), req); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestValidatorStatus_PendingActive(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	pubKey := []byte{'A'}
	if err := db.SaveValidatorIndex(pubKey, 0); err != nil {
		t.Fatalf("Could not save validator index: %v", err)
	}

	// Pending active because activation epoch is still defaulted at far future slot.
	if err := db.SaveState(&pbp2p.BeaconState{ValidatorRegistry: []*pbp2p.Validator{
		{ActivationEpoch: params.BeaconConfig().FarFutureEpoch, Pubkey: pubKey},
	}}); err != nil {
		t.Fatalf("could not save state: %v", err)
	}

	vs := &ValidatorServer{
		beaconDB: db,
	}
	req := &pb.ValidatorIndexRequest{
		PublicKey: pubKey,
	}
	resp, err := vs.ValidatorStatus(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not get validator status %v", err)
	}
	if resp.Status != pb.ValidatorStatus_PENDING_ACTIVE {
		t.Errorf("Wanted %v, got %v", pb.ValidatorStatus_PENDING_ACTIVE, resp.Status)
	}
}

func TestValidatorStatus_Active(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	pubKey := []byte{'A'}
	if err := db.SaveValidatorIndex(pubKey, 0); err != nil {
		t.Fatalf("Could not save validator index: %v", err)
	}

	// Active because activation epoch <= current epoch < exit epoch.
	if err := db.SaveState(&pbp2p.BeaconState{
		Slot: params.BeaconConfig().GenesisSlot,
		ValidatorRegistry: []*pbp2p.Validator{{
			ActivationEpoch: params.BeaconConfig().GenesisEpoch,
			ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
			Pubkey:          pubKey},
		}}); err != nil {
		t.Fatalf("could not save state: %v", err)
	}

	vs := &ValidatorServer{
		beaconDB: db,
	}
	req := &pb.ValidatorIndexRequest{
		PublicKey: pubKey,
	}
	resp, err := vs.ValidatorStatus(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not get validator status %v", err)
	}
	if resp.Status != pb.ValidatorStatus_ACTIVE {
		t.Errorf("Wanted %v, got %v", pb.ValidatorStatus_ACTIVE, resp.Status)
	}
}

func TestValidatorStatus_InitiatedExit(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	pubKey := []byte{'A'}
	if err := db.SaveValidatorIndex(pubKey, 0); err != nil {
		t.Fatalf("Could not save validator index: %v", err)
	}

	// Initiated exit because validator status flag = Validator_INITIATED_EXIT.
	if err := db.SaveState(&pbp2p.BeaconState{
		Slot: params.BeaconConfig().GenesisSlot,
		ValidatorRegistry: []*pbp2p.Validator{{
			StatusFlags: pbp2p.Validator_INITIATED_EXIT,
			Pubkey:      pubKey},
		}}); err != nil {
		t.Fatalf("could not save state: %v", err)
	}

	vs := &ValidatorServer{
		beaconDB: db,
	}
	req := &pb.ValidatorIndexRequest{
		PublicKey: pubKey,
	}
	resp, err := vs.ValidatorStatus(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not get validator status %v", err)
	}
	if resp.Status != pb.ValidatorStatus_INITIATED_EXIT {
		t.Errorf("Wanted %v, got %v", pb.ValidatorStatus_INITIATED_EXIT, resp.Status)
	}
}

func TestValidatorStatus_Withdrawable(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	pubKey := []byte{'A'}
	if err := db.SaveValidatorIndex(pubKey, 0); err != nil {
		t.Fatalf("Could not save validator index: %v", err)
	}

	// Withdrawable exit because validator status flag = Validator_WITHDRAWABLE.
	if err := db.SaveState(&pbp2p.BeaconState{
		Slot: params.BeaconConfig().GenesisSlot,
		ValidatorRegistry: []*pbp2p.Validator{{
			StatusFlags: pbp2p.Validator_WITHDRAWABLE,
			Pubkey:      pubKey},
		}}); err != nil {
		t.Fatalf("could not save state: %v", err)
	}

	vs := &ValidatorServer{
		beaconDB: db,
	}
	req := &pb.ValidatorIndexRequest{
		PublicKey: pubKey,
	}
	resp, err := vs.ValidatorStatus(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not get validator status %v", err)
	}
	if resp.Status != pb.ValidatorStatus_WITHDRAWABLE {
		t.Errorf("Wanted %v, got %v", pb.ValidatorStatus_WITHDRAWABLE, resp.Status)
	}
}

func TestValidatorStatus_ExitedSlashed(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	pubKey := []byte{'A'}
	if err := db.SaveValidatorIndex(pubKey, 0); err != nil {
		t.Fatalf("Could not save validator index: %v", err)
	}

	// Exit slashed because exit epoch and slashed epoch are =< current epoch.
	if err := db.SaveState(&pbp2p.BeaconState{
		Slot: params.BeaconConfig().GenesisSlot,
		ValidatorRegistry: []*pbp2p.Validator{{
			Pubkey: pubKey},
		}}); err != nil {
		t.Fatalf("could not save state: %v", err)
	}

	vs := &ValidatorServer{
		beaconDB: db,
	}
	req := &pb.ValidatorIndexRequest{
		PublicKey: pubKey,
	}
	resp, err := vs.ValidatorStatus(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not get validator status %v", err)
	}
	if resp.Status != pb.ValidatorStatus_EXITED_SLASHED {
		t.Errorf("Wanted %v, got %v", pb.ValidatorStatus_EXITED_SLASHED, resp.Status)
	}
}

func TestValidatorStatus_Exited(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	pubKey := []byte{'A'}
	if err := db.SaveValidatorIndex(pubKey, 0); err != nil {
		t.Fatalf("Could not save validator index: %v", err)
	}

	// Exit because only exit epoch is =< current epoch.
	if err := db.SaveState(&pbp2p.BeaconState{
		Slot: params.BeaconConfig().GenesisSlot + 64,
		ValidatorRegistry: []*pbp2p.Validator{{
			Pubkey:       pubKey,
			SlashedEpoch: params.BeaconConfig().FarFutureEpoch},
		}}); err != nil {
		t.Fatalf("could not save state: %v", err)
	}

	vs := &ValidatorServer{
		beaconDB: db,
	}
	req := &pb.ValidatorIndexRequest{
		PublicKey: pubKey,
	}
	resp, err := vs.ValidatorStatus(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not get validator status %v", err)
	}
	if resp.Status != pb.ValidatorStatus_EXITED {
		t.Errorf("Wanted %v, got %v", pb.ValidatorStatus_EXITED, resp.Status)
	}
}

func TestValidatorStatus_UnknownStatus(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	pubKey := []byte{'A'}
	if err := db.SaveValidatorIndex(pubKey, 0); err != nil {
		t.Fatalf("Could not save validator index: %v", err)
	}

	if err := db.SaveState(&pbp2p.BeaconState{
		Slot: params.BeaconConfig().GenesisSlot,
		ValidatorRegistry: []*pbp2p.Validator{{
			ActivationEpoch: params.BeaconConfig().GenesisSlot,
			ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
			Pubkey:          pubKey},
		}}); err != nil {
		t.Fatalf("could not save state: %v", err)
	}

	vs := &ValidatorServer{
		beaconDB: db,
	}
	req := &pb.ValidatorIndexRequest{
		PublicKey: pubKey,
	}
	resp, err := vs.ValidatorStatus(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not get validator status %v", err)
	}
	if resp.Status != pb.ValidatorStatus_UNKNOWN_STATUS {
		t.Errorf("Wanted %v, got %v", pb.ValidatorStatus_UNKNOWN_STATUS, resp.Status)
	}
}

func TestWaitForActivation_ContextClosed(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	beaconState := &pbp2p.BeaconState{
		Slot: params.BeaconConfig().GenesisSlot,
	}
	if err := db.SaveState(beaconState); err != nil {
		t.Fatalf("could not save state: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	vs := &ValidatorServer{
		beaconDB:           db,
		ctx:                ctx,
		chainService:       newMockChainService(),
		canonicalStateChan: make(chan *pbp2p.BeaconState, 1),
	}
	req := &pb.ValidatorActivationRequest{
		Pubkey: []byte("A"),
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := internal.NewMockValidatorService_WaitForActivationServer(ctrl)
	exitRoutine := make(chan bool)
	go func(tt *testing.T) {
		want := "context closed"
		if err := vs.WaitForActivation(req, mockStream); !strings.Contains(err.Error(), want) {
			tt.Errorf("Could not call RPC method: %v", err)
		}
		<-exitRoutine
	}(t)
	cancel()
	exitRoutine <- true
}

func TestWaitForActivation_ValidatorOriginallyExists(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	pubKey := []byte{'A'}
	if err := db.SaveValidatorIndex(pubKey, 0); err != nil {
		t.Fatalf("Could not save validator index: %v", err)
	}

	beaconState := &pbp2p.BeaconState{
		Slot: params.BeaconConfig().GenesisSlot,
		ValidatorRegistry: []*pbp2p.Validator{{
			ActivationEpoch: params.BeaconConfig().GenesisSlot,
			ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
			Pubkey:          pubKey},
		},
	}
	if err := db.SaveState(beaconState); err != nil {
		t.Fatalf("could not save state: %v", err)
	}

	vs := &ValidatorServer{
		beaconDB:           db,
		ctx:                context.Background(),
		chainService:       newMockChainService(),
		canonicalStateChan: make(chan *pbp2p.BeaconState, 1),
	}
	req := &pb.ValidatorActivationRequest{
		Pubkey: pubKey,
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := internal.NewMockValidatorService_WaitForActivationServer(ctrl)
	mockStream.EXPECT().Send(
		&pb.ValidatorActivationResponse{
			Validator: beaconState.ValidatorRegistry[0],
		},
	).Return(nil)

	if err := vs.WaitForActivation(req, mockStream); err != nil {
		t.Fatalf("Could not setup wait for activation stream: %v", err)
	}
}
