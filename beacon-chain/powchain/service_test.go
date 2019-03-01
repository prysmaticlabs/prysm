package powchain

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/ssz"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
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
			Number: big.NewInt(0),
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

var amount32Eth, _ = new(big.Int).SetString("32000000000000000000", 10)
var depositsReqForChainStart = 8

type testAccount struct {
	addr         common.Address
	contract     *contracts.DepositContract
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
	backend := backends.NewSimulatedBackend(genesis, 2100000000)

	depositsRequired := big.NewInt(int64(depositsReqForChainStart))
	minDeposit := big.NewInt(1e9)
	maxDeposit := big.NewInt(32e9)
	contractAddr, _, contract, err := contracts.DeployDepositContract(
		txOpts,
		backend,
		depositsRequired,
		minDeposit,
		maxDeposit,
		big.NewInt(1),
		addr,
	)
	if err != nil {
		return nil, err
	}

	return &testAccount{addr, contract, contractAddr, backend, txOpts}, nil
}

func TestNewWeb3Service_OK(t *testing.T) {
	endpoint := "http://127.0.0.1"
	ctx := context.Background()
	var err error
	if _, err = NewWeb3Service(ctx, &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: common.Address{},
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
	}); err == nil {
		t.Errorf("passing in an HTTP endpoint should throw an error, received nil")
	}
	endpoint = "ftp://127.0.0.1"
	if _, err = NewWeb3Service(ctx, &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: common.Address{},
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
	}); err == nil {
		t.Errorf("passing in a non-ws, wss, or ipc endpoint should throw an error, received nil")
	}
	endpoint = "ws://127.0.0.1"
	if _, err = NewWeb3Service(ctx, &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: common.Address{},
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
	}); err != nil {
		t.Errorf("passing in as ws endpoint should not throw error, received %v", err)
	}
	endpoint = "ipc://geth.ipc"
	if _, err = NewWeb3Service(ctx, &Web3ServiceConfig{
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

	endpoint := "ws://127.0.0.1"
	testAcc, err := setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: testAcc.contractAddr,
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
		BlockFetcher:    &goodFetcher{},
		ContractBackend: testAcc.backend,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}
	testAcc.backend.Commit()

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

	endpoint := "ws://127.0.0.1"
	testAcc, err := setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: testAcc.contractAddr,
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
		BlockFetcher:    &goodFetcher{},
		ContractBackend: testAcc.backend,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	testAcc.backend.Commit()

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
	endpoint := "ws://127.0.0.1"
	testAcc, err := setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: testAcc.contractAddr,
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
		ContractBackend: testAcc.backend,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	testAcc.backend.Commit()

	if err := web3Service.initDataFromContract(); err != nil {
		t.Fatalf("Could not init from deposit contract: %v", err)
	}

	computedRoot := web3Service.depositTrie.Root()

	if !bytes.Equal(web3Service.depositRoot, computedRoot[:]) {
		t.Errorf("Deposit root is not empty %v", web3Service.depositRoot)
	}

	testAcc.txOpts.Value = amount32Eth
	if _, err := testAcc.contract.Deposit(testAcc.txOpts, []byte{'a'}); err != nil {
		t.Fatalf("Could not deposit to deposit contract %v", err)
	}
	testAcc.backend.Commit()

	if err := web3Service.initDataFromContract(); err != nil {
		t.Fatalf("Could not init from deposit contract: %v", err)
	}

	if bytes.Equal(web3Service.depositRoot, []byte{}) {
		t.Errorf("Deposit root is  empty %v", web3Service.depositRoot)
	}

}

func TestSaveInTrie_OK(t *testing.T) {
	endpoint := "ws://127.0.0.1"
	testAcc, err := setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: testAcc.contractAddr,
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
		ContractBackend: testAcc.backend,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	testAcc.backend.Commit()

	web3Service.depositTrie = trieutil.NewDepositTrie()
	mockTrie := trieutil.NewDepositTrie()
	mockTrie.UpdateDepositTrie([]byte{'A'})

	if err := web3Service.saveInTrie([]byte{'A'}, mockTrie.Root()); err != nil {
		t.Errorf("Unable to save deposit in trie %v", err)
	}

}

