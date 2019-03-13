package powchain

import (
	"bytes"
	"context"
	"encoding/binary"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/ssz"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

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
		BeaconDB:        &db.BeaconDB{},
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	testAcc.backend.Commit()

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
		BeaconDB:        &db.BeaconDB{},
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
