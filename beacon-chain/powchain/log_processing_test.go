package powchain

import (
	"bytes"
	"context"
	"encoding/binary"
	"io/ioutil"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

func TestProcessDepositLog_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	testutil.ResetCache()
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
		BeaconDB:        &kv.Store{},
		DepositCache:    depositcache.NewDepositCache(),
		BlockFetcher:    &goodFetcher{},
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	testAcc.Backend.Commit()
	deposits, _ := testutil.SetupInitialDeposits(t, 1)
	data := deposits[0].Data

	testAcc.TxOpts.Value = contracts.Amount32Eth()
	testAcc.TxOpts.GasLimit = 1000000
	if _, err := testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature); err != nil {
		t.Fatalf("Could not deposit to deposit contract %v", err)
	}

	testAcc.Backend.Commit()

	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			web3Service.depositContractAddress,
		},
	}

	logs, err := testAcc.Backend.FilterLogs(web3Service.ctx, query)
	if err != nil {
		t.Fatalf("Unable to retrieve logs %v", err)
	}
	if len(logs) == 0 {
		t.Fatal("no logs")
	}

	web3Service.ProcessLog(context.Background(), logs[0])

	testutil.AssertLogsDoNotContain(t, hook, "Could not unpack log")
	testutil.AssertLogsDoNotContain(t, hook, "Could not save in trie")
	testutil.AssertLogsDoNotContain(t, hook, "could not deserialize validator public key")
	testutil.AssertLogsDoNotContain(t, hook, "could not convert bytes to signature")
	testutil.AssertLogsDoNotContain(t, hook, "could not sign root for deposit data")
	testutil.AssertLogsDoNotContain(t, hook, "deposit signature did not verify")
	testutil.AssertLogsDoNotContain(t, hook, "could not tree hash deposit data")
	testutil.AssertLogsDoNotContain(t, hook, "deposit merkle branch of deposit root did not verify for root")
	testutil.AssertLogsContain(t, hook, "Deposit registered from deposit contract")

	hook.Reset()
}

func TestProcessDepositLog_InsertsPendingDeposit(t *testing.T) {
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
		HTTPLogger:      &goodLogger{},
		ContractBackend: testAcc.Backend,
		BeaconDB:        &kv.Store{},
		DepositCache:    depositcache.NewDepositCache(),
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	testAcc.Backend.Commit()

	var pubkey [48]byte
	var withdrawalCreds [32]byte
	var sig [96]byte
	copy(pubkey[:], []byte("testing"))
	copy(sig[:], []byte("testing"))
	copy(withdrawalCreds[:], []byte("testing"))

	data := &ethpb.Deposit_Data{
		PublicKey:             pubkey[:],
		Signature:             sig[:],
		WithdrawalCredentials: withdrawalCreds[:],
	}

	testAcc.TxOpts.Value = contracts.Amount32Eth()
	testAcc.TxOpts.GasLimit = 1000000

	if _, err := testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature); err != nil {
		t.Fatalf("Could not deposit to deposit contract %v", err)
	}

	if _, err := testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature); err != nil {
		t.Fatalf("Could not deposit to deposit contract %v", err)
	}

	testAcc.Backend.Commit()

	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			web3Service.depositContractAddress,
		},
	}

	logs, err := testAcc.Backend.FilterLogs(web3Service.ctx, query)
	if err != nil {
		t.Fatalf("Unable to retrieve logs %v", err)
	}

	web3Service.chainStarted = true

	web3Service.ProcessDepositLog(context.Background(), logs[0])
	web3Service.ProcessDepositLog(context.Background(), logs[1])
	pendingDeposits := web3Service.depositCache.PendingDeposits(context.Background(), nil /*blockNum*/)
	if len(pendingDeposits) != 2 {
		t.Errorf("Unexpected number of deposits. Wanted 2 deposit, got %+v", pendingDeposits)
	}
	hook.Reset()
}

