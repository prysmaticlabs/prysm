package validator

import (
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	blk "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

// pubKey is a helper to generate a well-formed public key.
func pubKey(i uint64) []byte {
	pubKey := make([]byte, params.BeaconConfig().BLSPubkeyLength)
	binary.LittleEndian.PutUint64(pubKey, uint64(i))
	return pubKey
}
func TestGetDuties_NextEpoch_CantFindValidatorIdx(t *testing.T) {
	db := dbutil.SetupDB(t)
	ctx := context.Background()
	beaconState, _ := testutil.DeterministicGenesisState(t, 10)

	genesis := blk.NewGenesisBlock([]byte{})
	genesisRoot, err := stateutil.BlockRoot(genesis.Block)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
	}

	height := time.Unix(int64(params.BeaconConfig().Eth1FollowDistance), 0).Unix()
	p := &mockPOW.POWChain{
		TimesByHeight: map[int]uint64{
			0: uint64(height),
		},
	}

	vs := &Server{
		BeaconDB:        db,
		HeadFetcher:     &mockChain.ChainService{State: beaconState, Root: genesisRoot[:]},
		SyncChecker:     &mockSync.Sync{IsSyncing: false},
		Eth1InfoFetcher: p,
		DepositFetcher:  depositcache.NewDepositCache(),
	}

	pubKey := pubKey(99999)
	req := &ethpb.DutiesRequest{
		PublicKeys: [][]byte{pubKey},
		Epoch:      0,
	}
	want := fmt.Sprintf("validator %#x does not exist", req.PublicKeys[0])
	if _, err := vs.GetDuties(ctx, req); err != nil && !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestGetDuties_OK(t *testing.T) {
	db := dbutil.SetupDB(t)

	genesis := blk.NewGenesisBlock([]byte{})
	depChainStart := params.BeaconConfig().MinGenesisActiveValidatorCount
	deposits, _, err := testutil.DeterministicDepositsAndKeys(depChainStart)
	if err != nil {
		t.Fatal(err)
	}
	eth1Data, err := testutil.DeterministicEth1Data(len(deposits))
	if err != nil {
		t.Fatal(err)
	}
	bs, err := state.GenesisBeaconState(deposits, 0, eth1Data)
	if err != nil {
		t.Fatalf("Could not setup genesis bs: %v", err)
	}
	genesisRoot, err := stateutil.BlockRoot(genesis.Block)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
	}

	pubKeys := make([][]byte, len(deposits))
	indices := make([]uint64, len(deposits))
	for i := 0; i < len(deposits); i++ {
		pubKeys[i] = deposits[i].Data.PublicKey
		indices[i] = uint64(i)
	}

	pubkeysAs48ByteType := make([][48]byte, len(pubKeys))
	for i, pk := range pubKeys {
		pubkeysAs48ByteType[i] = bytesutil.ToBytes48(pk)
	}

	vs := &Server{
		BeaconDB:    db,
		HeadFetcher: &mockChain.ChainService{State: bs, Root: genesisRoot[:]},
		SyncChecker: &mockSync.Sync{IsSyncing: false},
	}

	// Test the first validator in registry.
	req := &ethpb.DutiesRequest{
		PublicKeys: [][]byte{deposits[0].Data.PublicKey},
		Epoch:      0,
	}
	res, err := vs.GetDuties(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not call epoch committee assignment %v", err)
	}
	if res.Duties[0].AttesterSlot > bs.Slot()+params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Assigned slot %d can't be higher than %d",
			res.Duties[0].AttesterSlot, bs.Slot()+params.BeaconConfig().SlotsPerEpoch)
	}

	// Test the last validator in registry.
	lastValidatorIndex := depChainStart - 1
	req = &ethpb.DutiesRequest{
		PublicKeys: [][]byte{deposits[lastValidatorIndex].Data.PublicKey},
		Epoch:      0,
	}
	res, err = vs.GetDuties(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not call epoch committee assignment %v", err)
	}
	if res.Duties[0].AttesterSlot > bs.Slot()+params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Assigned slot %d can't be higher than %d",
			res.Duties[0].AttesterSlot, bs.Slot()+params.BeaconConfig().SlotsPerEpoch)
	}

	// We request for duties for all validators.
	req = &ethpb.DutiesRequest{
		PublicKeys: pubKeys,
		Epoch:      0,
	}
	res, err = vs.GetDuties(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not call epoch committee assignment %v", err)
	}
	for i := 0; i < len(res.Duties); i++ {
		if res.Duties[i].ValidatorIndex != uint64(i) {
			t.Errorf("Wanted %d, received %d", i, res.Duties[i].ValidatorIndex)
		}
	}
}

