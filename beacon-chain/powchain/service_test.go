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

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	contracts "github.com/prysmaticlabs/prysm/contracts/validator-registration-contract"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/ssz"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/trie"
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

var amount32Eth, _ = new(big.Int).SetString("32000000000000000000", 10)

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

func TestStart(t *testing.T) {
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
		t.Fatalf("unable to setup web3 PoW chain service: %v", err)
	}
	testAcc.backend.Commit()

	web3Service.Start()

	msg := hook.LastEntry().Message
	want := "Could not connect to PoW chain RPC client"
	if strings.Contains(want, msg) {
		t.Errorf("incorrect log, expected %s, got %s", want, msg)
	}
	hook.Reset()
	web3Service.cancel()
}

func TestStop(t *testing.T) {
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
		t.Fatalf("unable to setup web3 PoW chain service: %v", err)
	}

	testAcc.backend.Commit()

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
	if _, err := testAcc.contract.Deposit(testAcc.txOpts, []byte{'a'}); err != nil {
		t.Fatalf("Could not deposit to VRC %v", err)
	}
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

func TestSaveInTrie(t *testing.T) {
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
		t.Fatalf("unable to setup web3 PoW chain service: %v", err)
	}

	testAcc.backend.Commit()

	web3Service.depositTrie = trie.NewDepositTrie()

	currentRoot := web3Service.depositTrie.Root()

	if err := web3Service.saveInTrie([]byte{'A'}, currentRoot); err != nil {
		t.Errorf("Unable to save deposit in trie %v", err)
	}

}

func TestBadReader(t *testing.T) {
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
		t.Fatalf("unable to setup web3 PoW chain service: %v", err)
	}

	testAcc.backend.Commit()
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
		t.Fatalf("unable to setup web3 PoW chain service: %v", err)
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
		t.Fatalf("unable to setup web3 PoW chain service: %v", err)
	}
	testAcc.backend.Commit()

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

func TestProcessDepositLog(t *testing.T) {
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
		t.Fatalf("unable to setup web3 PoW chain service: %v", err)
	}

	testAcc.backend.Commit()

	web3Service.depositTrie = trie.NewDepositTrie()

	currentRoot := web3Service.depositTrie.Root()

	var stub [48]byte
	copy(stub[:], []byte("testing"))

	data := &pb.DepositInput{
		Pubkey:                      stub[:],
		ProofOfPossession:           stub[:],
		WithdrawalCredentialsHash32: []byte("withdraw"),
		RandaoCommitmentHash32:      []byte("randao"),
		CustodyCommitmentHash32:     []byte("custody"),
	}

	serializedData := new(bytes.Buffer)
	if err := ssz.Encode(serializedData, data); err != nil {
		t.Fatalf("Could not serialize data %v", err)
	}

	testAcc.txOpts.Value = amount32Eth
	if _, err := testAcc.contract.Deposit(testAcc.txOpts, serializedData.Bytes()); err != nil {
		t.Fatalf("Could not deposit to VRC %v", err)
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

	logs[0].Topics[1] = currentRoot

	web3Service.ProcessLog(logs[0])

	testutil.AssertLogsDoNotContain(t, hook, "Could not unpack log")
	testutil.AssertLogsDoNotContain(t, hook, "Could not save in trie")
	testutil.AssertLogsDoNotContain(t, hook, "Could not decode deposit input")
	testutil.AssertLogsContain(t, hook, "Validator registered in VRC with public key and index")

	hook.Reset()
}

