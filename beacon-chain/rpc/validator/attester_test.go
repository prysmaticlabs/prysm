package validator

import (
	"context"
	"math/rand"
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
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"google.golang.org/grpc/status"
)

func TestProposeAttestation_OK(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()

	attesterServer := &Server{
		HeadFetcher:       &mock.ChainService{},
		P2P:               &mockp2p.MockBroadcaster{},
		BeaconDB:          db,
		AttestationCache:  cache.NewAttestationCache(),
		AttPool:           attestations.NewPool(),
		OperationNotifier: (&mock.ChainService{}).OperationNotifier(),
	}
	head := testutil.NewBeaconBlock()
	head.Block.Slot = 999
	head.Block.ParentRoot = bytesutil.PadTo([]byte{'a'}, 32)
	if err := db.SaveBlock(ctx, head); err != nil {
		t.Fatal(err)
	}
	root, err := stateutil.BlockRoot(head.Block)
	if err != nil {
		t.Fatal(err)
	}

	validators := make([]*ethpb.Validator, 64)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
		}
	}

	state := testutil.NewBeaconState()
	if err := state.SetSlot(params.BeaconConfig().SlotsPerEpoch + 1); err != nil {
		t.Fatal(err)
	}
	if err := state.SetValidators(validators); err != nil {
		t.Fatal(err)
	}

	if err := db.SaveState(ctx, state, root); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveHeadBlockRoot(ctx, root); err != nil {
		t.Fatal(err)
	}

	sk := bls.RandKey()
	sig := sk.Sign([]byte("dummy_test_data"))
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
	db, _ := dbutil.SetupDB(t)

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
	if _, err := attesterServer.ProposeAttestation(context.Background(), req); err == nil || !strings.Contains(err.Error(), wanted) {
		t.Errorf("Did not get wanted error")
	}
}