func TestGetDuties_CurrentEpoch_ShouldNotFail(t *testing.T) {
	db := dbutil.SetupDB(t)

	genesis := blk.NewGenesisBlock([]byte{})
	depChainStart := params.BeaconConfig().MinGenesisActiveValidatorCount
	deposits, _, err := testutil.DeterministicDepositsAndKeys(depChainStart)
	if err != nil {
		t.Fatal(err)
	}
	eth1Data, err := testutil.DeterministicEth1Data(len(deposits))
	if err != nil {
		t.Fatal(err)
	}
	bState, err := state.GenesisBeaconState(deposits, 0, eth1Data)
	if err != nil {
		t.Fatalf("Could not setup genesis state: %v", err)
	}
	// Set state to non-epoch start slot.
	if err := bState.SetSlot(5); err != nil {
		t.Fatal(err)
	}

	genesisRoot, err := stateutil.BlockRoot(genesis.Block)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
	}

	pubKeys := make([][48]byte, len(deposits))
	indices := make([]uint64, len(deposits))
	for i := 0; i < len(deposits); i++ {
		pubKeys[i] = bytesutil.ToBytes48(deposits[i].Data.PublicKey)
		indices[i] = uint64(i)
	}

	vs := &Server{
		BeaconDB:    db,
		HeadFetcher: &mockChain.ChainService{State: bState, Root: genesisRoot[:]},
		SyncChecker: &mockSync.Sync{IsSyncing: false},
	}

	// Test the first validator in registry.
	req := &ethpb.DutiesRequest{
		PublicKeys: [][]byte{deposits[0].Data.PublicKey},
		Epoch:      0,
	}
	res, err := vs.GetDuties(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Duties) != 1 {
		t.Error("Expected 1 assignment")
	}
}

func TestGetDuties_MultipleKeys_OK(t *testing.T) {
	db := dbutil.SetupDB(t)

	genesis := blk.NewGenesisBlock([]byte{})
	depChainStart := uint64(64)
	testutil.ResetCache()
	deposits, _, err := testutil.DeterministicDepositsAndKeys(depChainStart)
	if err != nil {
		t.Fatal(err)
	}
	eth1Data, err := testutil.DeterministicEth1Data(len(deposits))
	if err != nil {
		t.Fatal(err)
	}
	bs, err := state.GenesisBeaconState(deposits, 0, eth1Data)
	if err != nil {
		t.Fatalf("Could not setup genesis bs: %v", err)
	}
	genesisRoot, err := stateutil.BlockRoot(genesis.Block)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
	}

	pubKeys := make([][48]byte, len(deposits))
	indices := make([]uint64, len(deposits))
	for i := 0; i < len(deposits); i++ {
		pubKeys[i] = bytesutil.ToBytes48(deposits[i].Data.PublicKey)
		indices[i] = uint64(i)
	}

	vs := &Server{
		BeaconDB:    db,
		HeadFetcher: &mockChain.ChainService{State: bs, Root: genesisRoot[:]},
		SyncChecker: &mockSync.Sync{IsSyncing: false},
	}

	pubkey0 := deposits[0].Data.PublicKey
	pubkey1 := deposits[1].Data.PublicKey

	// Test the first validator in registry.
	req := &ethpb.DutiesRequest{
		PublicKeys: [][]byte{pubkey0, pubkey1},
		Epoch:      0,
	}
	res, err := vs.GetDuties(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not call epoch committee assignment %v", err)
	}

	if len(res.Duties) != 2 {
		t.Errorf("expected 2 assignments but got %d", len(res.Duties))
	}
	if res.Duties[0].AttesterSlot != 2 {
		t.Errorf("Expected res.Duties[0].AttesterSlot == 7, got %d", res.Duties[0].AttesterSlot)
	}
	if res.Duties[1].AttesterSlot != 1 {
		t.Errorf("Expected res.Duties[1].AttesterSlot == 1, got %d", res.Duties[1].AttesterSlot)
	}
}