func TestWeb3Service_BadReader(t *testing.T) {
	hook := logTest.NewGlobal()
	endpoint := "ws://127.0.0.1"
	testAcc, err := setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: testAcc.contractAddr,
		Reader:          &badReader{},
		Logger:          &goodLogger{},
		ContractBackend: testAcc.backend,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	testAcc.backend.Commit()
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

func TestLatestMainchainInfo_OK(t *testing.T) {
	endpoint := "ws://127.0.0.1"
	testAcc, err := setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: testAcc.contractAddr,
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
		BlockFetcher:    &goodFetcher{},
		ContractBackend: testAcc.backend,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}
	testAcc.backend.Commit()
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

	if web3Service.blockHeight.Cmp(header.Number) != 0 {
		t.Errorf("block number not set, expected %v, got %v", header.Number, web3Service.blockHeight)
	}

	if web3Service.blockHash.Hex() != header.Hash().Hex() {
		t.Errorf("block hash not set, expected %v, got %v", header.Hash().Hex(), web3Service.blockHash.Hex())
	}

	blockInfoExistsInCache, info, err := web3Service.blockCache.BlockInfoByHash(web3Service.blockHash)
	if err != nil {
		t.Fatal(err)
	}
	if !blockInfoExistsInCache {
		t.Error("Expected block info to exist in cache")
	}
	if info.Hash != web3Service.blockHash {
		t.Errorf(
			"Expected block info hash to be %v, got %v",
			web3Service.blockHash,
			info.Hash,
		)
	}
}

func TestProcessDepositLog_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	endpoint := "ws://127.0.0.1"
	testAcc, err := setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: testAcc.contractAddr,
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
		ContractBackend: testAcc.backend,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	testAcc.backend.Commit()

	web3Service.depositTrie = trieutil.NewDepositTrie()

	var stub [48]byte
	copy(stub[:], []byte("testing"))

	data := &pb.DepositInput{
		Pubkey:                      stub[:],
		ProofOfPossession:           stub[:],
		WithdrawalCredentialsHash32: []byte("withdraw"),
	}

	serializedData := new(bytes.Buffer)
	if err := ssz.Encode(serializedData, data); err != nil {
		t.Fatalf("Could not serialize data %v", err)
	}

	testAcc.txOpts.Value = amount32Eth
	if _, err := testAcc.contract.Deposit(testAcc.txOpts, serializedData.Bytes()); err != nil {
		t.Fatalf("Could not deposit to deposit contract %v", err)
	}

	testAcc.backend.Commit()

	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			web3Service.depositContractAddress,
		},
	}

	logs, err := testAcc.backend.FilterLogs(web3Service.ctx, query)
	if err != nil {
		t.Fatalf("Unable to retrieve logs %v", err)
	}

	web3Service.ProcessLog(logs[0])

	testutil.AssertLogsDoNotContain(t, hook, "Could not unpack log")
	testutil.AssertLogsDoNotContain(t, hook, "Could not save in trie")
	testutil.AssertLogsDoNotContain(t, hook, "Could not decode deposit input")
	testutil.AssertLogsContain(t, hook, "Validator registered in deposit contract")

	hook.Reset()
}

