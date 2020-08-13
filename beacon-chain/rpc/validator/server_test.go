package validator

import (
	"context"
	"testing"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/mock"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestValidatorIndex_OK(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()
	st := testutil.NewBeaconState()
	require.NoError(t, db.SaveState(ctx, st.Copy(), [32]byte{}))

	pubKey := pubKey(1)

	err := st.SetValidators([]*ethpb.Validator{{PublicKey: pubKey}})
	require.NoError(t, err)

	Server := &Server{
		BeaconDB:    db,
		HeadFetcher: &mockChain.ChainService{State: st},
	}

	req := &ethpb.ValidatorIndexRequest{
		PublicKey: pubKey,
	}
	_, err = Server.ValidatorIndex(context.Background(), req)
	assert.NoError(t, err, "Could not get validator index")
}

func TestWaitForActivation_ContextClosed(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()

	beaconState, err := stateTrie.InitializeFromProto(&pbp2p.BeaconState{
		Slot:       0,
		Validators: []*ethpb.Validator{},
	})
	require.NoError(t, err)
	block := testutil.NewBeaconBlock()
	require.NoError(t, db.SaveBlock(ctx, block), "Could not save genesis block")
	genesisRoot, err := stateutil.BlockRoot(block.Block)
	require.NoError(t, err, "Could not get signing root")

	ctx, cancel := context.WithCancel(context.Background())
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	vs := &Server{
		BeaconDB:           db,
		Ctx:                ctx,
		ChainStartFetcher:  &mockPOW.POWChain{},
		BlockFetcher:       &mockPOW.POWChain{},
		Eth1InfoFetcher:    &mockPOW.POWChain{},
		CanonicalStateChan: make(chan *pbp2p.BeaconState, 1),
		DepositFetcher:     depositCache,
		HeadFetcher:        &mockChain.ChainService{State: beaconState, Root: genesisRoot[:]},
	}
	req := &ethpb.ValidatorActivationRequest{
		PublicKeys: [][]byte{pubKey(1)},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockChainStream := mock.NewMockBeaconNodeValidator_WaitForActivationServer(ctrl)
	mockChainStream.EXPECT().Context().Return(context.Background())
	mockChainStream.EXPECT().Send(gomock.Any()).Return(nil)
	mockChainStream.EXPECT().Context().Return(context.Background())
	exitRoutine := make(chan bool)
	go func(tt *testing.T) {
		want := "context canceled"
		assert.ErrorContains(tt, want, vs.WaitForActivation(req, mockChainStream))
		<-exitRoutine
	}(t)
	cancel()
	exitRoutine <- true
}

func TestWaitForActivation_ValidatorOriginallyExists(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	// This test breaks if it doesnt use mainnet config
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	ctx := context.Background()

	priv1 := bls.RandKey()
	priv2 := bls.RandKey()

	pubKey1 := priv1.PublicKey().Marshal()[:]
	pubKey2 := priv2.PublicKey().Marshal()[:]

	beaconState := &pbp2p.BeaconState{
		Slot: 4000,
		Validators: []*ethpb.Validator{
			{
				ActivationEpoch: 0,
				ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
				PublicKey:       pubKey1,
			},
		},
	}
	block := testutil.NewBeaconBlock()
	genesisRoot, err := stateutil.BlockRoot(block.Block)
	require.NoError(t, err, "Could not get signing root")
	depData := &ethpb.Deposit_Data{
		PublicKey:             pubKey1,
		WithdrawalCredentials: []byte("hey"),
	}
	domain, err := helpers.ComputeDomain(params.BeaconConfig().DomainDeposit, nil, nil)
	require.NoError(t, err)
	signingRoot, err := helpers.ComputeSigningRoot(depData, domain)
	require.NoError(t, err)
	depData.Signature = priv1.Sign(signingRoot[:]).Marshal()[:]

	deposit := &ethpb.Deposit{
		Data: depData,
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	require.NoError(t, err, "Could not setup deposit trie")
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	depositCache.InsertDeposit(ctx, deposit, 10 /*blockNum*/, 0, depositTrie.Root())
	trie, err := stateTrie.InitializeFromProtoUnsafe(beaconState)
	require.NoError(t, err)
	vs := &Server{
		BeaconDB:           db,
		Ctx:                context.Background(),
		CanonicalStateChan: make(chan *pbp2p.BeaconState, 1),
		ChainStartFetcher:  &mockPOW.POWChain{},
		BlockFetcher:       &mockPOW.POWChain{},
		Eth1InfoFetcher:    &mockPOW.POWChain{},
		DepositFetcher:     depositCache,
		HeadFetcher:        &mockChain.ChainService{State: trie, Root: genesisRoot[:]},
	}
	req := &ethpb.ValidatorActivationRequest{
		PublicKeys: [][]byte{pubKey1, pubKey2},
	}
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()
	mockChainStream := mock.NewMockBeaconNodeValidator_WaitForActivationServer(ctrl)
	mockChainStream.EXPECT().Context().Return(context.Background())
	mockChainStream.EXPECT().Send(
		&ethpb.ValidatorActivationResponse{
			Statuses: []*ethpb.ValidatorActivationResponse_Status{
				{
					PublicKey: pubKey1,
					Status: &ethpb.ValidatorStatusResponse{
						Status: ethpb.ValidatorStatus_ACTIVE,
					},
					Index: 0,
				},
				{
					PublicKey: pubKey2,
					Status: &ethpb.ValidatorStatusResponse{
						ActivationEpoch: params.BeaconConfig().FarFutureEpoch,
					},
					Index: nonExistentIndex,
				},
			},
		},
	).Return(nil)

	require.NoError(t, vs.WaitForActivation(req, mockChainStream), "Could not setup wait for activation stream")
}

func TestWaitForActivation_MultipleStatuses(t *testing.T) {
	db, _ := dbutil.SetupDB(t)

	priv1 := bls.RandKey()
	priv2 := bls.RandKey()
	priv3 := bls.RandKey()

	pubKey1 := priv1.PublicKey().Marshal()[:]
	pubKey2 := priv2.PublicKey().Marshal()[:]
	pubKey3 := priv3.PublicKey().Marshal()[:]

	beaconState := &pbp2p.BeaconState{
		Slot: 4000,
		Validators: []*ethpb.Validator{
			{
				PublicKey:       pubKey1,
				ActivationEpoch: 1,
				ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
			},
			{
				PublicKey:                  pubKey2,
				ActivationEpoch:            params.BeaconConfig().FarFutureEpoch,
				ActivationEligibilityEpoch: 6,
				ExitEpoch:                  params.BeaconConfig().FarFutureEpoch,
			},
			{
				PublicKey:                  pubKey3,
				ActivationEpoch:            0,
				ActivationEligibilityEpoch: 0,
				ExitEpoch:                  0,
			},
		},
	}
	block := testutil.NewBeaconBlock()
	genesisRoot, err := stateutil.BlockRoot(block.Block)
	require.NoError(t, err, "Could not get signing root")
	trie, err := stateTrie.InitializeFromProtoUnsafe(beaconState)
	require.NoError(t, err)
	vs := &Server{
		BeaconDB:           db,
		Ctx:                context.Background(),
		CanonicalStateChan: make(chan *pbp2p.BeaconState, 1),
		ChainStartFetcher:  &mockPOW.POWChain{},
		HeadFetcher:        &mockChain.ChainService{State: trie, Root: genesisRoot[:]},
	}
	req := &ethpb.ValidatorActivationRequest{
		PublicKeys: [][]byte{pubKey1, pubKey2, pubKey3},
	}
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()
	mockChainStream := mock.NewMockBeaconNodeValidator_WaitForActivationServer(ctrl)
	mockChainStream.EXPECT().Context().Return(context.Background())
	mockChainStream.EXPECT().Send(
		&ethpb.ValidatorActivationResponse{
			Statuses: []*ethpb.ValidatorActivationResponse_Status{
				{
					PublicKey: pubKey1,
					Status: &ethpb.ValidatorStatusResponse{
						Status:          ethpb.ValidatorStatus_ACTIVE,
						ActivationEpoch: 1,
					},
					Index: 0,
				},
				{
					PublicKey: pubKey2,
					Status: &ethpb.ValidatorStatusResponse{
						Status:                    ethpb.ValidatorStatus_PENDING,
						ActivationEpoch:           params.BeaconConfig().FarFutureEpoch,
						PositionInActivationQueue: 1,
					},
					Index: 1,
				},
				{
					PublicKey: pubKey3,
					Status: &ethpb.ValidatorStatusResponse{
						Status: ethpb.ValidatorStatus_EXITED,
					},
					Index: 2,
				},
			},
		},
	).Return(nil)

	require.NoError(t, vs.WaitForActivation(req, mockChainStream), "Could not setup wait for activation stream")
}

func TestWaitForChainStart_ContextClosed(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()

	ctx, cancel := context.WithCancel(context.Background())
	chainService := &mockChain.ChainService{}
	Server := &Server{
		Ctx: ctx,
		ChainStartFetcher: &mockPOW.FaultyMockPOWChain{
			ChainFeed: new(event.Feed),
		},
		StateNotifier: chainService.StateNotifier(),
		BeaconDB:      db,
		HeadFetcher:   chainService,
	}

	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mock.NewMockBeaconNodeValidator_WaitForChainStartServer(ctrl)
	go func(tt *testing.T) {
		err := Server.WaitForChainStart(&ptypes.Empty{}, mockStream)
		assert.ErrorContains(tt, "Context canceled", err)
		<-exitRoutine
	}(t)
	cancel()
	exitRoutine <- true
}

func TestWaitForChainStart_AlreadyStarted(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()
	headBlockRoot := [32]byte{0x01, 0x02}
	trie := testutil.NewBeaconState()
	require.NoError(t, trie.SetSlot(3))
	require.NoError(t, db.SaveState(ctx, trie, headBlockRoot))
	require.NoError(t, db.SaveHeadBlockRoot(ctx, headBlockRoot))

	chainService := &mockChain.ChainService{State: trie}
	Server := &Server{
		Ctx: context.Background(),
		ChainStartFetcher: &mockPOW.POWChain{
			ChainFeed: new(event.Feed),
		},
		BeaconDB:      db,
		StateNotifier: chainService.StateNotifier(),
		HeadFetcher:   chainService,
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mock.NewMockBeaconNodeValidator_WaitForChainStartServer(ctrl)
	mockStream.EXPECT().Send(
		&ethpb.ChainStartResponse{
			Started:     true,
			GenesisTime: uint64(time.Unix(0, 0).Unix()),
		},
	).Return(nil)
	assert.NoError(t, Server.WaitForChainStart(&ptypes.Empty{}, mockStream), "Could not call RPC method")
}

func TestWaitForChainStart_NotStartedThenLogFired(t *testing.T) {
	db, _ := dbutil.SetupDB(t)

	hook := logTest.NewGlobal()
	chainService := &mockChain.ChainService{}
	Server := &Server{
		Ctx: context.Background(),
		ChainStartFetcher: &mockPOW.FaultyMockPOWChain{
			ChainFeed: new(event.Feed),
		},
		BeaconDB:      db,
		StateNotifier: chainService.StateNotifier(),
		HeadFetcher:   chainService,
	}
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mock.NewMockBeaconNodeValidator_WaitForChainStartServer(ctrl)
	mockStream.EXPECT().Send(
		&ethpb.ChainStartResponse{
			Started:     true,
			GenesisTime: uint64(time.Unix(0, 0).Unix()),
		},
	).Return(nil)
	go func(tt *testing.T) {
		assert.NoError(tt, Server.WaitForChainStart(&ptypes.Empty{}, mockStream))
		<-exitRoutine
	}(t)

	// Send in a loop to ensure it is delivered (busy wait for the service to subscribe to the state feed).
	for sent := 0; sent == 0; {
		sent = Server.StateNotifier.StateFeed().Send(&feed.Event{
			Type: statefeed.ChainStarted,
			Data: &statefeed.ChainStartedData{
				StartTime: time.Unix(0, 0),
			},
		})
	}

	exitRoutine <- true
	require.LogsContain(t, hook, "Sending genesis time")
}

func TestWaitForSynced_ContextClosed(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()

	ctx, cancel := context.WithCancel(context.Background())
	chainService := &mockChain.ChainService{}
	Server := &Server{
		Ctx: ctx,
		ChainStartFetcher: &mockPOW.FaultyMockPOWChain{
			ChainFeed: new(event.Feed),
		},
		StateNotifier: chainService.StateNotifier(),
		BeaconDB:      db,
		HeadFetcher:   chainService,
	}

	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mock.NewMockBeaconNodeValidator_WaitForSyncedServer(ctrl)
	go func(tt *testing.T) {
		err := Server.WaitForSynced(&ptypes.Empty{}, mockStream)
		assert.ErrorContains(tt, "Context canceled", err)
		<-exitRoutine
	}(t)
	cancel()
	exitRoutine <- true
}

func TestWaitForSynced_AlreadySynced(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()
	headBlockRoot := [32]byte{0x01, 0x02}
	trie := testutil.NewBeaconState()
	require.NoError(t, trie.SetSlot(3))
	require.NoError(t, db.SaveState(ctx, trie, headBlockRoot))
	require.NoError(t, db.SaveHeadBlockRoot(ctx, headBlockRoot))

	chainService := &mockChain.ChainService{State: trie}
	Server := &Server{
		Ctx: context.Background(),
		ChainStartFetcher: &mockPOW.POWChain{
			ChainFeed: new(event.Feed),
		},
		BeaconDB:      db,
		StateNotifier: chainService.StateNotifier(),
		HeadFetcher:   chainService,
		SyncChecker:   &mockSync.Sync{IsSyncing: false},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mock.NewMockBeaconNodeValidator_WaitForSyncedServer(ctrl)
	mockStream.EXPECT().Send(
		&ethpb.SyncedResponse{
			Synced:      true,
			GenesisTime: uint64(time.Unix(0, 0).Unix()),
		},
	).Return(nil)
	assert.NoError(t, Server.WaitForSynced(&ptypes.Empty{}, mockStream), "Could not call RPC method")
}

func TestWaitForSynced_NotStartedThenLogFired(t *testing.T) {
	db, _ := dbutil.SetupDB(t)

	hook := logTest.NewGlobal()
	chainService := &mockChain.ChainService{}
	Server := &Server{
		Ctx: context.Background(),
		ChainStartFetcher: &mockPOW.FaultyMockPOWChain{
			ChainFeed: new(event.Feed),
		},
		BeaconDB:      db,
		StateNotifier: chainService.StateNotifier(),
		HeadFetcher:   chainService,
	}
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mock.NewMockBeaconNodeValidator_WaitForSyncedServer(ctrl)
	mockStream.EXPECT().Send(
		&ethpb.SyncedResponse{
			Synced:      true,
			GenesisTime: uint64(time.Unix(0, 0).Unix()),
		},
	).Return(nil)
	go func(tt *testing.T) {
		assert.NoError(tt, Server.WaitForSynced(&ptypes.Empty{}, mockStream), "Could not call RPC method")
		<-exitRoutine
	}(t)

	// Send in a loop to ensure it is delivered (busy wait for the service to subscribe to the state feed).
	for sent := 0; sent == 0; {
		sent = Server.StateNotifier.StateFeed().Send(&feed.Event{
			Type: statefeed.Synced,
			Data: &statefeed.SyncedData{
				StartTime: time.Unix(0, 0),
			},
		})
	}

	exitRoutine <- true
	require.LogsContain(t, hook, "Sending genesis time")
}
