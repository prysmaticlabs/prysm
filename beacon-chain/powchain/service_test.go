package powchain

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/async/event"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/config/params"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit"
	"github.com/prysmaticlabs/prysm/contracts/deposit/mock"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/monitoring/clientstats"
	"github.com/prysmaticlabs/prysm/network"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
	"github.com/prysmaticlabs/prysm/time/slots"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

var _ ChainStartFetcher = (*Service)(nil)
var _ ChainInfoFetcher = (*Service)(nil)
var _ POWBlockFetcher = (*Service)(nil)
var _ Chain = (*Service)(nil)

type goodLogger struct {
	backend *backends.SimulatedBackend
}

func (g *goodLogger) SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- gethTypes.Log) (ethereum.Subscription, error) {
	if g.backend == nil {
		return new(event.Feed).Subscribe(ch), nil
	}
	return g.backend.SubscribeFilterLogs(ctx, q, ch)
}

func (g *goodLogger) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]gethTypes.Log, error) {
	if g.backend == nil {
		logs := make([]gethTypes.Log, 3)
		for i := 0; i < len(logs); i++ {
			logs[i].Address = common.Address{}
			logs[i].Topics = make([]common.Hash, 5)
			logs[i].Topics[0] = common.Hash{'a'}
			logs[i].Topics[1] = common.Hash{'b'}
			logs[i].Topics[2] = common.Hash{'c'}

		}
		return logs, nil
	}
	return g.backend.FilterLogs(ctx, q)
}

type goodNotifier struct {
	MockStateFeed *event.Feed
}

func (g *goodNotifier) StateFeed() *event.Feed {
	if g.MockStateFeed == nil {
		g.MockStateFeed = new(event.Feed)
	}
	return g.MockStateFeed
}

type goodFetcher struct {
	backend *backends.SimulatedBackend
}

func (g *goodFetcher) HeaderByHash(_ context.Context, hash common.Hash) (*gethTypes.Header, error) {
	if bytes.Equal(hash.Bytes(), common.BytesToHash([]byte{0}).Bytes()) {
		return nil, fmt.Errorf("expected block hash to be nonzero %v", hash)
	}
	if g.backend == nil {
		return &gethTypes.Header{
			Number: big.NewInt(0),
		}, nil
	}
	header := g.backend.Blockchain().GetHeaderByHash(hash)
	if header == nil {
		return nil, errors.New("nil header returned")
	}
	return header, nil

}

func (g *goodFetcher) HeaderByNumber(_ context.Context, number *big.Int) (*gethTypes.Header, error) {
	if g.backend == nil {
		return &gethTypes.Header{
			Number: big.NewInt(15),
			Time:   150,
		}, nil
	}
	var header *gethTypes.Header
	if number == nil {
		header = g.backend.Blockchain().CurrentHeader()
	} else {
		header = g.backend.Blockchain().GetHeaderByNumber(number.Uint64())
	}
	if header == nil {
		return nil, errors.New("nil header returned")
	}
	return header, nil
}

func (_ *goodFetcher) SyncProgress(_ context.Context) (*ethereum.SyncProgress, error) {
	return nil, nil
}

var depositsReqForChainStart = 64

func TestStart_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB := dbutil.SetupDB(t)
	testAcc, err := mock.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	server, endpoint, err := mockPOW.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	web3Service, err := NewService(context.Background(),
		WithHttpEndpoints([]string{endpoint}),
		WithDepositContractAddress(testAcc.ContractAddr),
		WithDatabase(beaconDB),
	)
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")
	web3Service = setDefaultMocks(web3Service)
	web3Service.rpcClient = &mockPOW.RPCClient{Backend: testAcc.Backend}
	web3Service.depositContractCaller, err = contracts.NewDepositContractCaller(testAcc.ContractAddr, testAcc.Backend)
	require.NoError(t, err)
	testAcc.Backend.Commit()

	web3Service.Start()
	if len(hook.Entries) > 0 {
		msg := hook.LastEntry().Message
		want := "Could not connect to ETH1.0 chain RPC client"
		if strings.Contains(want, msg) {
			t.Errorf("incorrect log, expected %s, got %s", want, msg)
		}
	}
	hook.Reset()
	web3Service.cancel()
}