func TestProcessDepositLog_InsertsPendingDeposit(t *testing.T) {
	endpoint := "ws://127.0.0.1"
	testAcc, err := setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: testAcc.contractAddr,
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
		ContractBackend: testAcc.backend,
		BeaconDB:        &db.BeaconDB{},
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	testAcc.backend.Commit()

	web3Service.depositTrie = trieutil.NewDepositTrie()

	var stub [48]byte
	copy(stub[:], []byte("testing"))

	data := &pb.DepositInput{
		Pubkey:                      stub[:],
		ProofOfPossession:           stub[:],
		WithdrawalCredentialsHash32: []byte("withdraw"),
	}

	serializedData := new(bytes.Buffer)
	if err := ssz.Encode(serializedData, data); err != nil {
		t.Fatalf("Could not serialize data %v", err)
	}

	testAcc.txOpts.Value = amount32Eth
	if _, err := testAcc.contract.Deposit(testAcc.txOpts, serializedData.Bytes()); err != nil {
		t.Fatalf("Could not deposit to deposit contract %v", err)
	}

	testAcc.backend.Commit()

	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			web3Service.depositContractAddress,
		},
	}

	logs, err := testAcc.backend.FilterLogs(web3Service.ctx, query)
	if err != nil {
		t.Fatalf("Unable to retrieve logs %v", err)
	}

	web3Service.chainStarted = true

	web3Service.ProcessDepositLog(logs[0])
	pendingDeposits := web3Service.beaconDB.PendingDeposits(context.Background(), nil /*blockNum*/)
	if len(pendingDeposits) != 1 {
		t.Errorf("Unexpected number of deposits. Wanted 1 deposit, got %+v", pendingDeposits)
	}
}

func TestProcessDepositLog_SkipDuplicateLog(t *testing.T) {
	endpoint := "ws://127.0.0.1"
	testAcc, err := setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: testAcc.contractAddr,
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
		ContractBackend: testAcc.backend,
		BeaconDB:        &db.BeaconDB{},
	})
	if err != nil {
		t.Fatalf("Unable to setup web3 ETH1.0 chain service: %v", err)
	}

	testAcc.backend.Commit()

	web3Service.depositTrie = trieutil.NewDepositTrie()

	var stub [48]byte
	copy(stub[:], []byte("testing"))

	data := &pb.DepositInput{
		Pubkey:                      stub[:],
		ProofOfPossession:           stub[:],
		WithdrawalCredentialsHash32: []byte("withdraw"),
	}

	serializedData := new(bytes.Buffer)
	if err := ssz.Encode(serializedData, data); err != nil {
		t.Fatalf("Could not serialize data %v", err)
	}

	testAcc.txOpts.Value = amount32Eth
	if _, err := testAcc.contract.Deposit(testAcc.txOpts, serializedData.Bytes()); err != nil {
		t.Fatalf("Could not deposit to deposit contract %v", err)
	}

	testAcc.backend.Commit()

	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			web3Service.depositContractAddress,
		},
	}

	logs, err := testAcc.backend.FilterLogs(web3Service.ctx, query)
	if err != nil {
		t.Fatalf("Unable to retrieve logs %v", err)
	}

	web3Service.ProcessDepositLog(logs[0])
	// We keep track of the current deposit root and make sure it doesn't change if we
	// receive a duplicate log from the contract.
	currentRoot := web3Service.depositTrie.Root()
	web3Service.ProcessDepositLog(logs[0])
	nextRoot := web3Service.depositTrie.Root()
	if currentRoot != nextRoot {
		t.Error("Processing a duplicate log should not update deposit trie")
	}
}

