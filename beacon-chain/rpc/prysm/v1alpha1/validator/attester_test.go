package validator

import (
	"context"
	"math/rand"
	"sync"
	"testing"
	"time"

	mock "github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/cache"
	dbutil "github.com/prysmaticlabs/prysm/v4/beacon-chain/db/testing"
	doublylinkedtree "github.com/prysmaticlabs/prysm/v4/beacon-chain/forkchoice/doubly-linked-tree"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/attestations"
	mockp2p "github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/core"
	state_native "github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/stategen"
	mockSync "github.com/prysmaticlabs/prysm/v4/beacon-chain/sync/initial-sync/testing"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
	prysmTime "github.com/prysmaticlabs/prysm/v4/time"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func TestProposeAttestation_OK(t *testing.T) {
	attesterServer := &Server{
		HeadFetcher:       &mock.ChainService{},
		P2P:               &mockp2p.MockBroadcaster{},
		AttPool:           attestations.NewPool(),
		OperationNotifier: (&mock.ChainService{}).OperationNotifier(),
	}
	head := util.NewBeaconBlock()
	head.Block.Slot = 999
	head.Block.ParentRoot = bytesutil.PadTo([]byte{'a'}, 32)
	root, err := head.Block.HashTreeRoot()
	require.NoError(t, err)

	validators := make([]*ethpb.Validator, 64)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			PublicKey:             make([]byte, 48),
			WithdrawalCredentials: make([]byte, 32),
			ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
		}
	}

	state, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, state.SetSlot(params.BeaconConfig().SlotsPerEpoch+1))
	require.NoError(t, state.SetValidators(validators))

	sk, err := bls.RandKey()
	require.NoError(t, err)
	sig := sk.Sign([]byte("dummy_test_data"))
	req := &ethpb.Attestation{
		Signature: sig.Marshal(),
		Data: &ethpb.AttestationData{
			BeaconBlockRoot: root[:],
			Source:          &ethpb.Checkpoint{Root: make([]byte, 32)},
			Target:          &ethpb.Checkpoint{Root: make([]byte, 32)},
		},
	}
	_, err = attesterServer.ProposeAttestation(context.Background(), req)
	assert.NoError(t, err)
}

func TestProposeAttestation_IncorrectSignature(t *testing.T) {
	attesterServer := &Server{
		HeadFetcher:       &mock.ChainService{},
		P2P:               &mockp2p.MockBroadcaster{},
		AttPool:           attestations.NewPool(),
		OperationNotifier: (&mock.ChainService{}).OperationNotifier(),
	}

	req := util.HydrateAttestation(&ethpb.Attestation{})
	wanted := "Incorrect attestation signature"
	_, err := attesterServer.ProposeAttestation(context.Background(), req)
	assert.ErrorContains(t, wanted, err)
}

func TestGetAttestationData_OK(t *testing.T) {
	block := util.NewBeaconBlock()
	block.Block.Slot = 3*params.BeaconConfig().SlotsPerEpoch + 1
	targetBlock := util.NewBeaconBlock()
	targetBlock.Block.Slot = 1 * params.BeaconConfig().SlotsPerEpoch
	justifiedBlock := util.NewBeaconBlock()
	justifiedBlock.Block.Slot = 2 * params.BeaconConfig().SlotsPerEpoch
	blockRoot, err := block.Block.HashTreeRoot()
	require.NoError(t, err, "Could not hash beacon block")
	justifiedRoot, err := justifiedBlock.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root for justified block")
	targetRoot, err := targetBlock.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root for target block")
	slot := 3*params.BeaconConfig().SlotsPerEpoch + 1
	beaconState, err := util.NewBeaconState()
	require.NoError(t, err)
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
	offset := int64(slot.Mul(params.BeaconConfig().SecondsPerSlot))
	attesterServer := &Server{
		SyncChecker:           &mockSync.Sync{IsSyncing: false},
		OptimisticModeFetcher: &mock.ChainService{Optimistic: false},
		TimeFetcher:           &mock.ChainService{Genesis: time.Now().Add(time.Duration(-1*offset) * time.Second)},
		CoreService: &core.Service{
			AttestationCache: cache.NewAttestationCache(),
			HeadFetcher: &mock.ChainService{
				State: beaconState, Root: blockRoot[:],
			},
			GenesisTimeFetcher: &mock.ChainService{
				Genesis: time.Now().Add(time.Duration(-1*offset) * time.Second),
			},
		},
	}

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
	as := Server{
		SyncChecker: &mockSync.Sync{IsSyncing: true},
	}
	_, err := as.GetAttestationData(context.Background(), &ethpb.AttestationDataRequest{})
	assert.ErrorContains(t, "Syncing to latest head", err)
}