func TestStart_NoHttpEndpointDefinedFails_WithoutChainStarted(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB := dbutil.SetupDB(t)
	testAcc, err := mock.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	_, err = NewService(context.Background(),
		WithHttpEndpoints([]string{""}),
		WithDepositContractAddress(testAcc.ContractAddr),
		WithDatabase(beaconDB),
	)
	require.NoError(t, err)
	require.LogsDoNotContain(t, hook, "missing address")
}

func TestStop_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	testAcc, err := mock.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	beaconDB := dbutil.SetupDB(t)
	server, endpoint, err := mockPOW.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	web3Service, err := NewService(context.Background(),
		WithHttpEndpoints([]string{endpoint}),
		WithDepositContractAddress(testAcc.ContractAddr),
		WithDatabase(beaconDB),
	)
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")
	web3Service = setDefaultMocks(web3Service)
	web3Service.depositContractCaller, err = contracts.NewDepositContractCaller(testAcc.ContractAddr, testAcc.Backend)
	require.NoError(t, err)

	testAcc.Backend.Commit()

	err = web3Service.Stop()
	require.NoError(t, err, "Unable to stop web3 ETH1.0 chain service")

	// The context should have been canceled.
	assert.NotNil(t, web3Service.ctx.Err(), "Context wasnt canceled")

	hook.Reset()
}

func TestService_Eth1Synced(t *testing.T) {
	testAcc, err := mock.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	beaconDB := dbutil.SetupDB(t)
	server, endpoint, err := mockPOW.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	web3Service, err := NewService(context.Background(),
		WithHttpEndpoints([]string{endpoint}),
		WithDepositContractAddress(testAcc.ContractAddr),
		WithDatabase(beaconDB),
	)
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")
	web3Service = setDefaultMocks(web3Service)
	web3Service.depositContractCaller, err = contracts.NewDepositContractCaller(testAcc.ContractAddr, testAcc.Backend)
	require.NoError(t, err)
	web3Service.eth1DataFetcher = &goodFetcher{backend: testAcc.Backend}

	currTime := testAcc.Backend.Blockchain().CurrentHeader().Time
	now := time.Now()
	assert.NoError(t, testAcc.Backend.AdjustTime(now.Sub(time.Unix(int64(currTime), 0))))
	testAcc.Backend.Commit()

	synced, err := web3Service.isEth1NodeSynced()
	require.NoError(t, err)
	assert.Equal(t, true, synced, "Expected eth1 nodes to be synced")
}

func TestFollowBlock_OK(t *testing.T) {
	testAcc, err := mock.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	beaconDB := dbutil.SetupDB(t)
	server, endpoint, err := mockPOW.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	web3Service, err := NewService(context.Background(),
		WithHttpEndpoints([]string{endpoint}),
		WithDepositContractAddress(testAcc.ContractAddr),
		WithDatabase(beaconDB),
	)
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")

	// simulated backend sets eth1 block
	// time as 10 seconds
	params.SetupTestConfigCleanup(t)
	conf := params.BeaconConfig()
	conf.SecondsPerETH1Block = 10
	params.OverrideBeaconConfig(conf)

	web3Service = setDefaultMocks(web3Service)
	web3Service.eth1DataFetcher = &goodFetcher{backend: testAcc.Backend}
	baseHeight := testAcc.Backend.Blockchain().CurrentBlock().NumberU64()
	// process follow_distance blocks
	for i := 0; i < int(params.BeaconConfig().Eth1FollowDistance); i++ {
		testAcc.Backend.Commit()
	}
	// set current height
	web3Service.latestEth1Data.BlockHeight = testAcc.Backend.Blockchain().CurrentBlock().NumberU64()
	web3Service.latestEth1Data.BlockTime = testAcc.Backend.Blockchain().CurrentBlock().Time()

	h, err := web3Service.followBlockHeight(context.Background())
	require.NoError(t, err)
	assert.Equal(t, baseHeight, h, "Unexpected block height")
	numToForward := uint64(2)
	expectedHeight := numToForward + baseHeight
	// forward 2 blocks
	for i := uint64(0); i < numToForward; i++ {
		testAcc.Backend.Commit()
	}
	// set current height
	web3Service.latestEth1Data.BlockHeight = testAcc.Backend.Blockchain().CurrentBlock().NumberU64()
	web3Service.latestEth1Data.BlockTime = testAcc.Backend.Blockchain().CurrentBlock().Time()

	h, err = web3Service.followBlockHeight(context.Background())
	require.NoError(t, err)
	assert.Equal(t, expectedHeight, h, "Unexpected block height")
}

