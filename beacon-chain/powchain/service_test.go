package powchain

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	depositcontract "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	protodb "github.com/prysmaticlabs/prysm/proto/beacon/db"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

var _ = ChainStartFetcher(&Service{})
var _ = ChainInfoFetcher(&Service{})
var _ = POWBlockFetcher(&Service{})
var _ = Chain(&Service{})

type badReader struct{}

func (b *badReader) SubscribeNewHead(ctx context.Context, ch chan<- *gethTypes.Header) (ethereum.Subscription, error) {
	return nil, errors.New("subscription has failed")
}

type goodReader struct {
	backend *backends.SimulatedBackend
}

func (g *goodReader) SubscribeNewHead(ctx context.Context, ch chan<- *gethTypes.Header) (ethereum.Subscription, error) {
	if g.backend == nil {
		return new(event.Feed).Subscribe(ch), nil
	}
	headChan := make(chan core.ChainHeadEvent)
	eventSub := g.backend.Blockchain().SubscribeChainHeadEvent(headChan)
	feed := new(event.Feed)
	sub := feed.Subscribe(ch)
	go func() {
		for {
			select {
			case blk := <-headChan:
				feed.Send(blk.Block.Header())
			case <-ctx.Done():
				eventSub.Unsubscribe()
				return
			}
		}
	}()
	return sub, nil
}

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

func (g *goodFetcher) BlockByHash(ctx context.Context, hash common.Hash) (*gethTypes.Block, error) {
	if bytes.Equal(hash.Bytes(), common.BytesToHash([]byte{0}).Bytes()) {
		return nil, fmt.Errorf("expected block hash to be nonzero %v", hash)
	}
	if g.backend == nil {
		return gethTypes.NewBlock(
			&gethTypes.Header{
				Number: big.NewInt(0),
			},
			[]*gethTypes.Transaction{},
			[]*gethTypes.Header{},
			[]*gethTypes.Receipt{},
		), nil
	}
	return g.backend.Blockchain().GetBlockByHash(hash), nil

}

func (g *goodFetcher) BlockByNumber(ctx context.Context, number *big.Int) (*gethTypes.Block, error) {
	if g.backend == nil {
		return gethTypes.NewBlock(
			&gethTypes.Header{
				Number: big.NewInt(15),
				Time:   150,
			},
			[]*gethTypes.Transaction{},
			[]*gethTypes.Header{},
			[]*gethTypes.Receipt{},
		), nil
	}

	return g.backend.Blockchain().GetBlockByNumber(number.Uint64()), nil
}

func (g *goodFetcher) HeaderByNumber(ctx context.Context, number *big.Int) (*gethTypes.Header, error) {
	if g.backend == nil {
		return &gethTypes.Header{
			Number: big.NewInt(0),
		}, nil
	}
	if number == nil {
		return g.backend.Blockchain().CurrentHeader(), nil
	}
	return g.backend.Blockchain().GetHeaderByNumber(number.Uint64()), nil
}

var depositsReqForChainStart = 64

func TestNewWeb3Service_OK(t *testing.T) {
	endpoint := "http://127.0.0.1"
	ctx := context.Background()
	var err error
	beaconDB := dbutil.SetupDB(t)
	if _, err = NewService(ctx, &Web3ServiceConfig{
		ETH1Endpoint:    endpoint,
		DepositContract: common.Address{},
		BeaconDB:        beaconDB,
	}); err == nil {
		t.Errorf("passing in an HTTP endpoint should throw an error, received nil")
	}
	endpoint = "ftp://127.0.0.1"
	if _, err = NewService(ctx, &Web3ServiceConfig{
		ETH1Endpoint:    endpoint,
		DepositContract: common.Address{},
		BeaconDB:        beaconDB,
	}); err == nil {
		t.Errorf("passing in a non-ws, wss, or ipc endpoint should throw an error, received nil")
	}
	endpoint = "ws://127.0.0.1"
	if _, err = NewService(ctx, &Web3ServiceConfig{
		ETH1Endpoint:    endpoint,
		DepositContract: common.Address{},
		BeaconDB:        beaconDB,
	}); err != nil {
		t.Errorf("passing in as ws endpoint should not throw error, received %v", err)
	}
	endpoint = "ipc://geth.ipc"
	if _, err = NewService(ctx, &Web3ServiceConfig{
		ETH1Endpoint:    endpoint,
		DepositContract: common.Address{},
		BeaconDB:        beaconDB,
	}); err != nil {
		t.Errorf("passing in an ipc endpoint should not throw error, received %v", err)
	}
}

func TestStart_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB := dbutil.SetupDB(t)
	testAcc, err := contracts.Setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		ETH1Endpoint:    endpoint,
		DepositContract: testAcc.ContractAddr,
		BeaconDB:        beaconDB,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}
	web3Service = setDefaultMocks(web3Service)
	web3Service.rpcClient = &mockPOW.RPCClient{Backend: testAcc.Backend}
	web3Service.depositContractCaller, err = contracts.NewDepositContractCaller(testAcc.ContractAddr, testAcc.Backend)
	if err != nil {
		t.Fatal(err)
	}
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

