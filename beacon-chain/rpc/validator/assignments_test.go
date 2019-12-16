package validator

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	blk "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestCommitteeAssignment_NextEpoch_WrongPubkeyLength(t *testing.T) {
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)
	ctx := context.Background()

	beaconState, _ := testutil.DeterministicGenesisState(t, 8)
	block := blk.NewGenesisBlock([]byte{})
	if err := db.SaveBlock(ctx, block); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}
	genesisRoot, err := ssz.SigningRoot(block)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
	}

	Server := &Server{
		BeaconDB:    db,
		HeadFetcher: &mockChain.ChainService{State: beaconState, Root: genesisRoot[:]},
		SyncChecker: &mockSync.Sync{IsSyncing: false},
	}
	req := &pb.AssignmentRequest{
		PublicKeys: [][]byte{{1}},
		EpochStart: 0,
	}
	want := fmt.Sprintf("expected public key to have length %d", params.BeaconConfig().BLSPubkeyLength)
	if _, err := Server.CommitteeAssignment(context.Background(), req); err != nil && !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestNextEpochCommitteeAssignment_CantFindValidatorIdx(t *testing.T) {
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)
	ctx := context.Background()
	beaconState, _ := testutil.DeterministicGenesisState(t, 10)

	genesis := blk.NewGenesisBlock([]byte{})
	genesisRoot, err := ssz.SigningRoot(genesis)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
	}

	vs := &Server{
		BeaconDB:    db,
		HeadFetcher: &mockChain.ChainService{State: beaconState, Root: genesisRoot[:]},
		SyncChecker: &mockSync.Sync{IsSyncing: false},
	}

	pubKey := make([]byte, 96)
	req := &pb.AssignmentRequest{
		PublicKeys: [][]byte{pubKey},
		EpochStart: 0,
	}
	want := fmt.Sprintf("validator %#x does not exist", req.PublicKeys[0])
	if _, err := vs.CommitteeAssignment(ctx, req); err != nil && !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestCommitteeAssignment_OK(t *testing.T) {
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)
	ctx := context.Background()

	genesis := blk.NewGenesisBlock([]byte{})
	depChainStart := uint64(64)
	deposits, _, _ := testutil.DeterministicDepositsAndKeys(depChainStart)
	eth1Data, err := testutil.DeterministicEth1Data(len(deposits))
	if err != nil {
		t.Fatal(err)
	}
	state, err := state.GenesisBeaconState(deposits, 0, eth1Data)
	if err != nil {
		t.Fatalf("Could not setup genesis state: %v", err)
	}
	genesisRoot, err := ssz.SigningRoot(genesis)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
	}

	var wg sync.WaitGroup
	numOfValidators := int(depChainStart)
	errs := make(chan error, numOfValidators)
	for i := 0; i < len(deposits); i++ {
		wg.Add(1)
		go func(index int) {
			errs <- db.SaveValidatorIndex(ctx, bytesutil.ToBytes48(deposits[index].Data.PublicKey), uint64(index))
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

	vs := &Server{
		BeaconDB:    db,
		HeadFetcher: &mockChain.ChainService{State: state, Root: genesisRoot[:]},
		SyncChecker: &mockSync.Sync{IsSyncing: false},
	}

	// Test the first validator in registry.
	req := &pb.AssignmentRequest{
		PublicKeys: [][]byte{deposits[0].Data.PublicKey},
		EpochStart: 0,
	}
	res, err := vs.CommitteeAssignment(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not call epoch committee assignment %v", err)
	}
	if res.ValidatorAssignment[0].AttesterSlot > state.Slot+params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Assigned slot %d can't be higher than %d",
			res.ValidatorAssignment[0].AttesterSlot, state.Slot+params.BeaconConfig().SlotsPerEpoch)
	}

	// Test the last validator in registry.
	lastValidatorIndex := depChainStart - 1
	req = &pb.AssignmentRequest{
		PublicKeys: [][]byte{deposits[lastValidatorIndex].Data.PublicKey},
		EpochStart: 0,
	}
	res, err = vs.CommitteeAssignment(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not call epoch committee assignment %v", err)
	}
	if res.ValidatorAssignment[0].AttesterSlot > state.Slot+params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Assigned slot %d can't be higher than %d",
			res.ValidatorAssignment[0].AttesterSlot, state.Slot+params.BeaconConfig().SlotsPerEpoch)
	}
}