func TestUnpackDepositLogData_OK(t *testing.T) {
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
		t.Fatalf("Could not init from contract: %v", err)
	}

	var pubkey [48]byte
	var withdrawalCreds [32]byte
	var sig [96]byte
	copy(pubkey[:], []byte("pubkey"))
	copy(sig[:], []byte("sig"))
	copy(withdrawalCreds[:], []byte("withdrawCreds"))

	data := &ethpb.Deposit_Data{
		PublicKey:             pubkey[:],
		Signature:             sig[:],
		WithdrawalCredentials: withdrawalCreds[:],
	}

	testAcc.TxOpts.Value = contracts.Amount32Eth()
	testAcc.TxOpts.GasLimit = 1000000
	if _, err := testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature); err != nil {
		t.Fatalf("Could not deposit to deposit contract %v", err)
	}
	testAcc.Backend.Commit()

	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			web3Service.depositContractAddress,
		},
	}

	logz, err := testAcc.Backend.FilterLogs(web3Service.ctx, query)
	if err != nil {
		t.Fatalf("Unable to retrieve logs %v", err)
	}

	loggedPubkey, withCreds, _, loggedSig, index, err := contracts.UnpackDepositLogData(logz[0].Data)
	if err != nil {
		t.Fatalf("Unable to unpack logs %v", err)
	}

	if binary.LittleEndian.Uint64(index) != 0 {
		t.Errorf("Retrieved merkle tree index is incorrect %d", index)
	}

	if !bytes.Equal(loggedPubkey, data.PublicKey) {
		t.Errorf("Pubkey is not the same as the data that was put in %v", loggedPubkey)
	}

	if !bytes.Equal(loggedSig, data.Signature) {
		t.Errorf("Proof of Possession is not the same as the data that was put in %v", loggedSig)
	}

	if !bytes.Equal(withCreds, data.WithdrawalCredentials) {
		t.Errorf("Withdrawal Credentials is not the same as the data that was put in %v", withCreds)
	}

}

func TestProcessETH2GenesisLog_8DuplicatePubkeys(t *testing.T) {
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
		HTTPLogger:      &goodLogger{},
		ContractBackend: testAcc.Backend,
		BeaconDB:        &kv.Store{},
		DepositCache:    depositcache.NewDepositCache(),
		BlockFetcher:    &goodFetcher{},
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	bConfig := params.MinimalSpecConfig()
	bConfig.MinGenesisTime = 0
	params.OverrideBeaconConfig(bConfig)

	testAcc.Backend.Commit()
	testAcc.Backend.AdjustTime(time.Duration(int64(time.Now().Nanosecond())))

	deposits, _ := testutil.SetupInitialDeposits(t, 1)
	data := deposits[0].Data

	testAcc.TxOpts.Value = contracts.Amount32Eth()
	testAcc.TxOpts.GasLimit = 1000000

	// 64 Validators are used as size required for beacon-chain to start. This number
	// is defined in the deposit contract as the number required for the testnet. The actual number
	// is 2**14
	for i := 0; i < depositsReqForChainStart; i++ {
		testAcc.TxOpts.Value = contracts.Amount32Eth()
		if _, err := testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature); err != nil {
			t.Fatalf("Could not deposit to deposit contract %v", err)
		}

		testAcc.Backend.Commit()
	}

	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			web3Service.depositContractAddress,
		},
	}

	logs, err := testAcc.Backend.FilterLogs(web3Service.ctx, query)
	if err != nil {
		t.Fatalf("Unable to retrieve logs %v", err)
	}

	for _, log := range logs {
		web3Service.ProcessLog(context.Background(), log)
	}

	if web3Service.chainStarted {
		t.Error("Genesis has been triggered despite being 8 duplicate keys")
	}

	testutil.AssertLogsDoNotContain(t, hook, "Minimum number of validators reached for beacon-chain to start")
	hook.Reset()
}

