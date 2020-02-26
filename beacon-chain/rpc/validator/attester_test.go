package validator

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	mockp2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func init() {
	// Use minimal config to reduce test setup time.
	params.OverrideBeaconConfig(params.MinimalSpecConfig())
}

func TestProposeAttestation_OK(t *testing.T) {
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)
	ctx := context.Background()

	attesterServer := &Server{
		HeadFetcher:       &mock.ChainService{},
		P2P:               &mockp2p.MockBroadcaster{},
		BeaconDB:          db,
		AttestationCache:  cache.NewAttestationCache(),
		AttPool:           attestations.NewPool(),
		OperationNotifier: (&mock.ChainService{}).OperationNotifier(),
	}
	head := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot:       999,
			ParentRoot: []byte{'a'},
		},
	}
	if err := db.SaveBlock(ctx, head); err != nil {
		t.Fatal(err)
	}
	root, err := ssz.HashTreeRoot(head.Block)
	if err != nil {
		t.Fatal(err)
	}

	validators := make([]*ethpb.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount/16)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
		}
	}

	state, _ := beaconstate.InitializeFromProto(&pbp2p.BeaconState{
		Slot:        params.BeaconConfig().SlotsPerEpoch + 1,
		Validators:  validators,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})

	if err := db.SaveState(ctx, state, root); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveHeadBlockRoot(ctx, root); err != nil {
		t.Fatal(err)
	}

	sk := bls.RandKey()
	sig := sk.Sign([]byte("dummy_test_data"), 0 /*domain*/)
	req := &ethpb.Attestation{
		Signature: sig.Marshal(),
		Data: &ethpb.AttestationData{
			BeaconBlockRoot: root[:],
			Source:          &ethpb.Checkpoint{},
			Target:          &ethpb.Checkpoint{},
		},
	}
	if _, err := attesterServer.ProposeAttestation(context.Background(), req); err != nil {
		t.Errorf("Could not attest head correctly: %v", err)
	}
}

func TestProposeAttestation_IncorrectSignature(t *testing.T) {
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)

	attesterServer := &Server{
		HeadFetcher:       &mock.ChainService{},
		P2P:               &mockp2p.MockBroadcaster{},
		BeaconDB:          db,
		AttestationCache:  cache.NewAttestationCache(),
		AttPool:           attestations.NewPool(),
		OperationNotifier: (&mock.ChainService{}).OperationNotifier(),
	}

	req := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{},
			Target: &ethpb.Checkpoint{},
		},
	}
	wanted := "Incorrect attestation signature"
	if _, err := attesterServer.ProposeAttestation(context.Background(), req); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Did not get wanted error")
	}
}

func TestGetAttestationData_OK(t *testing.T) {
	ctx := context.Background()
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)

	block := &ethpb.BeaconBlock{
		Slot: 3*params.BeaconConfig().SlotsPerEpoch + 1,
	}
	targetBlock := &ethpb.BeaconBlock{
		Slot: 1 * params.BeaconConfig().SlotsPerEpoch,
	}
	justifiedBlock := &ethpb.BeaconBlock{
		Slot: 2 * params.BeaconConfig().SlotsPerEpoch,
	}
	blockRoot, err := ssz.HashTreeRoot(block)
	if err != nil {
		t.Fatalf("Could not hash beacon block: %v", err)
	}
	justifiedRoot, err := ssz.HashTreeRoot(justifiedBlock)
	if err != nil {
		t.Fatalf("Could not get signing root for justified block: %v", err)
	}
	targetRoot, err := ssz.HashTreeRoot(targetBlock)
	if err != nil {
		t.Fatalf("Could not get signing root for target block: %v", err)
	}

	beaconState := &pbp2p.BeaconState{
		Slot:       3*params.BeaconConfig().SlotsPerEpoch + 1,
		BlockRoots: make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot),
		CurrentJustifiedCheckpoint: &ethpb.Checkpoint{
			Epoch: 2,
			Root:  justifiedRoot[:],
		},
	}
	beaconState.BlockRoots[1] = blockRoot[:]
	beaconState.BlockRoots[1*params.BeaconConfig().SlotsPerEpoch] = targetRoot[:]
	beaconState.BlockRoots[2*params.BeaconConfig().SlotsPerEpoch] = justifiedRoot[:]
	s, _ := beaconstate.InitializeFromProto(beaconState)
	attesterServer := &Server{
		BeaconDB:            db,
		P2P:                 &mockp2p.MockBroadcaster{},
		SyncChecker:         &mockSync.Sync{IsSyncing: false},
		AttestationCache:    cache.NewAttestationCache(),
		HeadFetcher:         &mock.ChainService{State: s, Root: blockRoot[:]},
		FinalizationFetcher: &mock.ChainService{CurrentJustifiedCheckPoint: beaconState.CurrentJustifiedCheckpoint},
		GenesisTimeFetcher:  &mock.ChainService{},
	}
	if err := db.SaveState(ctx, s, blockRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(ctx, &ethpb.SignedBeaconBlock{Block: block}); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveHeadBlockRoot(ctx, blockRoot); err != nil {
		t.Fatal(err)
	}

	req := &ethpb.AttestationDataRequest{
		CommitteeIndex: 0,
		Slot:           3*params.BeaconConfig().SlotsPerEpoch + 1,
	}
	res, err := attesterServer.GetAttestationData(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not get attestation info at slot: %v", err)
	}

	expectedInfo := &ethpb.AttestationData{
		Slot:            3*params.BeaconConfig().SlotsPerEpoch + 1,
		BeaconBlockRoot: blockRoot[:],
		Source: &ethpb.Checkpoint{
			Epoch: 2,
			Root:  justifiedRoot[:],
		},
		Target: &ethpb.Checkpoint{
			Epoch: 3,
			Root:  blockRoot[:],
		},
	}

	if !proto.Equal(res, expectedInfo) {
		t.Errorf("Expected attestation info to match, received %v, wanted %v", res, expectedInfo)
	}
}