func TestUnpackDepositLogData_OK(t *testing.T) {
	endpoint := "ws://127.0.0.1"
	testAcc, err := setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: testAcc.contractAddr,
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
		ContractBackend: testAcc.backend,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	testAcc.backend.Commit()

	if err := web3Service.initDataFromContract(); err != nil {
		t.Fatalf("Could not init from contract: %v", err)
	}

	computedRoot := web3Service.depositTrie.Root()

	if !bytes.Equal(web3Service.depositRoot, computedRoot[:]) {
		t.Errorf("Deposit root is not equal to computed root Got: %#x , Expected: %#x", web3Service.depositRoot, computedRoot)
	}

	var stub [48]byte
	copy(stub[:], []byte("testing"))

	data := &pb.DepositInput{
		Pubkey:                      stub[:],
		ProofOfPossession:           stub[:],
		WithdrawalCredentialsHash32: []byte("withdraw"),
	}

	serializedData := new(bytes.Buffer)
	if err := ssz.Encode(serializedData, data); err != nil {
		t.Fatalf("Could not serialize data %v", err)
	}

	testAcc.txOpts.Value = amount32Eth
	if _, err := testAcc.contract.Deposit(testAcc.txOpts, serializedData.Bytes()); err != nil {
		t.Fatalf("Could not deposit to deposit contract %v", err)
	}
	testAcc.backend.Commit()

	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			web3Service.depositContractAddress,
		},
	}

	logz, err := testAcc.backend.FilterLogs(web3Service.ctx, query)
	if err != nil {
		t.Fatalf("Unable to retrieve logs %v", err)
	}

	_, depositData, index, _, err := contracts.UnpackDepositLogData(logz[0].Data)
	if err != nil {
		t.Fatalf("Unable to unpack logs %v", err)
	}

	if binary.LittleEndian.Uint64(index) != 0 {
		t.Errorf("Retrieved merkle tree index is incorrect %d", index)
	}

	deserializeData, err := helpers.DecodeDepositInput(depositData)
	if err != nil {
		t.Fatalf("Unable to decode deposit input %v", err)
	}

	if !bytes.Equal(deserializeData.Pubkey, stub[:]) {
		t.Errorf("Pubkey is not the same as the data that was put in %v", deserializeData.Pubkey)
	}

	if !bytes.Equal(deserializeData.ProofOfPossession, stub[:]) {
		t.Errorf("Proof of Possession is not the same as the data that was put in %v", deserializeData.ProofOfPossession)
	}

	if !bytes.Equal(deserializeData.WithdrawalCredentialsHash32, []byte("withdraw")) {
		t.Errorf("Withdrawal Credentials is not the same as the data that was put in %v", deserializeData.WithdrawalCredentialsHash32)
	}

}

func TestProcessChainStartLog_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	endpoint := "ws://127.0.0.1"
	testAcc, err := setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: testAcc.contractAddr,
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
		ContractBackend: testAcc.backend,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	testAcc.backend.Commit()
	testAcc.backend.AdjustTime(time.Duration(int64(time.Now().Nanosecond())))

	web3Service.depositTrie = trieutil.NewDepositTrie()

	var stub [48]byte
	copy(stub[:], []byte("testing"))

	data := &pb.DepositInput{
		Pubkey:                      stub[:],
		ProofOfPossession:           stub[:],
		WithdrawalCredentialsHash32: []byte("withdraw"),
	}

	serializedData := new(bytes.Buffer)
	if err := ssz.Encode(serializedData, data); err != nil {
		t.Fatalf("Could not serialize data %v", err)
	}

	// 8 Validators are used as size required for beacon-chain to start. This number
	// is defined in the deposit contract as the number required for the testnet. The actual number
	// is 2**14
	for i := 0; i < depositsReqForChainStart; i++ {
		testAcc.txOpts.Value = amount32Eth
		if _, err := testAcc.contract.Deposit(testAcc.txOpts, serializedData.Bytes()); err != nil {
			t.Fatalf("Could not deposit to deposit contract %v", err)
		}

		testAcc.backend.Commit()
	}

	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			web3Service.depositContractAddress,
		},
	}

	logs, err := testAcc.backend.FilterLogs(web3Service.ctx, query)
	if err != nil {
		t.Fatalf("Unable to retrieve logs %v", err)
	}

	genesisTimeChan := make(chan time.Time, 1)
	sub := web3Service.chainStartFeed.Subscribe(genesisTimeChan)
	defer sub.Unsubscribe()

	for _, log := range logs {
		web3Service.ProcessLog(log)
	}

	cachedDeposits := web3Service.ChainStartDeposits()
	if len(cachedDeposits) != depositsReqForChainStart {
		t.Errorf(
			"Did not cache the chain start deposits correctly, received %d, wanted %d",
			len(cachedDeposits),
			depositsReqForChainStart,
		)
	}

	<-genesisTimeChan
	testutil.AssertLogsDoNotContain(t, hook, "Unable to unpack ChainStart log data")
	testutil.AssertLogsDoNotContain(t, hook, "Receipt root from log doesn't match the root saved in memory")
	testutil.AssertLogsDoNotContain(t, hook, "Invalid timestamp from log")
	testutil.AssertLogsContain(t, hook, "Minimum number of validators reached for beacon-chain to start")

	hook.Reset()

}

