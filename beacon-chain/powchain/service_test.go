package powchain

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	protodb "github.com/prysmaticlabs/prysm/proto/beacon/db"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
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

func (g *goodFetcher) SyncProgress(_ context.Context) (*ethereum.SyncProgress, error) {
	return nil, nil
}

var depositsReqForChainStart = 64

func TestStart_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB, _ := dbutil.SetupDB(t)
	testAcc, err := contracts.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint:    endpoint,
		DepositContract: testAcc.ContractAddr,
		BeaconDB:        beaconDB,
	})
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

func TestStart_NoHTTPEndpointDefinedFails_WithoutChainStarted(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB, _ := dbutil.SetupDB(t)
	testAcc, err := contracts.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	s, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint:    "", // No endpoint defined!
		DepositContract: testAcc.ContractAddr,
		BeaconDB:        beaconDB,
	})
	require.NoError(t, err)
	// Set custom exit func so test can proceed
	log.Logger.ExitFunc = func(i int) {
		panic(i)
	}
	defer func() {
		log.Logger.ExitFunc = nil
	}()
	wg := new(sync.WaitGroup)
	wg.Add(1)
	// Expect Start function to fail from a fatal call due
	// to no state existing.
	go func() {
		defer func() {
			if r := recover(); r != nil {
				wg.Done()
			}
		}()
		s.Start()
	}()
	testutil.WaitTimeout(wg, time.Second)
	require.LogsContain(t, hook, "cannot create genesis state: no eth1 http endpoint defined")
	hook.Reset()
}

func TestStart_NoHTTPEndpointDefinedSucceeds_WithGenesisState(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB, _ := dbutil.SetupDB(t)
	testAcc, err := contracts.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	st, _ := testutil.DeterministicGenesisState(t, 10)
	b := testutil.NewBeaconBlock()
	genRoot, err := b.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveState(context.Background(), st, genRoot))
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(context.Background(), genRoot))
	s, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint:    "", // No endpoint defined!
		DepositContract: testAcc.ContractAddr,
		BeaconDB:        beaconDB,
	})
	require.NoError(t, err)

	wg := new(sync.WaitGroup)
	wg.Add(1)

	go func() {
		s.Start()
		wg.Done()
	}()
	s.cancel()
	testutil.WaitTimeout(wg, time.Second)
	require.LogsDoNotContain(t, hook, "cannot create genesis state: no eth1 http endpoint defined")
	hook.Reset()
}

func TestStart_NoHTTPEndpointDefinedSucceeds_WithChainStarted(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB, _ := dbutil.SetupDB(t)
	testAcc, err := contracts.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")

	require.NoError(t, beaconDB.SavePowchainData(context.Background(), &protodb.ETH1ChainData{
		ChainstartData: &protodb.ChainStartData{Chainstarted: true},
		Trie:           &protodb.SparseMerkleTrie{},
	}))
	s, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint:    "", // No endpoint defined!
		DepositContract: testAcc.ContractAddr,
		BeaconDB:        beaconDB,
	})
	require.NoError(t, err)

	s.Start()
	require.LogsDoNotContain(t, hook, "cannot create genesis state: no eth1 http endpoint defined")
	hook.Reset()
}

func TestStop_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	testAcc, err := contracts.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	beaconDB, _ := dbutil.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint:    endpoint,
		DepositContract: testAcc.ContractAddr,
		BeaconDB:        beaconDB,
	})
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
	testAcc, err := contracts.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	beaconDB, _ := dbutil.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint:    endpoint,
		DepositContract: testAcc.ContractAddr,
		BeaconDB:        beaconDB,
	})
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")
	web3Service = setDefaultMocks(web3Service)
	web3Service.depositContractCaller, err = contracts.NewDepositContractCaller(testAcc.ContractAddr, testAcc.Backend)
	require.NoError(t, err)

	testAcc.Backend.Commit()

	synced, err := web3Service.isEth1NodeSynced()
	require.NoError(t, err)
	assert.Equal(t, true, synced, "Expected eth1 nodes to be synced")
}

func TestFollowBlock_OK(t *testing.T) {
	testAcc, err := contracts.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	beaconDB, _ := dbutil.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint:    endpoint,
		DepositContract: testAcc.ContractAddr,
		BeaconDB:        beaconDB,
	})
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")

	// simulated backend sets eth1 block
	// time as 10 seconds
	conf := params.BeaconConfig()
	conf.SecondsPerETH1Block = 10
	params.OverrideBeaconConfig(conf)
	defer func() {
		params.UseMainnetConfig()
	}()

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
		{isRunning: true, latestEth1Data: &protodb.LatestETH1Data{BlockTime: afterFiveMinutesAgo}}:   "",
		{isRunning: false, latestEth1Data: &protodb.LatestETH1Data{BlockTime: beforeFiveMinutesAgo}}: "",
		{isRunning: false, runError: errors.New("test runError")}:                                    "",
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
	beaconDB, _ := dbutil.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint: endpoint,
		BeaconDB:     beaconDB,
	})
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

	orgConfig := params.BeaconConfig().Copy()
	cfg := params.BeaconConfig()
	cfg.Eth1FollowDistance = 5
	params.OverrideBeaconConfig(cfg)
	defer func() {
		params.OverrideBeaconConfig(orgConfig)
	}()

	orgNetworkConfig := params.BeaconNetworkConfig().Copy()
	nCfg := params.BeaconNetworkConfig()
	nCfg.ContractDeploymentBlock = 0
	params.OverrideBeaconNetworkConfig(nCfg)
	defer func() {
		params.OverrideBeaconNetworkConfig(orgNetworkConfig)
	}()

	hook := logTest.NewGlobal()
	testAcc, err := contracts.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	beaconDB, _ := dbutil.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint:    endpoint,
		DepositContract: testAcc.ContractAddr,
		BeaconDB:        beaconDB,
	})
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")
	web3Service.depositContractCaller, err = contracts.NewDepositContractCaller(testAcc.ContractAddr, testAcc.Backend)
	require.NoError(t, err)

	web3Service.eth1DataFetcher = &goodFetcher{backend: testAcc.Backend}
	for i := 0; i < 30; i++ {
		testAcc.Backend.Commit()
	}
	web3Service.latestEth1Data = &protodb.LatestETH1Data{LastRequestedBlock: 0}
	// Spin off to a separate routine
	go web3Service.run(web3Service.ctx.Done())
	// Wait for 2 seconds so that the
	// info is logged.
	time.Sleep(2 * time.Second)
	web3Service.cancel()
	assert.LogsContain(t, hook, "Currently waiting for chainstart")
}