func TestStatus(t *testing.T) {
	now := time.Now()

	beforeFiveMinutesAgo := uint64(now.Add(-5*time.Minute - 30*time.Second).Unix())
	afterFiveMinutesAgo := uint64(now.Add(-5*time.Minute + 30*time.Second).Unix())

	testCases := map[*Service]string{
		// "status is ok" cases
		{}: "",
		{isRunning: true, latestEth1Data: &ethpb.LatestETH1Data{BlockTime: afterFiveMinutesAgo}}:   "",
		{isRunning: false, latestEth1Data: &ethpb.LatestETH1Data{BlockTime: beforeFiveMinutesAgo}}: "",
		{isRunning: false, runError: errors.New("test runError")}:                                  "",
		// "status is error" cases
		{isRunning: true, runError: errors.New("test runError")}: "test runError",
	}

	for web3ServiceState, wantedErrorText := range testCases {
		status := web3ServiceState.Status()
		if status == nil {
			assert.Equal(t, "", wantedErrorText)

		} else {
			assert.Equal(t, wantedErrorText, status.Error())
		}
	}
}

func TestHandlePanic_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB := dbutil.SetupDB(t)
	server, endpoint, err := mockPOW.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	web3Service, err := NewService(context.Background(),
		WithHttpEndpoints([]string{endpoint}),
		WithDatabase(beaconDB),
	)
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")
	// nil eth1DataFetcher would panic if cached value not used
	web3Service.eth1DataFetcher = nil
	web3Service.processBlockHeader(nil)
	require.LogsContain(t, hook, "Panicked when handling data from ETH 1.0 Chain!")
}

func TestLogTillGenesis_OK(t *testing.T) {
	// Reset the var at the end of the test.
	currPeriod := logPeriod
	logPeriod = 1 * time.Second
	defer func() {
		logPeriod = currPeriod
	}()

	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig()
	cfg.Eth1FollowDistance = 5
	params.OverrideBeaconConfig(cfg)

	nCfg := params.BeaconNetworkConfig()
	nCfg.ContractDeploymentBlock = 0
	params.OverrideBeaconNetworkConfig(nCfg)

	hook := logTest.NewGlobal()
	testAcc, err := mock.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	beaconDB := dbutil.SetupDB(t)
	server, endpoint, err := mockPOW.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	web3Service, err := NewService(context.Background(),
		WithHttpEndpoints([]string{endpoint}),
		WithDepositContractAddress(testAcc.ContractAddr),
		WithDatabase(beaconDB),
	)
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")
	web3Service.depositContractCaller, err = contracts.NewDepositContractCaller(testAcc.ContractAddr, testAcc.Backend)
	require.NoError(t, err)

	web3Service.rpcClient = &mockPOW.RPCClient{Backend: testAcc.Backend}
	web3Service.eth1DataFetcher = &goodFetcher{backend: testAcc.Backend}
	web3Service.httpLogger = testAcc.Backend
	for i := 0; i < 30; i++ {
		testAcc.Backend.Commit()
	}
	web3Service.latestEth1Data = &ethpb.LatestETH1Data{LastRequestedBlock: 0}
	// Spin off to a separate routine
	go web3Service.run(web3Service.ctx.Done())
	// Wait for 2 seconds so that the
	// info is logged.
	time.Sleep(2 * time.Second)
	web3Service.cancel()
	assert.LogsContain(t, hook, "Currently waiting for chainstart")
}

