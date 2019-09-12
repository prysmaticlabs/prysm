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
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	depositcontract "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	"github.com/prysmaticlabs/prysm/shared/event"
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

type goodReader struct{}

func (g *goodReader) SubscribeNewHead(ctx context.Context, ch chan<- *gethTypes.Header) (ethereum.Subscription, error) {
	return new(event.Feed).Subscribe(ch), nil
}

type goodLogger struct{}

func (g *goodLogger) SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- gethTypes.Log) (ethereum.Subscription, error) {
	return new(event.Feed).Subscribe(ch), nil
}

func (g *goodLogger) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]gethTypes.Log, error) {
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

type goodFetcher struct{}

func (g *goodFetcher) BlockByHash(ctx context.Context, hash common.Hash) (*gethTypes.Block, error) {
	if bytes.Equal(hash.Bytes(), common.BytesToHash([]byte{0}).Bytes()) {
		return nil, fmt.Errorf("expected block hash to be nonzero %v", hash)
	}

	block := gethTypes.NewBlock(
		&gethTypes.Header{
			Number: big.NewInt(0),
		},
		[]*gethTypes.Transaction{},
		[]*gethTypes.Header{},
		[]*gethTypes.Receipt{},
	)

	return block, nil
}

func (g *goodFetcher) BlockByNumber(ctx context.Context, number *big.Int) (*gethTypes.Block, error) {
	block := gethTypes.NewBlock(
		&gethTypes.Header{
			Number: big.NewInt(15),
			Time:   150,
		},
		[]*gethTypes.Transaction{},
		[]*gethTypes.Header{},
		[]*gethTypes.Receipt{},
	)

	return block, nil
}

func (g *goodFetcher) HeaderByNumber(ctx context.Context, number *big.Int) (*gethTypes.Header, error) {
	return &gethTypes.Header{
		Number: big.NewInt(0),
	}, nil
}

var depositsReqForChainStart = 64

func TestNewWeb3Service_OK(t *testing.T) {
	endpoint := "http://127.0.0.1"
	ctx := context.Background()
	var err error
	if _, err = NewService(ctx, &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: common.Address{},
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
	}); err == nil {
		t.Errorf("passing in an HTTP endpoint should throw an error, received nil")
	}
	endpoint = "ftp://127.0.0.1"
	if _, err = NewService(ctx, &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: common.Address{},
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
	}); err == nil {
		t.Errorf("passing in a non-ws, wss, or ipc endpoint should throw an error, received nil")
	}
	endpoint = "ws://127.0.0.1"
	if _, err = NewService(ctx, &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: common.Address{},
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
	}); err != nil {
		t.Errorf("passing in as ws endpoint should not throw error, received %v", err)
	}
	endpoint = "ipc://geth.ipc"
	if _, err = NewService(ctx, &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: common.Address{},
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
	}); err != nil {
		t.Errorf("passing in an ipc endpoint should not throw error, received %v", err)
	}
}

func TestStart_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, beaconDB)
	testAcc, err := contracts.Setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: testAcc.ContractAddr,
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
		HTTPLogger:      &goodLogger{},
		BlockFetcher:    &goodFetcher{},
		ContractBackend: testAcc.Backend,
		BeaconDB:        beaconDB,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}
	testAcc.Backend.Commit()

	web3Service.Start()

	msg := hook.LastEntry().Message
	want := "Could not connect to ETH1.0 chain RPC client"
	if strings.Contains(want, msg) {
		t.Errorf("incorrect log, expected %s, got %s", want, msg)
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
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: testAcc.ContractAddr,
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
		BlockFetcher:    &goodFetcher{},
		ContractBackend: testAcc.Backend,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	testAcc.Backend.Commit()

	if err := web3Service.Stop(); err != nil {
		t.Fatalf("Unable to stop web3 ETH1.0 chain service: %v", err)
	}

	msg := hook.LastEntry().Message
	want := "Stopping service"
	if msg != want {
		t.Errorf("incorrect log, expected %s, got %s", want, msg)
	}

	// The context should have been canceled.
	if web3Service.ctx.Err() == nil {
		t.Error("context was not canceled")
	}
	hook.Reset()
}

func TestInitDataFromContract_OK(t *testing.T) {

	testAcc, err := contracts.Setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: testAcc.ContractAddr,
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
		HTTPLogger:      &goodLogger{},
		ContractBackend: testAcc.Backend,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
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
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: testAcc.ContractAddr,
		Reader:          &badReader{},
		Logger:          &goodLogger{},
		HTTPLogger:      &goodLogger{},
		ContractBackend: testAcc.Backend,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	testAcc.Backend.Commit()
	web3Service.reader = &badReader{}
	web3Service.logger = &goodLogger{}
	web3Service.run(web3Service.ctx.Done())
	msg := hook.LastEntry().Message
	want := "Unable to subscribe to incoming ETH1.0 chain headers: subscription has failed"
	if msg != want {
		t.Errorf("incorrect log, expected %s, got %s", want, msg)
	}
	hook.Reset()
}

func TestStatus(t *testing.T) {
	now := time.Now()

	beforeFiveMinutesAgo := now.Add(-5*time.Minute - 30*time.Second)
	afterFiveMinutesAgo := now.Add(-5*time.Minute + 30*time.Second)

	testCases := map[*Service]string{
		// "status is ok" cases
		{}: "",
		{isRunning: true, blockTime: afterFiveMinutesAgo}:         "",
		{isRunning: false, blockTime: beforeFiveMinutesAgo}:       "",
		{isRunning: false, runError: errors.New("test runError")}: "",
		// "status is error" cases
		{isRunning: true, blockTime: beforeFiveMinutesAgo}: "eth1 client is not syncing",
		{isRunning: true}: "eth1 client is not syncing",
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

	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		Endpoint:     endpoint,
		BlockFetcher: nil, // nil blockFetcher would panic if cached value not used
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	web3Service.processSubscribedHeaders(nil)
	testutil.AssertLogsContain(t, hook, "Panicked when handling data from ETH 1.0 Chain!")
}
