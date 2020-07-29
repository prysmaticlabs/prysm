package validator

import (
	"context"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
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
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
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
	require.NoError(t, db.SaveBlock(ctx, head))
	root, err := stateutil.BlockRoot(head.Block)
	require.NoError(t, err)

	validators := make([]*ethpb.Validator, 64)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
		}
	}

	state := testutil.NewBeaconState()
	require.NoError(t, state.SetSlot(params.BeaconConfig().SlotsPerEpoch+1))
	require.NoError(t, state.SetValidators(validators))
	require.NoError(t, db.SaveState(ctx, state, root))
	require.NoError(t, db.SaveHeadBlockRoot(ctx, root))

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
	_, err = attesterServer.ProposeAttestation(context.Background(), req)
	assert.NoError(t, err)
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
	_, err := attesterServer.ProposeAttestation(context.Background(), req)
	assert.ErrorContains(t, wanted, err)
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
	require.NoError(t, err, "Could not hash beacon block")
	justifiedRoot, err := stateutil.BlockRoot(justifiedBlock)
	require.NoError(t, err, "Could not get signing root for justified block")
	targetRoot, err := stateutil.BlockRoot(targetBlock)
	require.NoError(t, err, "Could not get signing root for target block")
	slot := 3*params.BeaconConfig().SlotsPerEpoch + 1
	beaconState := testutil.NewBeaconState()
	require.NoError(t, beaconState.SetSlot(slot))
	err = beaconState.SetCurrentJustifiedCheckpoint(&ethpb.Checkpoint{
		Epoch: 2,
		Root:  justifiedRoot[:],
	})
	require.NoError(t, err)

	blockRoots := beaconState.BlockRoots()
	blockRoots[1] = blockRoot[:]
	blockRoots[1*params.BeaconConfig().SlotsPerEpoch] = targetRoot[:]
	blockRoots[2*params.BeaconConfig().SlotsPerEpoch] = justifiedRoot[:]
	require.NoError(t, beaconState.SetBlockRoots(blockRoots))
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
	require.NoError(t, db.SaveState(ctx, beaconState, blockRoot))
	require.NoError(t, db.SaveBlock(ctx, &ethpb.SignedBeaconBlock{Block: block}))
	require.NoError(t, db.SaveHeadBlockRoot(ctx, blockRoot))

	req := &ethpb.AttestationDataRequest{
		CommitteeIndex: 0,
		Slot:           3*params.BeaconConfig().SlotsPerEpoch + 1,
	}
	res, err := attesterServer.GetAttestationData(context.Background(), req)
	require.NoError(t, err, "Could not get attestation info at slot")

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
	assert.ErrorContains(t, "Syncing to latest head", err)
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
	require.NoError(t, err, "Could not hash beacon block")
	justifiedBlockRoot, err := stateutil.BlockRoot(justifiedBlock)
	require.NoError(t, err, "Could not hash justified block")
	epochBoundaryRoot, err := stateutil.BlockRoot(epochBoundaryBlock)
	require.NoError(t, err, "Could not hash justified block")
	slot := uint64(10000)

	beaconState := testutil.NewBeaconState()
	require.NoError(t, beaconState.SetSlot(slot))
	err = beaconState.SetCurrentJustifiedCheckpoint(&ethpb.Checkpoint{
		Epoch: helpers.SlotToEpoch(1500),
		Root:  justifiedBlockRoot[:],
	})
	require.NoError(t, err)
	blockRoots := beaconState.BlockRoots()
	blockRoots[1] = blockRoot[:]
	blockRoots[1*params.BeaconConfig().SlotsPerEpoch] = epochBoundaryRoot[:]
	blockRoots[2*params.BeaconConfig().SlotsPerEpoch] = justifiedBlockRoot[:]
	require.NoError(t, beaconState.SetBlockRoots(blockRoots))
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
	require.NoError(t, db.SaveState(ctx, beaconState, blockRoot))
	require.NoError(t, db.SaveBlock(ctx, &ethpb.SignedBeaconBlock{Block: block}))
	require.NoError(t, db.SaveHeadBlockRoot(ctx, blockRoot))

	req := &ethpb.AttestationDataRequest{
		CommitteeIndex: 0,
		Slot:           10000,
	}
	res, err := attesterServer.GetAttestationData(context.Background(), req)
	require.NoError(t, err, "Could not get attestation info at slot")

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
	require.NoError(t, err)
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

	require.NoError(t, server.AttestationCache.MarkInProgress(req))

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		response, err := server.GetAttestationData(ctx, req)
		require.NoError(t, err)
		if !proto.Equal(res, response) {
			t.Error("Expected  equal responses from cache")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		assert.NoError(t, server.AttestationCache.Put(ctx, req, res))
		assert.NoError(t, server.AttestationCache.MarkNotInProgress(req))
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
	assert.ErrorContains(t, "invalid request", err)
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
	require.NoError(t, err, "Could not hash beacon block")
	blockRoot2, err := block2.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, &ethpb.SignedBeaconBlock{Block: block2}))
	justifiedRoot, err := stateutil.BlockRoot(justifiedBlock)
	require.NoError(t, err, "Could not get signing root for justified block")
	targetRoot, err := stateutil.BlockRoot(targetBlock)
	require.NoError(t, err, "Could not get signing root for target block")

	beaconState := testutil.NewBeaconState()
	require.NoError(t, beaconState.SetSlot(slot))
	require.NoError(t, beaconState.SetGenesisTime(uint64(time.Now().Unix()-int64(slot*params.BeaconConfig().SecondsPerSlot))))
	err = beaconState.SetLatestBlockHeader(&ethpb.BeaconBlockHeader{
		ParentRoot: blockRoot2[:],
		StateRoot:  make([]byte, 32),
		BodyRoot:   make([]byte, 32),
	})
	require.NoError(t, err)
	err = beaconState.SetCurrentJustifiedCheckpoint(&ethpb.Checkpoint{
		Epoch: 2,
		Root:  justifiedRoot[:],
	})
	require.NoError(t, err)
	blockRoots := beaconState.BlockRoots()
	blockRoots[1] = blockRoot[:]
	blockRoots[1*params.BeaconConfig().SlotsPerEpoch] = targetRoot[:]
	blockRoots[2*params.BeaconConfig().SlotsPerEpoch] = justifiedRoot[:]
	blockRoots[3*params.BeaconConfig().SlotsPerEpoch] = blockRoot2[:]
	require.NoError(t, beaconState.SetBlockRoots(blockRoots))

	beaconState2 := beaconState.Copy()
	require.NoError(t, beaconState2.SetSlot(beaconState2.Slot()-1))
	require.NoError(t, db.SaveState(ctx, beaconState2, blockRoot2))
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
	require.NoError(t, db.SaveState(ctx, beaconState, blockRoot))
	require.NoError(t, db.SaveBlock(ctx, &ethpb.SignedBeaconBlock{Block: block}))
	require.NoError(t, db.SaveHeadBlockRoot(ctx, blockRoot))

	req := &ethpb.AttestationDataRequest{
		CommitteeIndex: 0,
		Slot:           slot - 1,
	}
	res, err := attesterServer.GetAttestationData(ctx, req)
	require.NoError(t, err, "Could not get attestation info at slot")

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
	require.NoError(t, err, "Could not hash beacon block")
	justifiedRoot, err := stateutil.BlockRoot(justifiedBlock)
	require.NoError(t, err, "Could not get signing root for justified block")
	targetRoot, err := stateutil.BlockRoot(targetBlock)
	require.NoError(t, err, "Could not get signing root for target block")

	beaconState := testutil.NewBeaconState()
	require.NoError(t, beaconState.SetSlot(slot))
	err = beaconState.SetCurrentJustifiedCheckpoint(&ethpb.Checkpoint{
		Epoch: 0,
		Root:  justifiedRoot[:],
	})
	require.NoError(t, err)
	blockRoots := beaconState.BlockRoots()
	blockRoots[1] = blockRoot[:]
	blockRoots[1*params.BeaconConfig().SlotsPerEpoch] = targetRoot[:]
	blockRoots[2*params.BeaconConfig().SlotsPerEpoch] = justifiedRoot[:]
	require.NoError(t, beaconState.SetBlockRoots(blockRoots))
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
	require.NoError(t, db.SaveState(ctx, beaconState, blockRoot))
	require.NoError(t, db.SaveBlock(ctx, &ethpb.SignedBeaconBlock{Block: block}))
	require.NoError(t, db.SaveHeadBlockRoot(ctx, blockRoot))

	req := &ethpb.AttestationDataRequest{
		CommitteeIndex: 0,
		Slot:           5,
	}
	res, err := attesterServer.GetAttestationData(context.Background(), req)
	require.NoError(t, err, "Could not get attestation info at slot")

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
	assert.ErrorContains(t, "no attester slots provided", err)
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
	assert.ErrorContains(t, "request fields are not the same length", err)
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
	require.NoError(t, state.SetValidators(validators))

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
	require.NoError(t, err)
	for i := uint64(100); i < 200; i++ {
		subnets := cache.SubnetIDs.GetAttesterSubnetIDs(i)
		assert.Equal(t, 1, len(subnets))
		if isAggregator[i-100] {
			subnets = cache.SubnetIDs.GetAggregatorSubnetIDs(i)
			assert.Equal(t, 1, len(subnets))
		}
	}
}