func TestInitDepositCache_OK(t *testing.T) {
	ctrs := []*ethpb.DepositContainer{
		{Index: 0, Eth1BlockHeight: 2, Deposit: &ethpb.Deposit{Proof: [][]byte{[]byte("A")}, Data: &ethpb.Deposit_Data{PublicKey: []byte{}}}},
		{Index: 1, Eth1BlockHeight: 4, Deposit: &ethpb.Deposit{Proof: [][]byte{[]byte("B")}, Data: &ethpb.Deposit_Data{PublicKey: []byte{}}}},
		{Index: 2, Eth1BlockHeight: 6, Deposit: &ethpb.Deposit{Proof: [][]byte{[]byte("c")}, Data: &ethpb.Deposit_Data{PublicKey: []byte{}}}},
	}
	gs, _ := util.DeterministicGenesisState(t, 1)
	beaconDB := dbutil.SetupDB(t)
	s := &Service{
		chainStartData:  &ethpb.ChainStartData{Chainstarted: false},
		preGenesisState: gs,
		cfg:             &config{beaconDB: beaconDB},
	}
	var err error
	s.cfg.depositCache, err = depositcache.New()
	require.NoError(t, err)
	require.NoError(t, s.initDepositCaches(context.Background(), ctrs))

	require.Equal(t, 0, len(s.cfg.depositCache.PendingContainers(context.Background(), nil)))

	blockRootA := [32]byte{'a'}

	emptyState, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, s.cfg.beaconDB.SaveGenesisBlockRoot(context.Background(), blockRootA))
	require.NoError(t, s.cfg.beaconDB.SaveState(context.Background(), emptyState, blockRootA))
	s.chainStartData.Chainstarted = true
	require.NoError(t, s.initDepositCaches(context.Background(), ctrs))
	require.Equal(t, 3, len(s.cfg.depositCache.PendingContainers(context.Background(), nil)))
}

func TestInitDepositCacheWithFinalization_OK(t *testing.T) {
	ctrs := []*ethpb.DepositContainer{
		{
			Index:           0,
			Eth1BlockHeight: 2,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{0}, 48),
					WithdrawalCredentials: make([]byte, 32),
					Signature:             make([]byte, 96),
				},
			},
		},
		{
			Index:           1,
			Eth1BlockHeight: 4,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{1}, 48),
					WithdrawalCredentials: make([]byte, 32),
					Signature:             make([]byte, 96),
				},
			},
		},
		{
			Index:           2,
			Eth1BlockHeight: 6,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{2}, 48),
					WithdrawalCredentials: make([]byte, 32),
					Signature:             make([]byte, 96),
				},
			},
		},
	}
	gs, _ := util.DeterministicGenesisState(t, 1)
	beaconDB := dbutil.SetupDB(t)
	s := &Service{
		chainStartData:  &ethpb.ChainStartData{Chainstarted: false},
		preGenesisState: gs,
		cfg:             &config{beaconDB: beaconDB},
	}
	var err error
	s.cfg.depositCache, err = depositcache.New()
	require.NoError(t, err)
	require.NoError(t, s.initDepositCaches(context.Background(), ctrs))

	require.Equal(t, 0, len(s.cfg.depositCache.PendingContainers(context.Background(), nil)))

	headBlock := util.NewBeaconBlock()
	headRoot, err := headBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	stateGen := stategen.New(beaconDB)

	emptyState, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, s.cfg.beaconDB.SaveGenesisBlockRoot(context.Background(), headRoot))
	require.NoError(t, s.cfg.beaconDB.SaveState(context.Background(), emptyState, headRoot))
	require.NoError(t, stateGen.SaveState(context.Background(), headRoot, emptyState))
	s.cfg.stateGen = stateGen
	require.NoError(t, emptyState.SetEth1DepositIndex(2))

	ctx := context.Background()
	require.NoError(t, beaconDB.SaveFinalizedCheckpoint(ctx, &ethpb.Checkpoint{Epoch: slots.ToEpoch(0), Root: headRoot[:]}))
	s.cfg.finalizedStateAtStartup = emptyState

	s.chainStartData.Chainstarted = true
	require.NoError(t, s.initDepositCaches(context.Background(), ctrs))

	deps := s.cfg.depositCache.NonFinalizedDeposits(context.Background(), nil)
	assert.Equal(t, 0, len(deps))
}