func TestGetAttestationData_Optimistic(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig().Copy()
	cfg.BellatrixForkEpoch = 0
	params.OverrideBeaconConfig(cfg)

	as := &Server{
		SyncChecker:           &mockSync.Sync{},
		OptimisticModeFetcher: &mock.ChainService{Optimistic: true},
		TimeFetcher:           &mock.ChainService{Genesis: time.Now()},
		CoreService: &core.Service{
			GenesisTimeFetcher: &mock.ChainService{Genesis: time.Now()},
			HeadFetcher:        &mock.ChainService{},
		},
	}
	_, err := as.GetAttestationData(context.Background(), &ethpb.AttestationDataRequest{})
	s, ok := status.FromError(err)
	require.Equal(t, true, ok)
	require.DeepEqual(t, codes.Unavailable, s.Code())
	require.ErrorContains(t, errOptimisticMode.Error(), err)

	beaconState, err := util.NewBeaconState()
	require.NoError(t, err)
	as = &Server{
		SyncChecker:           &mockSync.Sync{},
		OptimisticModeFetcher: &mock.ChainService{Optimistic: false},
		TimeFetcher:           &mock.ChainService{Genesis: time.Now()},
		CoreService: &core.Service{
			GenesisTimeFetcher: &mock.ChainService{Genesis: time.Now()},
			HeadFetcher:        &mock.ChainService{Optimistic: false, State: beaconState},
			AttestationCache:   cache.NewAttestationCache(),
		},
	}
	_, err = as.GetAttestationData(context.Background(), &ethpb.AttestationDataRequest{})
	require.NoError(t, err)
}

func TestAttestationDataSlot_handlesInProgressRequest(t *testing.T) {
	s := &ethpb.BeaconState{Slot: 100}
	state, err := state_native.InitializeFromProtoPhase0(s)
	require.NoError(t, err)
	ctx := context.Background()
	slot := primitives.Slot(2)
	offset := int64(slot.Mul(params.BeaconConfig().SecondsPerSlot))
	server := &Server{
		SyncChecker:           &mockSync.Sync{IsSyncing: false},
		OptimisticModeFetcher: &mock.ChainService{Optimistic: false},
		TimeFetcher:           &mock.ChainService{Genesis: time.Now().Add(time.Duration(-1*offset) * time.Second)},
		CoreService: &core.Service{
			AttestationCache:   cache.NewAttestationCache(),
			HeadFetcher:        &mock.ChainService{State: state},
			GenesisTimeFetcher: &mock.ChainService{Genesis: time.Now().Add(time.Duration(-1*offset) * time.Second)},
		},
	}

	req := &ethpb.AttestationDataRequest{
		CommitteeIndex: 1,
		Slot:           slot,
	}

	res := &ethpb.AttestationData{
		CommitteeIndex: 1,
		Target:         &ethpb.Checkpoint{Epoch: 55, Root: make([]byte, 32)},
	}

	require.NoError(t, server.CoreService.AttestationCache.MarkInProgress(req))

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

		assert.NoError(t, server.CoreService.AttestationCache.Put(ctx, req, res))
		assert.NoError(t, server.CoreService.AttestationCache.MarkNotInProgress(req))
	}()

	wg.Wait()
}

