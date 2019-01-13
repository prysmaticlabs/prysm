package powchain

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	contracts "github.com/prysmaticlabs/prysm/contracts/validator-registration-contract"
	"github.com/prysmaticlabs/prysm/shared/event"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

type badReader struct{}

func (b *badReader) SubscribeNewHead(ctx context.Context, ch chan<- *gethTypes.Header) (ethereum.Subscription, error) {
	return nil, errors.New("subscription has failed")
}

type goodReader struct{}

func (g *goodReader) SubscribeNewHead(ctx context.Context, ch chan<- *gethTypes.Header) (ethereum.Subscription, error) {
	return new(event.Feed).Subscribe(ch), nil
}

type badLogger struct{}

func (b *badLogger) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]gethTypes.Log, error) {
	return nil, errors.New("unable to retrieve logs")
}

func (b *badLogger) SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- gethTypes.Log) (ethereum.Subscription, error) {
	return nil, errors.New("subscription has failed")
}

type goodLogger struct{}

func (g *goodLogger) SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- gethTypes.Log) (ethereum.Subscription, error) {
	return new(event.Feed).Subscribe(ch), nil
}

func (b *goodLogger) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]gethTypes.Log, error) {
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

var (
	amount33Eth, _        = new(big.Int).SetString("33000000000000000000", 10)
	amount32Eth, _        = new(big.Int).SetString("32000000000000000000", 10)
	amountLessThan1Eth, _ = new(big.Int).SetString("500000000000000000", 10)
)

type testAccount struct {
	addr         common.Address
	contract     *contracts.ValidatorRegistration
	contractAddr common.Address
	backend      *backends.SimulatedBackend
	txOpts       *bind.TransactOpts
}

func setup() (*testAccount, error) {
	genesis := make(core.GenesisAlloc)
	privKey, _ := crypto.GenerateKey()
	pubKeyECDSA, ok := privKey.Public().(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("error casting public key to ECDSA")
	}

	// strip off the 0x and the first 2 characters 04 which is always the EC prefix and is not required.
	publicKeyBytes := crypto.FromECDSAPub(pubKeyECDSA)[4:]
	var pubKey = make([]byte, 48)
	copy(pubKey[:], []byte(publicKeyBytes))

	addr := crypto.PubkeyToAddress(privKey.PublicKey)
	txOpts := bind.NewKeyedTransactor(privKey)
	startingBalance, _ := new(big.Int).SetString("1000000000000000000000", 10)
	genesis[addr] = core.GenesisAccount{Balance: startingBalance}
	backend := backends.NewSimulatedBackend(genesis, 2100000)

	contractAddr, _, contract, err := contracts.DeployValidatorRegistration(txOpts, backend)
	if err != nil {
		return nil, err
	}

	return &testAccount{addr, contract, contractAddr, backend, txOpts}, nil
}

func TestNewWeb3Service(t *testing.T) {
	endpoint := "http://127.0.0.1"
	ctx := context.Background()
	var err error
	if _, err = NewWeb3Service(ctx, &Web3ServiceConfig{
		Endpoint: endpoint,
		VrcAddr:  common.Address{},
		Reader:   &goodReader{},
		Logger:   &goodLogger{},
	}); err == nil {
		t.Errorf("passing in an HTTP endpoint should throw an error, received nil")
	}
	endpoint = "ftp://127.0.0.1"
	if _, err = NewWeb3Service(ctx, &Web3ServiceConfig{
		Endpoint: endpoint,
		VrcAddr:  common.Address{},
		Reader:   &goodReader{},
		Logger:   &goodLogger{},
	}); err == nil {
		t.Errorf("passing in a non-ws, wss, or ipc endpoint should throw an error, received nil")
	}
	endpoint = "ws://127.0.0.1"
	if _, err = NewWeb3Service(ctx, &Web3ServiceConfig{
		Endpoint: endpoint,
		VrcAddr:  common.Address{},
		Reader:   &goodReader{},
		Logger:   &goodLogger{},
	}); err != nil {
		t.Errorf("passing in as ws endpoint should not throw error, received %v", err)
	}
	endpoint = "ipc://geth.ipc"
	if _, err = NewWeb3Service(ctx, &Web3ServiceConfig{
		Endpoint: endpoint,
		VrcAddr:  common.Address{},
		Reader:   &goodReader{},
		Logger:   &goodLogger{},
	}); err != nil {
		t.Errorf("passing in an ipc endpoint should not throw error, received %v", err)
	}
}

func TestStart(t *testing.T) {
	hook := logTest.NewGlobal()

	endpoint := "ws://127.0.0.1"
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint: endpoint,
		VrcAddr:  common.Address{},
		Reader:   &goodReader{},
		Logger:   &goodLogger{},
	})
	if err != nil {
		t.Fatalf("unable to setup web3 PoW chain service: %v", err)
	}

	web3Service.Start()

	msg := hook.LastEntry().Message
	want := "Could not connect to PoW chain RPC client"
	if strings.Contains(want, msg) {
		t.Errorf("incorrect log, expected %s, got %s", want, msg)
	}
	hook.Reset()
}