func TestUnpackDepositLogs(t *testing.T) {
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

	var stub [48]byte
	copy(stub[:], []byte("testing"))

	data := &pb.DepositInput{
		Pubkey:                      stub[:],
		ProofOfPossession:           stub[:],
		WithdrawalCredentialsHash32: []byte("withdraw"),
		RandaoCommitmentHash32:      []byte("randao"),
		CustodyCommitmentHash32:     []byte("custody"),
	}

	serializedData := new(bytes.Buffer)
	if err := ssz.Encode(serializedData, data); err != nil {
		t.Fatalf("Could not serialize data %v", err)
	}

	testAcc.txOpts.Value = amount32Eth
	if _, err := testAcc.contract.Deposit(testAcc.txOpts, serializedData.Bytes()); err != nil {
		t.Fatalf("Could not deposit to VRC %v", err)
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

	depData, index, err := utils.UnpackDepositLogData(logz[0].Data)
	if err != nil {
		t.Fatalf("Unable to unpack logs %v", err)
	}

	if binary.BigEndian.Uint64(index) != 65536 {
		t.Errorf("Retrieved merkle tree index is incorrect %d", index)
	}

	deserializeData, err := blocks.DecodeDepositInput(depData)
	if err != nil {
		t.Fatalf("Unable to decode deposit input %v", err)
	}

	if !bytes.Equal(deserializeData.Pubkey, stub[:]) {
		t.Errorf("Pubkey is not the same as the data that was put in %v", deserializeData.Pubkey)
	}

	if !bytes.Equal(deserializeData.ProofOfPossession, stub[:]) {
		t.Errorf("Proof of Possession is not the same as the data that was put in %v", deserializeData.ProofOfPossession)
	}

	if !bytes.Equal(deserializeData.CustodyCommitmentHash32, []byte("custody")) {
		t.Errorf("Custody commitment is not the same as the data that was put in %v", deserializeData.CustodyCommitmentHash32)
	}

	if !bytes.Equal(deserializeData.RandaoCommitmentHash32, []byte("randao")) {
		t.Errorf("Randao Commitment is not the same as the data that was put in %v", deserializeData.RandaoCommitmentHash32)
	}

	if !bytes.Equal(deserializeData.WithdrawalCredentialsHash32, []byte("withdraw")) {
		t.Errorf("Withdrawal Credentials is not the same as the data that was put in %v", deserializeData.WithdrawalCredentialsHash32)
	}

}

func TestProcessChainStartLog(t *testing.T) {
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
		t.Fatalf("unable to setup web3 PoW chain service: %v", err)
	}

	testAcc.backend.Commit()
	testAcc.backend.AdjustTime(time.Duration(int64(time.Now().Nanosecond())))

	web3Service.depositTrie = trie.NewDepositTrie()

	currentRoot := web3Service.depositTrie.Root()

	var stub [48]byte
	copy(stub[:], []byte("testing"))

	data := &pb.DepositInput{
		Pubkey:                      stub[:],
		ProofOfPossession:           stub[:],
		WithdrawalCredentialsHash32: []byte("withdraw"),
		RandaoCommitmentHash32:      []byte("randao"),
		CustodyCommitmentHash32:     []byte("custody"),
	}

	serializedData := new(bytes.Buffer)
	if err := ssz.Encode(serializedData, data); err != nil {
		t.Fatalf("Could not serialize data %v", err)
	}

	// 8 Validators are used as size required for beacon-chain to start. This number
	// is defined in the VRC as the number required for the testnet. The actual number
	// is 2**14
	for i := 0; i < 8; i++ {
		testAcc.txOpts.Value = amount32Eth
		if _, err := testAcc.contract.Deposit(testAcc.txOpts, serializedData.Bytes()); err != nil {
			t.Fatalf("Could not deposit to VRC %v", err)
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

	logs[len(logs)-1].Topics[1] = currentRoot

	web3Service.ProcessLog(logs[len(logs)-1])

	testutil.AssertLogsDoNotContain(t, hook, "Unable to unpack ChainStart log data")
	testutil.AssertLogsDoNotContain(t, hook, "Receipt root from log doesn't match the root saved in memory")
	testutil.AssertLogsDoNotContain(t, hook, "Invalid timestamp from log")
	testutil.AssertLogsContain(t, hook, "Minimum Number of Validators Reached for beacon-chain to start")

	hook.Reset()

}

func TestUnpackChainStartLogs(t *testing.T) {
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
		t.Fatalf("unable to setup web3 PoW chain service: %v", err)
	}

	testAcc.backend.Commit()

	testAcc.backend.AdjustTime(time.Duration(int64(time.Now().Nanosecond())))

	var stub [48]byte
	copy(stub[:], []byte("testing"))

	data := &pb.DepositInput{
		Pubkey:                      stub[:],
		ProofOfPossession:           stub[:],
		WithdrawalCredentialsHash32: []byte("withdraw"),
		RandaoCommitmentHash32:      []byte("randao"),
		CustodyCommitmentHash32:     []byte("custody"),
	}

	serializedData := new(bytes.Buffer)
	if err := ssz.Encode(serializedData, data); err != nil {
		t.Fatalf("Could not serialize data %v", err)
	}

	// 8 Validators are used as size required for beacon-chain to start. This number
	// is defined in the VRC as the number required for the testnet.
	for i := 0; i < 8; i++ {
		testAcc.txOpts.Value = amount32Eth
		if _, err := testAcc.contract.Deposit(testAcc.txOpts, serializedData.Bytes()); err != nil {
			t.Fatalf("Could not deposit to VRC %v", err)
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

	timestampData, err := utils.UnpackChainStartLogData(logs[len(logs)-1].Data)
	if err != nil {
		t.Fatalf("Unable to unpack logs %v", err)
	}

	timestamp := binary.BigEndian.Uint64(timestampData)

	if timestamp > uint64(time.Now().Unix()) {
		t.Errorf("Timestamp from log is higher than the current time %d > %d", timestamp, time.Now().Unix())
	}

}