func TestCommitteeAssignment_CurrentEpoch_ShouldNotFail(t *testing.T) {
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)
	ctx := context.Background()

	genesis := blk.NewGenesisBlock([]byte{})
	depChainStart := uint64(64)
	deposits, _, _ := testutil.DeterministicDepositsAndKeys(depChainStart)
	eth1Data, err := testutil.DeterministicEth1Data(len(deposits))
	if err != nil {
		t.Fatal(err)
	}
	bState, err := state.GenesisBeaconState(deposits, 0, eth1Data)
	if err != nil {
		t.Fatalf("Could not setup genesis state: %v", err)
	}
	bState.Slot = 5 // Set state to non-epoch start slot.

	genesisRoot, err := ssz.SigningRoot(genesis)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
	}

	var wg sync.WaitGroup
	numOfValidators := int(depChainStart)
	errs := make(chan error, numOfValidators)
	for i := 0; i < len(deposits); i++ {
		wg.Add(1)
		go func(index int) {
			errs <- db.SaveValidatorIndex(ctx, bytesutil.ToBytes48(deposits[index].Data.PublicKey), uint64(index))
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

	vs := &Server{
		BeaconDB:    db,
		HeadFetcher: &mockChain.ChainService{State: bState, Root: genesisRoot[:]},
		SyncChecker: &mockSync.Sync{IsSyncing: false},
	}

	// Test the first validator in registry.
	req := &pb.AssignmentRequest{
		PublicKeys: [][]byte{deposits[0].Data.PublicKey},
		EpochStart: 0,
	}
	res, err := vs.CommitteeAssignment(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.ValidatorAssignment) != 1 {
		t.Error("Expected 1 assignment")
	}
}

func TestCommitteeAssignment_MultipleKeys_OK(t *testing.T) {
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)
	ctx := context.Background()

	genesis := blk.NewGenesisBlock([]byte{})
	depChainStart := uint64(64)
	deposits, _, _ := testutil.DeterministicDepositsAndKeys(depChainStart)
	eth1Data, err := testutil.DeterministicEth1Data(len(deposits))
	if err != nil {
		t.Fatal(err)
	}
	state, err := state.GenesisBeaconState(deposits, 0, eth1Data)
	if err != nil {
		t.Fatalf("Could not setup genesis state: %v", err)
	}
	genesisRoot, err := ssz.SigningRoot(genesis)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
	}

	var wg sync.WaitGroup
	numOfValidators := int(depChainStart)
	errs := make(chan error, numOfValidators)
	for i := 0; i < numOfValidators; i++ {
		wg.Add(1)
		go func(index int) {
			errs <- db.SaveValidatorIndex(ctx, bytesutil.ToBytes48(deposits[index].Data.PublicKey), uint64(index))
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

	vs := &Server{
		BeaconDB:    db,
		HeadFetcher: &mockChain.ChainService{State: state, Root: genesisRoot[:]},
		SyncChecker: &mockSync.Sync{IsSyncing: false},
	}

	pubkey0 := deposits[0].Data.PublicKey
	pubkey1 := deposits[1].Data.PublicKey

	// Test the first validator in registry.
	req := &pb.AssignmentRequest{
		PublicKeys: [][]byte{pubkey0, pubkey1},
		EpochStart: 0,
	}
	res, err := vs.CommitteeAssignment(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not call epoch committee assignment %v", err)
	}

	if len(res.ValidatorAssignment) != 2 {
		t.Errorf("expected 2 assignments but got %d", len(res.ValidatorAssignment))
	}
	if res.ValidatorAssignment[0].AttesterSlot != 4 {
		t.Errorf("Expected res.ValidatorAssignment[0].AttesterSlot == 4, got %d", res.ValidatorAssignment[0].AttesterSlot)
	}
	if res.ValidatorAssignment[1].AttesterSlot != 3 {
		t.Errorf("Expected res.ValidatorAssignment[1].AttesterSlot == 3, got %d", res.ValidatorAssignment[0].AttesterSlot)
	}
}

func TestCommitteeAssignment_SyncNotReady(t *testing.T) {
	vs := &Server{
		SyncChecker: &mockSync.Sync{IsSyncing: true},
	}
	_, err := vs.CommitteeAssignment(context.Background(), &pb.AssignmentRequest{})
	if strings.Contains(err.Error(), "syncing to latest head") {
		t.Error("Did not get wanted error")
	}
}

func BenchmarkCommitteeAssignment(b *testing.B) {
	db := dbutil.SetupDB(b)
	defer dbutil.TeardownDB(b, db)
	ctx := context.Background()

	genesis := blk.NewGenesisBlock([]byte{})
	depChainStart := uint64(8192*2)
	deposits, _, _ := testutil.DeterministicDepositsAndKeys(depChainStart)
	eth1Data, err := testutil.DeterministicEth1Data(len(deposits))
	if err != nil {
		b.Fatal(err)
	}
	state, err := state.GenesisBeaconState(deposits, 0, eth1Data)
	if err != nil {
		b.Fatalf("Could not setup genesis state: %v", err)
	}
	genesisRoot, err := ssz.SigningRoot(genesis)
	if err != nil {
		b.Fatalf("Could not get signing root %v", err)
	}

	var wg sync.WaitGroup
	numOfValidators := int(depChainStart)
	errs := make(chan error, numOfValidators)
	for i := 0; i < numOfValidators; i++ {
		wg.Add(1)
		go func(index int) {
			errs <- db.SaveValidatorIndex(ctx, bytesutil.ToBytes48(deposits[index].Data.PublicKey), uint64(index))
			wg.Done()
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			b.Fatalf("Could not save validator index: %v", err)
		}
	}

	vs := &Server{
		BeaconDB:    db,
		HeadFetcher: &mockChain.ChainService{State: state, Root: genesisRoot[:]},
		SyncChecker: &mockSync.Sync{IsSyncing: false},
	}

	// Create request for all validators in the system.
	pks := make([][]byte, len(deposits))
	for i, deposit := range deposits {
		pks[i] = deposit.Data.PublicKey
	}
	req := &pb.AssignmentRequest{
		PublicKeys: pks,
		EpochStart: 0,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := vs.CommitteeAssignment(context.Background(), req)
		if err != nil {
			b.Error(err)
		}
	}
}