func TestStop(t *testing.T) {
	hook := logTest.NewGlobal()

	endpoint := "ws://127.0.0.1"
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint: endpoint,
		VrcAddr:  common.Address{},
		Reader:   &goodReader{},
		Logger:   &goodLogger{},
	})
	if err != nil {
		t.Fatalf("unable to setup web3 PoW chain service: %v", err)
	}

	if err := web3Service.Stop(); err != nil {
		t.Fatalf("Unable to stop web3 PoW chain service: %v", err)
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

func TestInitDataFromVRC(t *testing.T) {
	//hook := logTest.NewGlobal()
	endpoint := "ws://127.0.0.1"
	testAcc, err := setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:        endpoint,
		VrcAddr:         testAcc.contractAddr,
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
		ContractBackend: testAcc.backend,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 PoW chain service: %v", err)
	}

	testAcc.backend.Commit()

	if err := web3Service.initDataFromVRC(); err != nil {
		t.Fatalf("Could not init from vrc %v", err)
	}

	if web3Service.depositCount != 0 {
		t.Errorf("Deposit count is not equal to zero %d", web3Service.depositCount)
	}

	if !bytes.Equal(web3Service.depositRoot, []byte{}) {
		t.Errorf("Deposit root is not empty %v", web3Service.depositRoot)
	}

	testAcc.txOpts.Value = amount32Eth
	testAcc.contract.Deposit(testAcc.txOpts, []byte{'A'})
	testAcc.backend.Commit()

	if err := web3Service.initDataFromVRC(); err != nil {
		t.Fatalf("Could not init from vrc %v", err)
	}

	if web3Service.depositCount != 1 {
		t.Errorf("Deposit count is not equal to one %d", web3Service.depositCount)
	}

	if bytes.Equal(web3Service.depositRoot, []byte{}) {
		t.Errorf("Deposit root is  empty %v", web3Service.depositRoot)
	}
}

func TestBadReader(t *testing.T) {
	hook := logTest.NewGlobal()
	endpoint := "ws://127.0.0.1"
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint: endpoint,
		VrcAddr:  common.Address{},
		Reader:   &badReader{},
		Logger:   &goodLogger{},
	})
	if err != nil {
		t.Fatalf("unable to setup web3 PoW chain service: %v", err)
	}
	web3Service.reader = &badReader{}
	web3Service.logger = &goodLogger{}
	web3Service.run(web3Service.ctx.Done())
	msg := hook.LastEntry().Message
	want := "Unable to subscribe to incoming PoW chain headers: subscription has failed"
	if msg != want {
		t.Errorf("incorrect log, expected %s, got %s", want, msg)
	}
	hook.Reset()
}

func TestLatestMainchainInfo(t *testing.T) {
	endpoint := "ws://127.0.0.1"
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint: endpoint,
		VrcAddr:  common.Address{},
		Reader:   &goodReader{},
		Logger:   &goodLogger{},
	})
	if err != nil {
		t.Fatalf("unable to setup web3 PoW chain service: %v", err)
	}
	web3Service.reader = &goodReader{}
	web3Service.logger = &goodLogger{}

	exitRoutine := make(chan bool)

	go func() {
		web3Service.run(web3Service.ctx.Done())
		<-exitRoutine
	}()

	header := &gethTypes.Header{Number: big.NewInt(42)}

	web3Service.headerChan <- header
	web3Service.cancel()
	exitRoutine <- true

	if web3Service.blockNumber.Cmp(header.Number) != 0 {
		t.Errorf("block number not set, expected %v, got %v", header.Number, web3Service.blockNumber)
	}

	if web3Service.blockHash.Hex() != header.Hash().Hex() {
		t.Errorf("block hash not set, expected %v, got %v", header.Hash().Hex(), web3Service.blockHash.Hex())
	}
}

func TestBadLogger(t *testing.T) {
	hook := logTest.NewGlobal()
	endpoint := "ws://127.0.0.1"
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint: endpoint,
		VrcAddr:  common.Address{},
		Reader:   &goodReader{},
		Logger:   &goodLogger{},
	})
	if err != nil {
		t.Fatalf("unable to setup web3 PoW chain service: %v", err)
	}
	web3Service.reader = &goodReader{}
	web3Service.logger = &badLogger{}

	web3Service.run(web3Service.ctx.Done())
	msg := hook.LastEntry().Message
	want := "Unable to query logs from VRC: subscription has failed"
	if msg != want {
		t.Errorf("incorrect log, expected %s, got %s", want, msg)
	}
	hook.Reset()
}