func TestNewService_EarliestVotingBlock(t *testing.T) {
	testAcc, err := mock.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	beaconDB := dbutil.SetupDB(t)
	server, endpoint, err := mockPOW.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	web3Service, err := NewService(context.Background(),
		WithHttpEndpoints([]string{endpoint}),
		WithDepositContractAddress(testAcc.ContractAddr),
		WithDatabase(beaconDB),
	)
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")
	web3Service.eth1DataFetcher = &goodFetcher{backend: testAcc.Backend}
	// simulated backend sets eth1 block
	// time as 10 seconds
	params.SetupTestConfigCleanup(t)
	conf := params.BeaconConfig()
	conf.SecondsPerETH1Block = 10
	conf.Eth1FollowDistance = 50
	params.OverrideBeaconConfig(conf)

	// Genesis not set
	followBlock := uint64(2000)
	blk, err := web3Service.determineEarliestVotingBlock(context.Background(), followBlock)
	require.NoError(t, err)
	assert.Equal(t, followBlock-conf.Eth1FollowDistance, blk, "unexpected earliest voting block")

	// Genesis is set.

	numToForward := 1500
	// forward 1500 blocks
	for i := 0; i < numToForward; i++ {
		testAcc.Backend.Commit()
	}
	currTime := testAcc.Backend.Blockchain().CurrentHeader().Time
	now := time.Now()
	err = testAcc.Backend.AdjustTime(now.Sub(time.Unix(int64(currTime), 0)))
	require.NoError(t, err)
	testAcc.Backend.Commit()

	currTime = testAcc.Backend.Blockchain().CurrentHeader().Time
	web3Service.latestEth1Data.BlockHeight = testAcc.Backend.Blockchain().CurrentHeader().Number.Uint64()
	web3Service.latestEth1Data.BlockTime = testAcc.Backend.Blockchain().CurrentHeader().Time
	web3Service.chainStartData.GenesisTime = currTime

	// With a current slot of zero, only request follow_blocks behind.
	blk, err = web3Service.determineEarliestVotingBlock(context.Background(), followBlock)
	require.NoError(t, err)
	assert.Equal(t, followBlock-conf.Eth1FollowDistance, blk, "unexpected earliest voting block")

}

func TestNewService_Eth1HeaderRequLimit(t *testing.T) {
	testAcc, err := mock.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	beaconDB := dbutil.SetupDB(t)

	server, endpoint, err := mockPOW.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	s1, err := NewService(context.Background(),
		WithHttpEndpoints([]string{endpoint}),
		WithDepositContractAddress(testAcc.ContractAddr),
		WithDatabase(beaconDB),
	)
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")
	assert.Equal(t, defaultEth1HeaderReqLimit, s1.cfg.eth1HeaderReqLimit, "default eth1 header request limit not set")
	s2, err := NewService(context.Background(),
		WithHttpEndpoints([]string{endpoint}),
		WithDepositContractAddress(testAcc.ContractAddr),
		WithDatabase(beaconDB),
		WithEth1HeaderRequestLimit(uint64(150)),
	)
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")
	assert.Equal(t, uint64(150), s2.cfg.eth1HeaderReqLimit, "unable to set eth1HeaderRequestLimit")
}

type mockBSUpdater struct {
	lastBS clientstats.BeaconNodeStats
}

func (mbs *mockBSUpdater) Update(bs clientstats.BeaconNodeStats) {
	mbs.lastBS = bs
}

var _ BeaconNodeStatsUpdater = &mockBSUpdater{}

func TestServiceFallbackCorrectly(t *testing.T) {
	testAcc, err := mock.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	beaconDB := dbutil.SetupDB(t)

	server, firstEndpoint, err := mockPOW.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	server2, secondEndpoint, err := mockPOW.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server2.Stop()
	})
	server3, thirdEndpoint, err := mockPOW.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server3.Stop()
	})

	mbs := &mockBSUpdater{}
	s1, err := NewService(context.Background(),
		WithHttpEndpoints([]string{firstEndpoint}),
		WithDepositContractAddress(testAcc.ContractAddr),
		WithDatabase(beaconDB),
		WithBeaconNodeStatsUpdater(mbs),
	)
	require.NoError(t, err)
	s1.cfg.beaconNodeStatsUpdater = mbs

	assert.Equal(t, firstEndpoint, s1.cfg.currHttpEndpoint.Url, "Unexpected http endpoint")

	// Stay at the first endpoint.
	s1.fallbackToNextEndpoint()
	assert.Equal(t, firstEndpoint, s1.cfg.currHttpEndpoint.Url, "Unexpected http endpoint")
	assert.Equal(t, false, mbs.lastBS.SyncEth1FallbackConfigured, "SyncEth1FallbackConfigured in clientstats update should be false when only 1 endpoint is configured")

	s1.cfg.httpEndpoints = append(s1.cfg.httpEndpoints, network.Endpoint{Url: secondEndpoint})

	s1.fallbackToNextEndpoint()
	assert.Equal(t, secondEndpoint, s1.cfg.currHttpEndpoint.Url, "Unexpected http endpoint")
	assert.Equal(t, true, mbs.lastBS.SyncEth1FallbackConfigured, "SyncEth1FallbackConfigured in clientstats update should be true when > 1 endpoint is configured")

	s1.cfg.httpEndpoints = append(s1.cfg.httpEndpoints, network.Endpoint{Url: thirdEndpoint})

	s1.fallbackToNextEndpoint()
	assert.Equal(t, thirdEndpoint, s1.cfg.currHttpEndpoint.Url, "Unexpected http endpoint")

	// Rollover correctly back to the first endpoint
	s1.fallbackToNextEndpoint()
	assert.Equal(t, firstEndpoint, s1.cfg.currHttpEndpoint.Url, "Unexpected http endpoint")
}

