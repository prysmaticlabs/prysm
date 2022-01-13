package validator

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/async/event"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/signing"
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/container/trie"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/mock"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestValidatorIndex_OK(t *testing.T) {
	st, err := util.NewBeaconState()
	require.NoError(t, err)

	pubKey := pubKey(1)

	err = st.SetValidators([]*ethpb.Validator{{PublicKey: pubKey}})
	require.NoError(t, err)

	Server := &Server{
		HeadFetcher: &mockChain.ChainService{State: st},
	}

	req := &ethpb.ValidatorIndexRequest{
		PublicKey: pubKey,
	}
	_, err = Server.ValidatorIndex(context.Background(), req)
	assert.NoError(t, err, "Could not get validator index")
}

func TestWaitForActivation_ContextClosed(t *testing.T) {
	beaconState, err := v1.InitializeFromProto(&ethpb.BeaconState{
		Slot:       0,
		Validators: []*ethpb.Validator{},
	})
	require.NoError(t, err)
	block := util.NewBeaconBlock()
	genesisRoot, err := block.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")

	ctx, cancel := context.WithCancel(context.Background())
	depositCache, err := depositcache.New()
	require.NoError(t, err)

	vs := &Server{
		Ctx:                ctx,
		ChainStartFetcher:  &mockPOW.POWChain{},
		BlockFetcher:       &mockPOW.POWChain{},
		Eth1InfoFetcher:    &mockPOW.POWChain{},
		CanonicalStateChan: make(chan *ethpb.BeaconState, 1),
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
	// This test breaks if it doesnt use mainnet config
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	ctx := context.Background()

	priv1, err := bls.RandKey()
	require.NoError(t, err)
	priv2, err := bls.RandKey()
	require.NoError(t, err)

	pubKey1 := priv1.PublicKey().Marshal()
	pubKey2 := priv2.PublicKey().Marshal()

	beaconState := &ethpb.BeaconState{
		Slot: 4000,
		Validators: []*ethpb.Validator{
			{
				ActivationEpoch:       0,
				ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
				PublicKey:             pubKey1,
				WithdrawalCredentials: make([]byte, 32),
			},
		},
	}
	block := util.NewBeaconBlock()
	genesisRoot, err := block.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	depData := &ethpb.Deposit_Data{
		PublicKey:             pubKey1,
		WithdrawalCredentials: bytesutil.PadTo([]byte("hey"), 32),
		Signature:             make([]byte, 96),
	}
	domain, err := signing.ComputeDomain(params.BeaconConfig().DomainDeposit, nil, nil)
	require.NoError(t, err)
	signingRoot, err := signing.ComputeSigningRoot(depData, domain)
	require.NoError(t, err)
	depData.Signature = priv1.Sign(signingRoot[:]).Marshal()

	deposit := &ethpb.Deposit{
		Data: depData,
	}
	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not setup deposit trie")
	depositCache, err := depositcache.New()
	require.NoError(t, err)

	assert.NoError(t, depositCache.InsertDeposit(ctx, deposit, 10 /*blockNum*/, 0, depositTrie.HashTreeRoot()))
	trie, err := v1.InitializeFromProtoUnsafe(beaconState)
	require.NoError(t, err)
	vs := &Server{
		Ctx:                context.Background(),
		CanonicalStateChan: make(chan *ethpb.BeaconState, 1),
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
	priv1, err := bls.RandKey()
	require.NoError(t, err)
	priv2, err := bls.RandKey()
	require.NoError(t, err)
	priv3, err := bls.RandKey()
	require.NoError(t, err)

	pubKey1 := priv1.PublicKey().Marshal()
	pubKey2 := priv2.PublicKey().Marshal()
	pubKey3 := priv3.PublicKey().Marshal()

	beaconState := &ethpb.BeaconState{
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
	block := util.NewBeaconBlock()
	genesisRoot, err := block.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	trie, err := v1.InitializeFromProtoUnsafe(beaconState)
	require.NoError(t, err)
	vs := &Server{
		Ctx:                context.Background(),
		CanonicalStateChan: make(chan *ethpb.BeaconState, 1),
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
	ctx, cancel := context.WithCancel(context.Background())
	chainService := &mockChain.ChainService{}
	Server := &Server{
		Ctx: ctx,
		ChainStartFetcher: &mockPOW.FaultyMockPOWChain{
			ChainFeed: new(event.Feed),
		},
		StateNotifier: chainService.StateNotifier(),
		HeadFetcher:   chainService,
	}

	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mock.NewMockBeaconNodeValidator_WaitForChainStartServer(ctrl)
	mockStream.EXPECT().Context().Return(ctx)
	go func(tt *testing.T) {
		err := Server.WaitForChainStart(&emptypb.Empty{}, mockStream)
		assert.ErrorContains(tt, "Context canceled", err)
		<-exitRoutine
	}(t)
	cancel()
	exitRoutine <- true
}

func TestWaitForChainStart_AlreadyStarted(t *testing.T) {
	st, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, st.SetSlot(3))
	genesisValidatorsRoot := bytesutil.ToBytes32([]byte("validators"))
	require.NoError(t, st.SetGenesisValidatorRoot(genesisValidatorsRoot[:]))

	chainService := &mockChain.ChainService{State: st, ValidatorsRoot: genesisValidatorsRoot}
	Server := &Server{
		Ctx: context.Background(),
		ChainStartFetcher: &mockPOW.POWChain{
			ChainFeed: new(event.Feed),
		},
		StateNotifier: chainService.StateNotifier(),
		HeadFetcher:   chainService,
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mock.NewMockBeaconNodeValidator_WaitForChainStartServer(ctrl)
	mockStream.EXPECT().Send(
		&ethpb.ChainStartResponse{
			Started:               true,
			GenesisTime:           uint64(time.Unix(0, 0).Unix()),
			GenesisValidatorsRoot: genesisValidatorsRoot[:],
		},
	).Return(nil)
	mockStream.EXPECT().Context().Return(context.Background())
	assert.NoError(t, Server.WaitForChainStart(&emptypb.Empty{}, mockStream), "Could not call RPC method")
}

func TestWaitForChainStart_HeadStateDoesNotExist(t *testing.T) {
	genesisValidatorRoot := params.BeaconConfig().ZeroHash

	// Set head state to nil
	chainService := &mockChain.ChainService{State: nil}
	notifier := chainService.StateNotifier()
	Server := &Server{
		Ctx: context.Background(),
		ChainStartFetcher: &mockPOW.POWChain{
			ChainFeed: new(event.Feed),
		},
		StateNotifier: chainService.StateNotifier(),
		HeadFetcher:   chainService,
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mock.NewMockBeaconNodeValidator_WaitForChainStartServer(ctrl)
	mockStream.EXPECT().Context().Return(context.Background())

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		assert.NoError(t, Server.WaitForChainStart(&emptypb.Empty{}, mockStream), "Could not call RPC method")
		wg.Done()
	}()
	// Simulate a late state initialization event, so that
	// method is able to handle race condition here.
	notifier.StateFeed().Send(&feed.Event{
		Type: statefeed.Initialized,
		Data: &statefeed.InitializedData{
			StartTime:             time.Unix(0, 0),
			GenesisValidatorsRoot: genesisValidatorRoot[:],
		},
	})
	util.WaitTimeout(wg, time.Second)
}

func TestWaitForChainStart_NotStartedThenLogFired(t *testing.T) {
	hook := logTest.NewGlobal()

	genesisValidatorsRoot := bytesutil.ToBytes32([]byte("validators"))
	chainService := &mockChain.ChainService{}
	Server := &Server{
		Ctx: context.Background(),
		ChainStartFetcher: &mockPOW.FaultyMockPOWChain{
			ChainFeed: new(event.Feed),
		},
		StateNotifier: chainService.StateNotifier(),
		HeadFetcher:   chainService,
	}
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mock.NewMockBeaconNodeValidator_WaitForChainStartServer(ctrl)
	mockStream.EXPECT().Send(
		&ethpb.ChainStartResponse{
			Started:               true,
			GenesisTime:           uint64(time.Unix(0, 0).Unix()),
			GenesisValidatorsRoot: genesisValidatorsRoot[:],
		},
	).Return(nil)
	mockStream.EXPECT().Context().Return(context.Background())
	go func(tt *testing.T) {
		assert.NoError(tt, Server.WaitForChainStart(&emptypb.Empty{}, mockStream))
		<-exitRoutine
	}(t)

	// Send in a loop to ensure it is delivered (busy wait for the service to subscribe to the state feed).
	for sent := 0; sent == 0; {
		sent = Server.StateNotifier.StateFeed().Send(&feed.Event{
			Type: statefeed.Initialized,
			Data: &statefeed.InitializedData{
				StartTime:             time.Unix(0, 0),
				GenesisValidatorsRoot: genesisValidatorsRoot[:],
			},
		})
	}

	exitRoutine <- true
	require.LogsContain(t, hook, "Sending genesis time")
}