func TestGetAttestationData_SyncNotReady(t *testing.T) {
	as := &Server{
		SyncChecker: &mockSync.Sync{IsSyncing: true},
	}
	_, err := as.GetAttestationData(context.Background(), &ethpb.AttestationDataRequest{})
	if strings.Contains(err.Error(), "syncing to latest head") {
		t.Error("Did not get wanted error")
	}
}

func TestAttestationDataAtSlot_HandlesFarAwayJustifiedEpoch(t *testing.T) {
	// Scenario:
	//
	// State slot = 10000
	// Last justified slot = epoch start of 1500
	// HistoricalRootsLimit = 8192
	//
	// More background: https://github.com/prysmaticlabs/prysm/issues/2153
	// This test breaks if it doesnt use mainnet config
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)
	ctx := context.Background()
	params.OverrideBeaconConfig(params.MainnetConfig())
	defer params.OverrideBeaconConfig(params.MinimalSpecConfig())

	// Ensure HistoricalRootsLimit matches scenario
	cfg := params.BeaconConfig()
	cfg.HistoricalRootsLimit = 8192
	params.OverrideBeaconConfig(cfg)

	block := &ethpb.BeaconBlock{
		Slot: 10000,
	}
	epochBoundaryBlock := &ethpb.BeaconBlock{
		Slot: helpers.StartSlot(helpers.SlotToEpoch(10000)),
	}
	justifiedBlock := &ethpb.BeaconBlock{
		Slot: helpers.StartSlot(helpers.SlotToEpoch(1500)) - 2, // Imagine two skip block
	}
	blockRoot, err := ssz.HashTreeRoot(block)
	if err != nil {
		t.Fatalf("Could not hash beacon block: %v", err)
	}
	justifiedBlockRoot, err := ssz.HashTreeRoot(justifiedBlock)
	if err != nil {
		t.Fatalf("Could not hash justified block: %v", err)
	}
	epochBoundaryRoot, err := ssz.HashTreeRoot(epochBoundaryBlock)
	if err != nil {
		t.Fatalf("Could not hash justified block: %v", err)
	}
	beaconState := &pbp2p.BeaconState{
		Slot:       10000,
		BlockRoots: make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot),
		CurrentJustifiedCheckpoint: &ethpb.Checkpoint{
			Epoch: helpers.SlotToEpoch(1500),
			Root:  justifiedBlockRoot[:],
		},
	}
	beaconState.BlockRoots[1] = blockRoot[:]
	beaconState.BlockRoots[1*params.BeaconConfig().SlotsPerEpoch] = epochBoundaryRoot[:]
	beaconState.BlockRoots[2*params.BeaconConfig().SlotsPerEpoch] = justifiedBlockRoot[:]
	s, _ := beaconstate.InitializeFromProto(beaconState)
	attesterServer := &Server{
		BeaconDB:            db,
		P2P:                 &mockp2p.MockBroadcaster{},
		AttestationCache:    cache.NewAttestationCache(),
		HeadFetcher:         &mock.ChainService{State: s, Root: blockRoot[:]},
		FinalizationFetcher: &mock.ChainService{CurrentJustifiedCheckPoint: beaconState.CurrentJustifiedCheckpoint},
		SyncChecker:         &mockSync.Sync{IsSyncing: false},
		GenesisTimeFetcher:  &mock.ChainService{},
	}
	if err := db.SaveState(ctx, s, blockRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(ctx, &ethpb.SignedBeaconBlock{Block: block}); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveHeadBlockRoot(ctx, blockRoot); err != nil {
		t.Fatal(err)
	}

	req := &ethpb.AttestationDataRequest{
		CommitteeIndex: 0,
		Slot:           10000,
	}
	res, err := attesterServer.GetAttestationData(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not get attestation info at slot: %v", err)
	}

	expectedInfo := &ethpb.AttestationData{
		Slot:            req.Slot,
		BeaconBlockRoot: blockRoot[:],
		Source: &ethpb.Checkpoint{
			Epoch: helpers.SlotToEpoch(1500),
			Root:  justifiedBlockRoot[:],
		},
		Target: &ethpb.Checkpoint{
			Epoch: 312,
			Root:  blockRoot[:],
		},
	}

	if !proto.Equal(res, expectedInfo) {
		t.Errorf("Expected attestation info to match, received %v, wanted %v", res, expectedInfo)
	}
}