func TestDedupEndpoints(t *testing.T) {
	assert.DeepEqual(t, []string{"A"}, dedupEndpoints([]string{"A"}), "did not dedup correctly")
	assert.DeepEqual(t, []string{"A", "B"}, dedupEndpoints([]string{"A", "B"}), "did not dedup correctly")
	assert.DeepEqual(t, []string{"A", "B"}, dedupEndpoints([]string{"A", "A", "A", "B"}), "did not dedup correctly")
	assert.DeepEqual(t, []string{"A", "B"}, dedupEndpoints([]string{"A", "A", "A", "B", "B"}), "did not dedup correctly")
}

func Test_batchRequestHeaders_UnderflowChecks(t *testing.T) {
	srv := &Service{}
	start := uint64(101)
	end := uint64(100)
	_, err := srv.batchRequestHeaders(start, end)
	require.ErrorContains(t, "cannot be >", err)

	start = uint64(200)
	end = uint64(100)
	_, err = srv.batchRequestHeaders(start, end)
	require.ErrorContains(t, "cannot be >", err)
}

func TestService_EnsureConsistentPowchainData(t *testing.T) {
	beaconDB := dbutil.SetupDB(t)
	cache, err := depositcache.New()
	require.NoError(t, err)
	srv, endpoint, err := mockPOW.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		srv.Stop()
	})
	s1, err := NewService(context.Background(),
		WithHttpEndpoints([]string{endpoint}),
		WithDatabase(beaconDB),
		WithDepositCache(cache),
	)
	require.NoError(t, err)
	genState, err := util.NewBeaconState()
	require.NoError(t, err)
	assert.NoError(t, genState.SetSlot(1000))

	require.NoError(t, s1.cfg.beaconDB.SaveGenesisData(context.Background(), genState))
	require.NoError(t, s1.ensureValidPowchainData(context.Background()))

	eth1Data, err := s1.cfg.beaconDB.PowchainData(context.Background())
	assert.NoError(t, err)

	assert.NotNil(t, eth1Data)
	assert.Equal(t, true, eth1Data.ChainstartData.Chainstarted)
}

func TestService_InitializeCorrectly(t *testing.T) {
	beaconDB := dbutil.SetupDB(t)
	cache, err := depositcache.New()
	require.NoError(t, err)

	srv, endpoint, err := mockPOW.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		srv.Stop()
	})
	s1, err := NewService(context.Background(),
		WithHttpEndpoints([]string{endpoint}),
		WithDatabase(beaconDB),
		WithDepositCache(cache),
	)
	require.NoError(t, err)
	genState, err := util.NewBeaconState()
	require.NoError(t, err)
	assert.NoError(t, genState.SetSlot(1000))

	require.NoError(t, s1.cfg.beaconDB.SaveGenesisData(context.Background(), genState))
	require.NoError(t, s1.ensureValidPowchainData(context.Background()))

	eth1Data, err := s1.cfg.beaconDB.PowchainData(context.Background())
	assert.NoError(t, err)

	assert.NoError(t, s1.initializeEth1Data(context.Background(), eth1Data))
	assert.Equal(t, int64(-1), s1.lastReceivedMerkleIndex, "received incorrect last received merkle index")
}