func TestServer_GetAttestationData_InvalidRequestSlot(t *testing.T) {
	ctx := context.Background()

	slot := 3*params.BeaconConfig().SlotsPerEpoch + 1
	offset := int64(slot.Mul(params.BeaconConfig().SecondsPerSlot))
	attesterServer := &Server{
		SyncChecker:           &mockSync.Sync{IsSyncing: false},
		OptimisticModeFetcher: &mock.ChainService{Optimistic: false},
		TimeFetcher:           &mock.ChainService{Genesis: time.Now().Add(time.Duration(-1*offset) * time.Second)},
		CoreService: &core.Service{
			GenesisTimeFetcher: &mock.ChainService{Genesis: time.Now().Add(time.Duration(-1*offset) * time.Second)},
		},
	}

	req := &ethpb.AttestationDataRequest{
		Slot: 1000000000000,
	}
	_, err := attesterServer.GetAttestationData(ctx, req)
	assert.ErrorContains(t, "invalid request", err)
}

func TestServer_GetAttestationData_HeadStateSlotGreaterThanRequestSlot(t *testing.T) {
	// There exists a rare scenario where the validator may request an attestation for a slot less
	// than the head state's slot. The Ethereum consensus spec constraints require the block root the
	// attestation is referencing be less than or equal to the attestation data slot.
	// See: https://github.com/prysmaticlabs/prysm/issues/5164
	ctx := context.Background()
	db := dbutil.SetupDB(t)

	slot := 3*params.BeaconConfig().SlotsPerEpoch + 1
	block := util.NewBeaconBlock()
	block.Block.Slot = slot
	block2 := util.NewBeaconBlock()
	block2.Block.Slot = slot - 1
	targetBlock := util.NewBeaconBlock()
	targetBlock.Block.Slot = 1 * params.BeaconConfig().SlotsPerEpoch
	justifiedBlock := util.NewBeaconBlock()
	justifiedBlock.Block.Slot = 2 * params.BeaconConfig().SlotsPerEpoch
	blockRoot, err := block.Block.HashTreeRoot()
	require.NoError(t, err, "Could not hash beacon block")
	blockRoot2, err := block2.HashTreeRoot()
	require.NoError(t, err)
	util.SaveBlock(t, ctx, db, block2)
	justifiedRoot, err := justifiedBlock.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root for justified block")
	targetRoot, err := targetBlock.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root for target block")

	beaconState, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, beaconState.SetSlot(slot))
	offset := int64(slot.Mul(params.BeaconConfig().SecondsPerSlot))
	require.NoError(t, beaconState.SetGenesisTime(uint64(time.Now().Unix()-offset)))
	err = beaconState.SetLatestBlockHeader(util.HydrateBeaconHeader(&ethpb.BeaconBlockHeader{
		ParentRoot: blockRoot2[:],
	}))
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

	beaconstate := beaconState.Copy()
	require.NoError(t, beaconstate.SetSlot(beaconstate.Slot()-1))
	require.NoError(t, db.SaveState(ctx, beaconstate, blockRoot2))
	offset = int64(slot.Mul(params.BeaconConfig().SecondsPerSlot))
	attesterServer := &Server{
		SyncChecker:           &mockSync.Sync{IsSyncing: false},
		OptimisticModeFetcher: &mock.ChainService{Optimistic: false},
		TimeFetcher:           &mock.ChainService{Genesis: time.Now().Add(time.Duration(-1*offset) * time.Second)},
		CoreService: &core.Service{
			AttestationCache:   cache.NewAttestationCache(),
			HeadFetcher:        &mock.ChainService{State: beaconState, Root: blockRoot[:]},
			GenesisTimeFetcher: &mock.ChainService{Genesis: time.Now().Add(time.Duration(-1*offset) * time.Second)},
			StateGen:           stategen.New(db, doublylinkedtree.New()),
		},
	}
	require.NoError(t, db.SaveState(ctx, beaconState, blockRoot))
	util.SaveBlock(t, ctx, db, block)
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
	slot := primitives.Slot(5)
	block := util.NewBeaconBlock()
	block.Block.Slot = slot
	targetBlock := util.NewBeaconBlock()
	targetBlock.Block.Slot = 0
	justifiedBlock := util.NewBeaconBlock()
	justifiedBlock.Block.Slot = 0
	blockRoot, err := block.Block.HashTreeRoot()
	require.NoError(t, err, "Could not hash beacon block")
	justifiedRoot, err := justifiedBlock.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root for justified block")
	targetRoot, err := targetBlock.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root for target block")

	beaconState, err := util.NewBeaconState()
	require.NoError(t, err)
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
	offset := int64(slot.Mul(params.BeaconConfig().SecondsPerSlot))
	attesterServer := &Server{
		SyncChecker:           &mockSync.Sync{IsSyncing: false},
		OptimisticModeFetcher: &mock.ChainService{Optimistic: false},
		TimeFetcher:           &mock.ChainService{Genesis: prysmTime.Now().Add(time.Duration(-1*offset) * time.Second)},
		CoreService: &core.Service{
			AttestationCache: cache.NewAttestationCache(),
			HeadFetcher: &mock.ChainService{
				State: beaconState, Root: blockRoot[:],
			},
			GenesisTimeFetcher: &mock.ChainService{Genesis: prysmTime.Now().Add(time.Duration(-1*offset) * time.Second)},
		},
	}

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
	attesterServer := &Server{
		HeadFetcher:       &mock.ChainService{},
		P2P:               &mockp2p.MockBroadcaster{},
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
	// fixed seed
	s := rand.NewSource(10)
	randGen := rand.New(s)

	attesterServer := &Server{
		HeadFetcher:       &mock.ChainService{},
		P2P:               &mockp2p.MockBroadcaster{},
		AttPool:           attestations.NewPool(),
		OperationNotifier: (&mock.ChainService{}).OperationNotifier(),
	}

	var ss []primitives.Slot
	var comIdxs []primitives.CommitteeIndex
	var isAggregator []bool

	for i := primitives.Slot(100); i < 200; i++ {
		ss = append(ss, i)
		comIdxs = append(comIdxs, primitives.CommitteeIndex(randGen.Int63n(64)))
		boolVal := randGen.Uint64()%2 == 0
		isAggregator = append(isAggregator, boolVal)
	}

	ss = append(ss, 321)

	_, err := attesterServer.SubscribeCommitteeSubnets(context.Background(), &ethpb.CommitteeSubnetsSubscribeRequest{
		Slots:        ss,
		CommitteeIds: comIdxs,
		IsAggregator: isAggregator,
	})
	assert.ErrorContains(t, "request fields are not the same length", err)
}