func TestUnpackChainStartLogData_OK(t *testing.T) {
	endpoint := "ws://127.0.0.1"
	testAcc, err := setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: testAcc.contractAddr,
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
		ContractBackend: testAcc.backend,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	testAcc.backend.Commit()

	testAcc.backend.AdjustTime(time.Duration(int64(time.Now().Nanosecond())))

	var stub [48]byte
	copy(stub[:], []byte("testing"))

	data := &pb.DepositInput{
		Pubkey:                      stub[:],
		ProofOfPossession:           stub[:],
		WithdrawalCredentialsHash32: []byte("withdraw"),
	}

	serializedData := new(bytes.Buffer)
	if err := ssz.Encode(serializedData, data); err != nil {
		t.Fatalf("Could not serialize data %v", err)
	}

	// 8 Validators are used as size required for beacon-chain to start. This number
	// is defined in the deposit contract as the number required for the testnet.
	for i := 0; i < depositsReqForChainStart; i++ {
		testAcc.txOpts.Value = amount32Eth
		if _, err := testAcc.contract.Deposit(testAcc.txOpts, serializedData.Bytes()); err != nil {
			t.Fatalf("Could not deposit to deposit contract %v", err)
		}

		testAcc.backend.Commit()
	}
	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			web3Service.depositContractAddress,
		},
	}

	logs, err := testAcc.backend.FilterLogs(web3Service.ctx, query)
	if err != nil {
		t.Fatalf("Unable to retrieve logs %v", err)
	}

	_, timestampData, err := contracts.UnpackChainStartLogData(logs[len(logs)-1].Data)
	if err != nil {
		t.Fatalf("Unable to unpack logs %v", err)
	}

	timestamp := binary.LittleEndian.Uint64(timestampData)

	if timestamp > uint64(time.Now().Unix()) {
		t.Errorf("Timestamp from log is higher than the current time %d > %d", timestamp, time.Now().Unix())
	}
}

func TestHasChainStartLogOccurred_OK(t *testing.T) {
	endpoint := "ws://127.0.0.1"
	testAcc, err := setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: testAcc.contractAddr,
		Reader:          &goodReader{},
		Logger:          testAcc.backend,
		ContractBackend: testAcc.backend,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	testAcc.backend.Commit()

	testAcc.backend.AdjustTime(time.Duration(int64(time.Now().Nanosecond())))

	var stub [48]byte
	copy(stub[:], []byte("testing"))

	data := &pb.DepositInput{
		Pubkey:                      stub[:],
		ProofOfPossession:           stub[:],
		WithdrawalCredentialsHash32: []byte("withdraw"),
	}

	serializedData := new(bytes.Buffer)
	if err := ssz.Encode(serializedData, data); err != nil {
		t.Fatalf("Could not serialize data %v", err)
	}
	ok, _, err := web3Service.HasChainStartLogOccurred()
	if err != nil {
		t.Fatalf("Could not check if chain start log occurred: %v", err)
	}
	if ok {
		t.Error("Expected chain start log to not have occurred")
	}

	// 8 Validators are used as size required for beacon-chain to start. This number
	// is defined in the deposit contract as the number required for the testnet.
	for i := 0; i < depositsReqForChainStart; i++ {
		testAcc.txOpts.Value = amount32Eth
		if _, err := testAcc.contract.Deposit(testAcc.txOpts, serializedData.Bytes()); err != nil {
			t.Fatalf("Could not deposit to deposit contract %v", err)
		}
		testAcc.backend.Commit()
	}
	ok, _, err = web3Service.HasChainStartLogOccurred()
	if err != nil {
		t.Fatalf("Could not check if chain start log occurred: %v", err)
	}
	if !ok {
		t.Error("Expected chain start log to have occurred")
	}
}