func TestService_EnsureValidPowchainData(t *testing.T) {
	beaconDB := dbutil.SetupDB(t)
	cache, err := depositcache.New()
	require.NoError(t, err)
	srv, endpoint, err := mockPOW.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		srv.Stop()
	})
	s1, err := NewService(context.Background(),
		WithHttpEndpoints([]string{endpoint}),
		WithDatabase(beaconDB),
		WithDepositCache(cache),
	)
	require.NoError(t, err)
	genState, err := util.NewBeaconState()
	require.NoError(t, err)
	assert.NoError(t, genState.SetSlot(1000))

	require.NoError(t, s1.cfg.beaconDB.SaveGenesisData(context.Background(), genState))

	err = s1.cfg.beaconDB.SavePowchainData(context.Background(), &ethpb.ETH1ChainData{
		ChainstartData:    &ethpb.ChainStartData{Chainstarted: true},
		DepositContainers: []*ethpb.DepositContainer{{Index: 1}},
	})
	require.NoError(t, err)
	require.NoError(t, s1.ensureValidPowchainData(context.Background()))

	eth1Data, err := s1.cfg.beaconDB.PowchainData(context.Background())
	assert.NoError(t, err)

	assert.NotNil(t, eth1Data)
	assert.Equal(t, 0, len(eth1Data.DepositContainers))
}

func TestService_ValidateDepositContainers(t *testing.T) {
	var tt = []struct {
		name        string
		ctrsFunc    func() []*ethpb.DepositContainer
		expectedRes bool
	}{
		{
			name: "zero containers",
			ctrsFunc: func() []*ethpb.DepositContainer {
				return make([]*ethpb.DepositContainer, 0)
			},
			expectedRes: true,
		},
		{
			name: "ordered containers",
			ctrsFunc: func() []*ethpb.DepositContainer {
				ctrs := make([]*ethpb.DepositContainer, 0)
				for i := 0; i < 10; i++ {
					ctrs = append(ctrs, &ethpb.DepositContainer{Index: int64(i), Eth1BlockHeight: uint64(i + 10)})
				}
				return ctrs
			},
			expectedRes: true,
		},
		{
			name: "0th container missing",
			ctrsFunc: func() []*ethpb.DepositContainer {
				ctrs := make([]*ethpb.DepositContainer, 0)
				for i := 1; i < 10; i++ {
					ctrs = append(ctrs, &ethpb.DepositContainer{Index: int64(i), Eth1BlockHeight: uint64(i + 10)})
				}
				return ctrs
			},
			expectedRes: false,
		},
		{
			name: "skipped containers",
			ctrsFunc: func() []*ethpb.DepositContainer {
				ctrs := make([]*ethpb.DepositContainer, 0)
				for i := 0; i < 10; i++ {
					if i == 5 || i == 7 {
						continue
					}
					ctrs = append(ctrs, &ethpb.DepositContainer{Index: int64(i), Eth1BlockHeight: uint64(i + 10)})
				}
				return ctrs
			},
			expectedRes: false,
		},
	}

	for _, test := range tt {
		assert.Equal(t, test.expectedRes, validateDepositContainers(test.ctrsFunc()))
	}
}

func TestTimestampIsChecked(t *testing.T) {
	timestamp := uint64(time.Now().Unix())
	assert.Equal(t, false, eth1HeadIsBehind(timestamp))

	// Give an older timestmap beyond threshold.
	timestamp = uint64(time.Now().Add(-eth1Threshold).Add(-1 * time.Minute).Unix())
	assert.Equal(t, true, eth1HeadIsBehind(timestamp))
}

func TestETH1Endpoints(t *testing.T) {
	server, firstEndpoint, err := mockPOW.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	server, secondEndpoint, err := mockPOW.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	endpoints := []string{firstEndpoint, secondEndpoint}

	testAcc, err := mock.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	beaconDB := dbutil.SetupDB(t)

	mbs := &mockBSUpdater{}
	s1, err := NewService(context.Background(),
		WithHttpEndpoints(endpoints),
		WithDepositContractAddress(testAcc.ContractAddr),
		WithDatabase(beaconDB),
		WithBeaconNodeStatsUpdater(mbs),
	)
	s1.cfg.beaconNodeStatsUpdater = mbs
	require.NoError(t, err)

	// Check default endpoint is set to current.
	assert.Equal(t, firstEndpoint, s1.CurrentETH1Endpoint(), "Unexpected http endpoint")

	// Check endpoints are all present.
	assert.DeepSSZEqual(t, endpoints, s1.ETH1Endpoints(), "Unexpected http endpoint slice")
}