func TestGetDuties_SyncNotReady(t *testing.T) {
	vs := &Server{
		SyncChecker: &mockSync.Sync{IsSyncing: true},
	}
	_, err := vs.GetDuties(context.Background(), &ethpb.DutiesRequest{})
	if err == nil || strings.Contains(err.Error(), "syncing to latest head") {
		t.Error("Did not get wanted error")
	}
}

func TestAssignValidatorToSubnet(t *testing.T) {
	k := pubKey(3)

	assignValidatorToSubnet(k, ethpb.ValidatorStatus_ACTIVE)
	coms, ok, exp := cache.CommitteeIDs.GetPersistentCommittees(k)
	if !ok {
		t.Fatal("No cache entry found for validator")
	}
	if uint64(len(coms)) != params.BeaconNetworkConfig().RandomSubnetsPerValidator {
		t.Errorf("Only %d committees subscribed when %d was needed.", len(coms), params.BeaconNetworkConfig().RandomSubnetsPerValidator)
	}
	epochDuration := time.Duration(params.BeaconConfig().SlotsPerEpoch * params.BeaconConfig().SecondsPerSlot)
	totalTime := time.Duration(params.BeaconNetworkConfig().EpochsPerRandomSubnetSubscription) * epochDuration * time.Second
	receivedTime := exp.Round(time.Second).Sub(time.Now())
	if receivedTime < totalTime {
		t.Fatalf("Expiration time of %f was less than expected duration of %f ", receivedTime.Seconds(), totalTime.Seconds())
	}

}

func BenchmarkCommitteeAssignment(b *testing.B) {
	db := dbutil.SetupDB(b)

	genesis := blk.NewGenesisBlock([]byte{})
	depChainStart := uint64(8192 * 2)
	deposits, _, err := testutil.DeterministicDepositsAndKeys(depChainStart)
	if err != nil {
		b.Fatal(err)
	}
	eth1Data, err := testutil.DeterministicEth1Data(len(deposits))
	if err != nil {
		b.Fatal(err)
	}
	bs, err := state.GenesisBeaconState(deposits, 0, eth1Data)
	if err != nil {
		b.Fatalf("Could not setup genesis bs: %v", err)
	}
	genesisRoot, err := stateutil.BlockRoot(genesis.Block)
	if err != nil {
		b.Fatalf("Could not get signing root %v", err)
	}

	pubKeys := make([][48]byte, len(deposits))
	indices := make([]uint64, len(deposits))
	for i := 0; i < len(deposits); i++ {
		pubKeys[i] = bytesutil.ToBytes48(deposits[i].Data.PublicKey)
		indices[i] = uint64(i)
	}

	vs := &Server{
		BeaconDB:    db,
		HeadFetcher: &mockChain.ChainService{State: bs, Root: genesisRoot[:]},
		SyncChecker: &mockSync.Sync{IsSyncing: false},
	}

	// Create request for all validators in the system.
	pks := make([][]byte, len(deposits))
	for i, deposit := range deposits {
		pks[i] = deposit.Data.PublicKey
	}
	req := &ethpb.DutiesRequest{
		PublicKeys: pks,
		Epoch:      0,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := vs.GetDuties(context.Background(), req)
		if err != nil {
			b.Error(err)
		}
	}
}