func TestAttestationDataSlot_handlesInProgressRequest(t *testing.T) {
	s := &pbp2p.BeaconState{Slot: 100}
	state, _ := beaconstate.InitializeFromProto(s)
	ctx := context.Background()
	chainService := &mock.ChainService{
		Genesis: time.Now(),
	}
	server := &Server{
		HeadFetcher:        &mock.ChainService{State: state},
		AttestationCache:   cache.NewAttestationCache(),
		SyncChecker:        &mockSync.Sync{IsSyncing: false},
		GenesisTimeFetcher: &mock.ChainService{},
		StateNotifier:      chainService.StateNotifier(),
	}

	req := &ethpb.AttestationDataRequest{
		CommitteeIndex: 1,
		Slot:           2,
	}

	res := &ethpb.AttestationData{
		Target: &ethpb.Checkpoint{Epoch: 55},
	}

	if err := server.AttestationCache.MarkInProgress(req); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		response, err := server.GetAttestationData(ctx, req)
		if err != nil {
			t.Error(err)
		}
		if !proto.Equal(res, response) {
			t.Error("Expected  equal responses from cache")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := server.AttestationCache.Put(ctx, req, res); err != nil {
			t.Error(err)
		}
		if err := server.AttestationCache.MarkNotInProgress(req); err != nil {
			t.Error(err)
		}
	}()

	wg.Wait()
}

func TestWaitForSlotOneThird_WaitedCorrectly(t *testing.T) {
	currentTime := uint64(time.Now().Unix())
	numOfSlots := uint64(4)
	genesisTime := currentTime - (numOfSlots * params.BeaconConfig().SecondsPerSlot)

	chainService := &mock.ChainService{
		Genesis: time.Now(),
	}
	server := &Server{
		AttestationCache:   cache.NewAttestationCache(),
		HeadFetcher:        &mock.ChainService{},
		SyncChecker:        &mockSync.Sync{IsSyncing: false},
		GenesisTimeFetcher: &mock.ChainService{Genesis: time.Unix(int64(genesisTime), 0)},
		StateNotifier:      chainService.StateNotifier(),
	}

	timeToSleep := params.BeaconConfig().SecondsPerSlot / 3
	oneThird := currentTime + timeToSleep
	server.waitToOneThird(context.Background(), numOfSlots)

	currentTime = uint64(time.Now().Unix())
	if currentTime != oneThird {
		t.Errorf("Wanted %d time for slot one third but got %d", oneThird, currentTime)
	}
}

func TestWaitForSlotOneThird_HeadIsHereNoWait(t *testing.T) {
	currentTime := uint64(time.Now().Unix())
	numOfSlots := uint64(4)
	genesisTime := currentTime - (numOfSlots * params.BeaconConfig().SecondsPerSlot)

	s := &pbp2p.BeaconState{Slot: 100}
	state, _ := beaconstate.InitializeFromProto(s)
	server := &Server{
		AttestationCache:   cache.NewAttestationCache(),
		HeadFetcher:        &mock.ChainService{State: state},
		SyncChecker:        &mockSync.Sync{IsSyncing: false},
		GenesisTimeFetcher: &mock.ChainService{Genesis: time.Unix(int64(genesisTime), 0)},
	}

	server.waitToOneThird(context.Background(), s.Slot)

	if currentTime != uint64(time.Now().Unix()) {
		t.Errorf("Wanted %d time for slot one third but got %d", uint64(time.Now().Unix()), currentTime)
	}
}