func TestServer_SubscribeCommitteeSubnets_MultipleSlots(t *testing.T) {
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

	state, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, state.SetValidators(validators))

	attesterServer := &Server{
		HeadFetcher:       &mock.ChainService{State: state},
		P2P:               &mockp2p.MockBroadcaster{},
		AttPool:           attestations.NewPool(),
		OperationNotifier: (&mock.ChainService{}).OperationNotifier(),
	}

	var ss []primitives.Slot
	var comIdxs []primitives.CommitteeIndex
	var isAggregator []bool

	for i := primitives.Slot(100); i < 200; i++ {
		ss = append(ss, i)
		comIdxs = append(comIdxs, primitives.CommitteeIndex(randGen.Int63n(64)))
		boolVal := randGen.Uint64()%2 == 0
		isAggregator = append(isAggregator, boolVal)
	}

	_, err = attesterServer.SubscribeCommitteeSubnets(context.Background(), &ethpb.CommitteeSubnetsSubscribeRequest{
		Slots:        ss,
		CommitteeIds: comIdxs,
		IsAggregator: isAggregator,
	})
	require.NoError(t, err)
	for i := primitives.Slot(100); i < 200; i++ {
		subnets := cache.SubnetIDs.GetAttesterSubnetIDs(i)
		assert.Equal(t, 1, len(subnets))
		if isAggregator[i-100] {
			subnets = cache.SubnetIDs.GetAggregatorSubnetIDs(i)
			assert.Equal(t, 1, len(subnets))
		}
	}
}