func TestProcessETH2GenesisLog(t *testing.T) {
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
		HTTPLogger:      &goodLogger{},
		ContractBackend: testAcc.Backend,
		BeaconDB:        &kv.Store{},
		DepositCache:    depositcache.NewDepositCache(),
		BlockFetcher:    &goodFetcher{},
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}
	bConfig := params.MinimalSpecConfig()
	bConfig.MinGenesisTime = 0
	params.OverrideBeaconConfig(bConfig)

	testAcc.Backend.Commit()
	testAcc.Backend.AdjustTime(time.Duration(int64(time.Now().Nanosecond())))

	deposits, _ := testutil.SetupInitialDeposits(t, uint64(depositsReqForChainStart))

	// 64 Validators are used as size required for beacon-chain to start. This number
	// is defined in the deposit contract as the number required for the testnet. The actual number
	// is 2**14
	for i := 0; i < depositsReqForChainStart; i++ {
		data := deposits[i].Data
		testAcc.TxOpts.Value = contracts.Amount32Eth()
		testAcc.TxOpts.GasLimit = 1000000
		if _, err := testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature); err != nil {
			t.Fatalf("Could not deposit to deposit contract %v", err)
		}

		testAcc.Backend.Commit()
	}

	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			web3Service.depositContractAddress,
		},
	}

	logs, err := testAcc.Backend.FilterLogs(web3Service.ctx, query)
	if err != nil {
		t.Fatalf("Unable to retrieve logs %v", err)
	}

	genesisTimeChan := make(chan time.Time, 1)
	sub := web3Service.chainStartFeed.Subscribe(genesisTimeChan)
	defer sub.Unsubscribe()

	for _, log := range logs {
		web3Service.ProcessLog(context.Background(), log)
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

func TestWeb3ServiceProcessDepositLog_RequestMissedDeposits(t *testing.T) {
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
		HTTPLogger:      testAcc.Backend,
		ContractBackend: testAcc.Backend,
		BeaconDB:        &kv.Store{},
		DepositCache:    depositcache.NewDepositCache(),
		BlockFetcher:    &goodFetcher{},
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}
	bConfig := params.MinimalSpecConfig()
	bConfig.MinGenesisTime = 0
	params.OverrideBeaconConfig(bConfig)

	testAcc.Backend.Commit()
	testAcc.Backend.AdjustTime(time.Duration(int64(time.Now().Nanosecond())))
	depositsWanted := 10
	deposits, _ := testutil.SetupInitialDeposits(t, uint64(depositsWanted))

	for i := 0; i < depositsWanted; i++ {
		data := deposits[i].Data
		testAcc.TxOpts.Value = contracts.Amount32Eth()
		testAcc.TxOpts.GasLimit = 1000000
		if _, err := testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature); err != nil {
			t.Fatalf("Could not deposit to deposit contract %v", err)
		}

		testAcc.Backend.Commit()
	}

	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			web3Service.depositContractAddress,
		},
	}

	logs, err := testAcc.Backend.FilterLogs(web3Service.ctx, query)
	if err != nil {
		t.Fatalf("Unable to retrieve logs %v", err)
	}

	logsToBeProcessed := append(logs[:depositsWanted-3], logs[depositsWanted-2:]...)
	// we purposely miss processing the middle two logs so that the service, re-requests them
	for _, log := range logsToBeProcessed {
		if err := web3Service.ProcessLog(context.Background(), log); err != nil {
			t.Fatal(err)
		}
		web3Service.lastRequestedBlock.Set(big.NewInt(int64(log.BlockNumber)))
	}

	if web3Service.lastReceivedMerkleIndex != int64(depositsWanted-1) {
		t.Errorf("missing logs were not re-requested. Wanted Index %d but got %d", depositsWanted-1, web3Service.lastReceivedMerkleIndex)
	}

	hook.Reset()
}