func TestGetAttestationData_OK(t *testing.T) {
	ctx := context.Background()
	db, _ := dbutil.SetupDB(t)

	block := &ethpb.BeaconBlock{
		Slot: 3*params.BeaconConfig().SlotsPerEpoch + 1,
	}
	targetBlock := &ethpb.BeaconBlock{
		Slot: 1 * params.BeaconConfig().SlotsPerEpoch,
	}
	justifiedBlock := &ethpb.BeaconBlock{
		Slot: 2 * params.BeaconConfig().SlotsPerEpoch,
	}
	blockRoot, err := stateutil.BlockRoot(block)
	if err != nil {
		t.Fatalf("Could not hash beacon block: %v", err)
	}
	justifiedRoot, err := stateutil.BlockRoot(justifiedBlock)
	if err != nil {
		t.Fatalf("Could not get signing root for justified block: %v", err)
	}
	targetRoot, err := stateutil.BlockRoot(targetBlock)
	if err != nil {
		t.Fatalf("Could not get signing root for target block: %v", err)
	}
	slot := 3*params.BeaconConfig().SlotsPerEpoch + 1
	beaconState := testutil.NewBeaconState()
	if err := beaconState.SetSlot(slot); err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetCurrentJustifiedCheckpoint(&ethpb.Checkpoint{
		Epoch: 2,
		Root:  justifiedRoot[:],
	}); err != nil {
		t.Fatal(err)
	}

	blockRoots := beaconState.BlockRoots()
	blockRoots[1] = blockRoot[:]
	blockRoots[1*params.BeaconConfig().SlotsPerEpoch] = targetRoot[:]
	blockRoots[2*params.BeaconConfig().SlotsPerEpoch] = justifiedRoot[:]
	if err := beaconState.SetBlockRoots(blockRoots); err != nil {
		t.Fatal(err)
	}
	chainService := &mock.ChainService{
		Genesis: time.Now(),
	}
	attesterServer := &Server{
		BeaconDB:         db,
		P2P:              &mockp2p.MockBroadcaster{},
		SyncChecker:      &mockSync.Sync{IsSyncing: false},
		AttestationCache: cache.NewAttestationCache(),
		HeadFetcher: &mock.ChainService{
			State: beaconState, Root: blockRoot[:],
		},
		FinalizationFetcher: &mock.ChainService{
			CurrentJustifiedCheckPoint: beaconState.CurrentJustifiedCheckpoint(),
		},
		GenesisTimeFetcher: &mock.ChainService{
			Genesis: time.Now().Add(time.Duration(-1*int64(slot*params.BeaconConfig().SecondsPerSlot)) * time.Second),
		},
		StateNotifier: chainService.StateNotifier(),
	}
	if err := db.SaveState(ctx, beaconState, blockRoot); err != nil {
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
	if err == nil || strings.Contains(err.Error(), "syncing to latest head") {
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
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()
	// Ensure HistoricalRootsLimit matches scenario
	params.SetupTestConfigCleanup(t)
	cfg := params.MainnetConfig()
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
	blockRoot, err := stateutil.BlockRoot(block)
	if err != nil {
		t.Fatalf("Could not hash beacon block: %v", err)
	}
	justifiedBlockRoot, err := stateutil.BlockRoot(justifiedBlock)
	if err != nil {
		t.Fatalf("Could not hash justified block: %v", err)
	}
	epochBoundaryRoot, err := stateutil.BlockRoot(epochBoundaryBlock)
	if err != nil {
		t.Fatalf("Could not hash justified block: %v", err)
	}
	slot := uint64(10000)

	beaconState := testutil.NewBeaconState()
	if err := beaconState.SetSlot(slot); err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetCurrentJustifiedCheckpoint(&ethpb.Checkpoint{
		Epoch: helpers.SlotToEpoch(1500),
		Root:  justifiedBlockRoot[:],
	}); err != nil {
		t.Fatal(err)
	}
	blockRoots := beaconState.BlockRoots()
	blockRoots[1] = blockRoot[:]
	blockRoots[1*params.BeaconConfig().SlotsPerEpoch] = epochBoundaryRoot[:]
	blockRoots[2*params.BeaconConfig().SlotsPerEpoch] = justifiedBlockRoot[:]
	if err := beaconState.SetBlockRoots(blockRoots); err != nil {
		t.Fatal(err)
	}
	chainService := &mock.ChainService{
		Genesis: time.Now(),
	}
	attesterServer := &Server{
		BeaconDB:         db,
		P2P:              &mockp2p.MockBroadcaster{},
		AttestationCache: cache.NewAttestationCache(),
		HeadFetcher:      &mock.ChainService{State: beaconState, Root: blockRoot[:]},
		FinalizationFetcher: &mock.ChainService{
			CurrentJustifiedCheckPoint: beaconState.CurrentJustifiedCheckpoint(),
		},
		SyncChecker:        &mockSync.Sync{IsSyncing: false},
		GenesisTimeFetcher: &mock.ChainService{Genesis: time.Now().Add(time.Duration(-1*int64(slot*params.BeaconConfig().SecondsPerSlot)) * time.Second)},
		StateNotifier:      chainService.StateNotifier(),
	}
	if err := db.SaveState(ctx, beaconState, blockRoot); err != nil {
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
	state, err := beaconstate.InitializeFromProto(s)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	chainService := &mock.ChainService{
		Genesis: time.Now(),
	}
	slot := uint64(2)
	server := &Server{
		HeadFetcher:        &mock.ChainService{State: state},
		AttestationCache:   cache.NewAttestationCache(),
		SyncChecker:        &mockSync.Sync{IsSyncing: false},
		GenesisTimeFetcher: &mock.ChainService{Genesis: time.Now().Add(time.Duration(-1*int64(slot*params.BeaconConfig().SecondsPerSlot)) * time.Second)},
		StateNotifier:      chainService.StateNotifier(),
	}

	req := &ethpb.AttestationDataRequest{
		CommitteeIndex: 1,
		Slot:           slot,
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

func TestServer_GetAttestationData_InvalidRequestSlot(t *testing.T) {
	ctx := context.Background()

	slot := 3*params.BeaconConfig().SlotsPerEpoch + 1
	attesterServer := &Server{
		SyncChecker:        &mockSync.Sync{IsSyncing: false},
		GenesisTimeFetcher: &mock.ChainService{Genesis: time.Now().Add(time.Duration(-1*int64(slot*params.BeaconConfig().SecondsPerSlot)) * time.Second)},
	}

	req := &ethpb.AttestationDataRequest{
		Slot: 1000000000000,
	}
	_, err := attesterServer.GetAttestationData(ctx, req)
	if s, ok := status.FromError(err); !ok || !strings.Contains(s.Message(), "invalid request") {
		t.Fatalf("Wrong error. Wanted error to start with %v, got %v", "invalid request", err)
	}
}

func TestServer_GetAttestationData_HeadStateSlotGreaterThanRequestSlot(t *testing.T) {
	// There exists a rare scenario where the validator may request an attestation for a slot less
	// than the head state's slot. The ETH2 spec constraints require that the block root the
	// attestation is referencing be less than or equal to the attestation data slot.
	// See: https://github.com/prysmaticlabs/prysm/issues/5164
	ctx := context.Background()
	db, sc := dbutil.SetupDB(t)

	slot := 3*params.BeaconConfig().SlotsPerEpoch + 1
	block := &ethpb.BeaconBlock{
		Slot: slot,
	}
	block2 := &ethpb.BeaconBlock{Slot: slot - 1}
	targetBlock := &ethpb.BeaconBlock{
		Slot: 1 * params.BeaconConfig().SlotsPerEpoch,
	}
	justifiedBlock := &ethpb.BeaconBlock{
		Slot: 2 * params.BeaconConfig().SlotsPerEpoch,
	}
	blockRoot, err := stateutil.BlockRoot(block)
	if err != nil {
		t.Fatalf("Could not hash beacon block: %v", err)
	}
	blockRoot2, err := ssz.HashTreeRoot(block2)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(ctx, &ethpb.SignedBeaconBlock{Block: block2}); err != nil {
		t.Fatal(err)
	}
	justifiedRoot, err := stateutil.BlockRoot(justifiedBlock)
	if err != nil {
		t.Fatalf("Could not get signing root for justified block: %v", err)
	}
	targetRoot, err := stateutil.BlockRoot(targetBlock)
	if err != nil {
		t.Fatalf("Could not get signing root for target block: %v", err)
	}

	beaconState := testutil.NewBeaconState()
	if err := beaconState.SetSlot(slot); err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetGenesisTime(uint64(time.Now().Unix() - int64(slot*params.BeaconConfig().SecondsPerSlot))); err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetLatestBlockHeader(&ethpb.BeaconBlockHeader{
		ParentRoot: blockRoot2[:],
		StateRoot:  make([]byte, 32),
		BodyRoot:   make([]byte, 32),
	}); err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetCurrentJustifiedCheckpoint(&ethpb.Checkpoint{
		Epoch: 2,
		Root:  justifiedRoot[:],
	}); err != nil {
		t.Fatal(err)
	}
	blockRoots := beaconState.BlockRoots()
	blockRoots[1] = blockRoot[:]
	blockRoots[1*params.BeaconConfig().SlotsPerEpoch] = targetRoot[:]
	blockRoots[2*params.BeaconConfig().SlotsPerEpoch] = justifiedRoot[:]
	blockRoots[3*params.BeaconConfig().SlotsPerEpoch] = blockRoot2[:]
	if err := beaconState.SetBlockRoots(blockRoots); err != nil {
		t.Fatal(err)
	}

	beaconState2 := beaconState.Copy()
	if err := beaconState2.SetSlot(beaconState2.Slot() - 1); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, beaconState2, blockRoot2); err != nil {
		t.Fatal(err)
	}
	chainService := &mock.ChainService{
		Genesis: time.Now(),
	}
	attesterServer := &Server{
		BeaconDB:            db,
		P2P:                 &mockp2p.MockBroadcaster{},
		SyncChecker:         &mockSync.Sync{IsSyncing: false},
		AttestationCache:    cache.NewAttestationCache(),
		HeadFetcher:         &mock.ChainService{State: beaconState, Root: blockRoot[:]},
		FinalizationFetcher: &mock.ChainService{CurrentJustifiedCheckPoint: beaconState.CurrentJustifiedCheckpoint()},
		GenesisTimeFetcher:  &mock.ChainService{Genesis: time.Now().Add(time.Duration(-1*int64(slot*params.BeaconConfig().SecondsPerSlot)) * time.Second)},
		StateNotifier:       chainService.StateNotifier(),
		StateGen:            stategen.New(db, sc),
	}
	if err := db.SaveState(ctx, beaconState, blockRoot); err != nil {
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
		Slot:           slot - 1,
	}
	res, err := attesterServer.GetAttestationData(ctx, req)
	if err != nil {
		t.Fatalf("Could not get attestation info at slot: %v", err)
	}

	expectedInfo := &ethpb.AttestationData{
		Slot:            slot - 1,
		BeaconBlockRoot: blockRoot2[:],
		Source: &ethpb.Checkpoint{
			Epoch: 2,
			Root:  justifiedRoot[:],
		},
		Target: &ethpb.Checkpoint{
			Epoch: 3,
			Root:  blockRoot2[:],
		},
	}

	if !proto.Equal(res, expectedInfo) {
		t.Errorf("Expected attestation info to match, received %v, wanted %v", res, expectedInfo)
	}
}

func TestGetAttestationData_SucceedsInFirstEpoch(t *testing.T) {
	ctx := context.Background()
	db, _ := dbutil.SetupDB(t)

	slot := uint64(5)
	block := &ethpb.BeaconBlock{
		Slot: slot,
	}
	targetBlock := &ethpb.BeaconBlock{
		Slot: 0,
	}
	justifiedBlock := &ethpb.BeaconBlock{
		Slot: 0,
	}
	blockRoot, err := stateutil.BlockRoot(block)
	if err != nil {
		t.Fatalf("Could not hash beacon block: %v", err)
	}
	justifiedRoot, err := stateutil.BlockRoot(justifiedBlock)
	if err != nil {
		t.Fatalf("Could not get signing root for justified block: %v", err)
	}
	targetRoot, err := stateutil.BlockRoot(targetBlock)
	if err != nil {
		t.Fatalf("Could not get signing root for target block: %v", err)
	}

	beaconState := testutil.NewBeaconState()
	if err := beaconState.SetSlot(slot); err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetCurrentJustifiedCheckpoint(&ethpb.Checkpoint{
		Epoch: 0,
		Root:  justifiedRoot[:],
	}); err != nil {
		t.Fatal(err)
	}
	blockRoots := beaconState.BlockRoots()
	blockRoots[1] = blockRoot[:]
	blockRoots[1*params.BeaconConfig().SlotsPerEpoch] = targetRoot[:]
	blockRoots[2*params.BeaconConfig().SlotsPerEpoch] = justifiedRoot[:]
	if err := beaconState.SetBlockRoots(blockRoots); err != nil {
		t.Fatal(err)
	}
	chainService := &mock.ChainService{
		Genesis: time.Now(),
	}
	attesterServer := &Server{
		BeaconDB:         db,
		P2P:              &mockp2p.MockBroadcaster{},
		SyncChecker:      &mockSync.Sync{IsSyncing: false},
		AttestationCache: cache.NewAttestationCache(),
		HeadFetcher: &mock.ChainService{
			State: beaconState, Root: blockRoot[:],
		},
		FinalizationFetcher: &mock.ChainService{
			CurrentJustifiedCheckPoint: beaconState.CurrentJustifiedCheckpoint(),
		},
		GenesisTimeFetcher: &mock.ChainService{Genesis: roughtime.Now().Add(time.Duration(-1*int64(slot*params.BeaconConfig().SecondsPerSlot)) * time.Second)},
		StateNotifier:      chainService.StateNotifier(),
	}
	if err := db.SaveState(ctx, beaconState, blockRoot); err != nil {
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
		Slot:           5,
	}
	res, err := attesterServer.GetAttestationData(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not get attestation info at slot: %v", err)
	}

	expectedInfo := &ethpb.AttestationData{
		Slot:            slot,
		BeaconBlockRoot: blockRoot[:],
		Source: &ethpb.Checkpoint{
			Epoch: 0,
			Root:  justifiedRoot[:],
		},
		Target: &ethpb.Checkpoint{
			Epoch: 0,
			Root:  blockRoot[:],
		},
	}

	if !proto.Equal(res, expectedInfo) {
		t.Errorf("Expected attestation info to match, received %v, wanted %v", res, expectedInfo)
	}
}

func TestServer_SubscribeCommitteeSubnets_NoSlots(t *testing.T) {
	db, _ := dbutil.SetupDB(t)

	attesterServer := &Server{
		HeadFetcher:       &mock.ChainService{},
		P2P:               &mockp2p.MockBroadcaster{},
		BeaconDB:          db,
		AttestationCache:  cache.NewAttestationCache(),
		AttPool:           attestations.NewPool(),
		OperationNotifier: (&mock.ChainService{}).OperationNotifier(),
	}

	_, err := attesterServer.SubscribeCommitteeSubnets(context.Background(), &ethpb.CommitteeSubnetsSubscribeRequest{
		Slots:        nil,
		CommitteeIds: nil,
		IsAggregator: nil,
	})
	if err == nil || !strings.Contains(err.Error(), "no attester slots provided") {
		t.Fatalf("Expected no attester slots provided error, received: %v", err)
	}
}

func TestServer_SubscribeCommitteeSubnets_DifferentLengthSlots(t *testing.T) {
	db, _ := dbutil.SetupDB(t)

	// fixed seed
	s := rand.NewSource(10)
	randGen := rand.New(s)

	attesterServer := &Server{
		HeadFetcher:       &mock.ChainService{},
		P2P:               &mockp2p.MockBroadcaster{},
		BeaconDB:          db,
		AttestationCache:  cache.NewAttestationCache(),
		AttPool:           attestations.NewPool(),
		OperationNotifier: (&mock.ChainService{}).OperationNotifier(),
	}

	var slots []uint64
	var comIdxs []uint64
	var isAggregator []bool

	for i := uint64(100); i < 200; i++ {
		slots = append(slots, i)
		comIdxs = append(comIdxs, uint64(randGen.Int63n(64)))
		boolVal := randGen.Uint64()%2 == 0
		isAggregator = append(isAggregator, boolVal)
	}

	slots = append(slots, 321)

	_, err := attesterServer.SubscribeCommitteeSubnets(context.Background(), &ethpb.CommitteeSubnetsSubscribeRequest{
		Slots:        slots,
		CommitteeIds: comIdxs,
		IsAggregator: isAggregator,
	})
	if err == nil || !strings.Contains(err.Error(), "request fields are not the same length") {
		t.Fatalf("Expected request fields are not the same length error, received: %v", err)
	}
}

func TestServer_SubscribeCommitteeSubnets_MultipleSlots(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	// fixed seed
	s := rand.NewSource(10)
	randGen := rand.New(s)

	validators := make([]*ethpb.Validator, 64)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
		}
	}

	state := testutil.NewBeaconState()
	if err := state.SetValidators(validators); err != nil {
		t.Fatal(err)
	}

	attesterServer := &Server{
		HeadFetcher:       &mock.ChainService{State: state},
		P2P:               &mockp2p.MockBroadcaster{},
		BeaconDB:          db,
		AttestationCache:  cache.NewAttestationCache(),
		AttPool:           attestations.NewPool(),
		OperationNotifier: (&mock.ChainService{}).OperationNotifier(),
	}

	var slots []uint64
	var comIdxs []uint64
	var isAggregator []bool

	for i := uint64(100); i < 200; i++ {
		slots = append(slots, i)
		comIdxs = append(comIdxs, uint64(randGen.Int63n(64)))
		boolVal := randGen.Uint64()%2 == 0
		isAggregator = append(isAggregator, boolVal)
	}

	_, err := attesterServer.SubscribeCommitteeSubnets(context.Background(), &ethpb.CommitteeSubnetsSubscribeRequest{
		Slots:        slots,
		CommitteeIds: comIdxs,
		IsAggregator: isAggregator,
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := uint64(100); i < 200; i++ {
		subnets := cache.SubnetIDs.GetAttesterSubnetIDs(i)
		if len(subnets) != 1 {
			t.Errorf("Wanted subnets of length 1 but got %d", len(subnets))
		}
		if isAggregator[i-100] {
			subnets = cache.SubnetIDs.GetAggregatorSubnetIDs(i)
			if len(subnets) != 1 {
				t.Errorf("Wanted subnets of length 1 but got %d", len(subnets))
			}
		}
	}
}