func TestBlockHashByHeight_ReturnsHash(t *testing.T) {
	endpoint := "ws://127.0.0.1"
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:     endpoint,
		BlockFetcher: &goodFetcher{},
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}
	ctx := context.Background()

	block := gethTypes.NewBlock(
		&gethTypes.Header{
			Number: big.NewInt(0),
		},
		[]*gethTypes.Transaction{},
		[]*gethTypes.Header{},
		[]*gethTypes.Receipt{},
	)
	wanted := block.Hash()

	hash, err := web3Service.BlockHashByHeight(ctx, big.NewInt(0))
	if err != nil {
		t.Fatalf("Could not get block hash with given height %v", err)
	}

	if !bytes.Equal(hash.Bytes(), wanted.Bytes()) {
		t.Fatalf("Block hash did not equal expected hash, expected: %v, got: %v", wanted, hash)
	}

	exists, _, err := web3Service.blockCache.BlockInfoByHash(wanted)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("Expected block info to be cached")
	}
}

func TestBlockExists_ValidHash(t *testing.T) {
	endpoint := "ws://127.0.0.1"
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:     endpoint,
		BlockFetcher: &goodFetcher{},
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	block := gethTypes.NewBlock(
		&gethTypes.Header{
			Number: big.NewInt(0),
		},
		[]*gethTypes.Transaction{},
		[]*gethTypes.Header{},
		[]*gethTypes.Receipt{},
	)

	exists, height, err := web3Service.BlockExists(context.Background(), block.Hash())
	if err != nil {
		t.Fatalf("Could not get block hash with given height %v", err)
	}

	if !exists {
		t.Fatal("Expected BlockExists to return true.")
	}
	if height.Cmp(block.Number()) != 0 {
		t.Fatalf("Block height did not equal expected height, expected: %v, got: %v", big.NewInt(42), height)
	}

	exists, _, err = web3Service.blockCache.BlockInfoByHeight(height)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("Expected block to be cached")
	}
}

func TestBlockExists_InvalidHash(t *testing.T) {
	endpoint := "ws://127.0.0.1"
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:     endpoint,
		BlockFetcher: &goodFetcher{},
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	_, _, err = web3Service.BlockExists(context.Background(), common.BytesToHash([]byte{0}))
	if err == nil {
		t.Fatal("Expected BlockExists to error with invalid hash")
	}
}

func TestBlockExists_UsesCachedBlockInfo(t *testing.T) {
	endpoint := "ws://127.0.0.1"
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:     endpoint,
		BlockFetcher: nil, // nil blockFetcher would panic if cached value not used
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	block := gethTypes.NewBlock(
		&gethTypes.Header{
			Number: big.NewInt(0),
		},
		[]*gethTypes.Transaction{},
		[]*gethTypes.Header{},
		[]*gethTypes.Receipt{},
	)

	if err := web3Service.blockCache.AddBlock(block); err != nil {
		t.Fatal(err)
	}

	exists, height, err := web3Service.BlockExists(context.Background(), block.Hash())
	if err != nil {
		t.Fatalf("Could not get block hash with given height %v", err)
	}

	if !exists {
		t.Fatal("Expected BlockExists to return true.")
	}
	if height.Cmp(block.Number()) != 0 {
		t.Fatalf("Block height did not equal expected height, expected: %v, got: %v", big.NewInt(42), height)
	}
}