func TestStop_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	testAcc, err := contracts.Setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	beaconDB := dbutil.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		ETH1Endpoint:    endpoint,
		DepositContract: testAcc.ContractAddr,
		BeaconDB:        beaconDB,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}
	web3Service = setDefaultMocks(web3Service)
	web3Service.depositContractCaller, err = contracts.NewDepositContractCaller(testAcc.ContractAddr, testAcc.Backend)
	if err != nil {
		t.Fatal(err)
	}

	testAcc.Backend.Commit()

	if err := web3Service.Stop(); err != nil {
		t.Fatalf("Unable to stop web3 ETH1.0 chain service: %v", err)
	}

	// The context should have been canceled.
	if web3Service.ctx.Err() == nil {
		t.Error("context was not canceled")
	}
	hook.Reset()
}

func TestFollowBlock_OK(t *testing.T) {
	depositcontract.Amount32Eth()
	testAcc, err := contracts.Setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	beaconDB := dbutil.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		ETH1Endpoint:    endpoint,
		DepositContract: testAcc.ContractAddr,
		BeaconDB:        beaconDB,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	// simulated backend sets eth1 block
	// time as 10 seconds
	conf := params.BeaconConfig()
	conf.SecondsPerETH1Block = 10
	params.OverrideBeaconConfig(conf)
	defer func() {
		params.UseMainnetConfig()
	}()

	web3Service = setDefaultMocks(web3Service)
	web3Service.blockFetcher = &goodFetcher{backend: testAcc.Backend}
	baseHeight := testAcc.Backend.Blockchain().CurrentBlock().NumberU64()
	// process follow_distance blocks
	for i := 0; i < int(params.BeaconConfig().Eth1FollowDistance); i++ {
		testAcc.Backend.Commit()
	}
	// set current height
	web3Service.latestEth1Data.BlockHeight = testAcc.Backend.Blockchain().CurrentBlock().NumberU64()
	web3Service.latestEth1Data.BlockTime = testAcc.Backend.Blockchain().CurrentBlock().Time()

	h, err := web3Service.followBlockHeight(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if h != baseHeight {
		t.Errorf("Unexpected block height of %d received instead of %d", h, baseHeight)
	}
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
	if err != nil {
		t.Fatal(err)
	}
	if h != expectedHeight {
		t.Errorf("Unexpected block height of %d received instead of %d", h, expectedHeight)
	}
}

func TestInitDataFromContract_OK(t *testing.T) {
	testAcc, err := contracts.Setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	beaconDB := dbutil.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		ETH1Endpoint:    endpoint,
		DepositContract: testAcc.ContractAddr,
		BeaconDB:        beaconDB,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}
	web3Service = setDefaultMocks(web3Service)
	web3Service.depositContractCaller, err = contracts.NewDepositContractCaller(testAcc.ContractAddr, testAcc.Backend)
	if err != nil {
		t.Fatal(err)
	}

	testAcc.Backend.Commit()
	if err := web3Service.initDataFromContract(); err != nil {
		t.Fatalf("Could not init from deposit contract: %v", err)
	}
}

func TestWeb3Service_BadReader(t *testing.T) {
	hook := logTest.NewGlobal()
	depositcontract.Amount32Eth()
	testAcc, err := contracts.Setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	beaconDB := dbutil.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		ETH1Endpoint:    endpoint,
		DepositContract: testAcc.ContractAddr,
		BeaconDB:        beaconDB,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}
	web3Service = setDefaultMocks(web3Service)
	web3Service.depositContractCaller, err = contracts.NewDepositContractCaller(testAcc.ContractAddr, testAcc.Backend)
	if err != nil {
		t.Fatal(err)
	}

	testAcc.Backend.Commit()
	web3Service.reader = &badReader{}
	web3Service.logger = &goodLogger{}
	go web3Service.initPOWService()
	time.Sleep(200 * time.Millisecond)
	web3Service.cancel()
	want := "Unable to subscribe to incoming ETH1.0 chain headers: subscription has failed"
	testutil.AssertLogsContain(t, hook, want)
	hook.Reset()
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
			if wantedErrorText != "" {
				t.Errorf("Wanted: \"%v\", but Status() return nil", wantedErrorText)
			}
		} else {
			if status.Error() != wantedErrorText {
				t.Errorf("Wanted: \"%v\", but Status() return: \"%v\"", wantedErrorText, status.Error())
			}
		}
	}
}

func TestHandlePanic_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB := dbutil.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		ETH1Endpoint: endpoint,
		BeaconDB:     beaconDB,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}
	// nil blockFetcher would panic if cached value not used
	web3Service.blockFetcher = nil

	web3Service.processSubscribedHeaders(nil)
	testutil.AssertLogsContain(t, hook, "Panicked when handling data from ETH 1.0 Chain!")
}
