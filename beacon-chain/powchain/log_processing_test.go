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
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"gopkg.in/d4l3k/messagediff.v1"
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
	beaconDB := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, beaconDB)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		ETH1Endpoint:    endpoint,
		DepositContract: testAcc.ContractAddr,
		BeaconDB:        beaconDB,
		DepositCache:    depositcache.NewDepositCache(),
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
	deposits, _, _ := testutil.DeterministicDepositsAndKeys(1)
	_, depositRoots, err := testutil.DeterministicDepositTrie(len(deposits))
	if err != nil {
		t.Fatal(err)
	}
	data := deposits[0].Data

	testAcc.TxOpts.Value = contracts.Amount32Eth()
	testAcc.TxOpts.GasLimit = 1000000
	if _, err := testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature, depositRoots[0]); err != nil {
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
	beaconDB := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, beaconDB)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		ETH1Endpoint:    endpoint,
		DepositContract: testAcc.ContractAddr,
		BeaconDB:        beaconDB,
		DepositCache:    depositcache.NewDepositCache(),
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

	deposits, _, _ := testutil.DeterministicDepositsAndKeys(1)
	_, depositRoots, err := testutil.DeterministicDepositTrie(len(deposits))
	if err != nil {
		t.Fatal(err)
	}
	data := deposits[0].Data

	testAcc.TxOpts.Value = contracts.Amount32Eth()
	testAcc.TxOpts.GasLimit = 1000000

	if _, err := testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature, depositRoots[0]); err != nil {
		t.Fatalf("Could not deposit to deposit contract %v", err)
	}

	if _, err := testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature, depositRoots[0]); err != nil {
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

	web3Service.chainStartData.Chainstarted = true

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
	beaconDB := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, beaconDB)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		ETH1Endpoint:    endpoint,
		BeaconDB:        beaconDB,
		DepositContract: testAcc.ContractAddr,
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
		t.Fatalf("Could not init from contract: %v", err)
	}

	deposits, _, _ := testutil.DeterministicDepositsAndKeys(1)
	_, depositRoots, err := testutil.DeterministicDepositTrie(len(deposits))
	if err != nil {
		t.Fatal(err)
	}
	data := deposits[0].Data

	testAcc.TxOpts.Value = contracts.Amount32Eth()
	testAcc.TxOpts.GasLimit = 1000000
	if _, err := testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature, depositRoots[0]); err != nil {
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
	beaconDB := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, beaconDB)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		ETH1Endpoint:    endpoint,
		DepositContract: testAcc.ContractAddr,
		BeaconDB:        beaconDB,
		DepositCache:    depositcache.NewDepositCache(),
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}
	web3Service = setDefaultMocks(web3Service)
	web3Service.depositContractCaller, err = contracts.NewDepositContractCaller(testAcc.ContractAddr, testAcc.Backend)
	if err != nil {
		t.Fatal(err)
	}

	bConfig := params.MinimalSpecConfig()
	bConfig.MinGenesisTime = 0
	params.OverrideBeaconConfig(bConfig)

	testAcc.Backend.Commit()
	testAcc.Backend.AdjustTime(time.Duration(int64(time.Now().Nanosecond())))

	deposits, _, _ := testutil.DeterministicDepositsAndKeys(1)
	_, depositRoots, err := testutil.DeterministicDepositTrie(len(deposits))
	if err != nil {
		t.Fatal(err)
	}
	data := deposits[0].Data

	testAcc.TxOpts.Value = contracts.Amount32Eth()
	testAcc.TxOpts.GasLimit = 1000000

	// 64 Validators are used as size required for beacon-chain to start. This number
	// is defined in the deposit contract as the number required for the testnet. The actual number
	// is 2**14
	for i := 0; i < depositsReqForChainStart; i++ {
		testAcc.TxOpts.Value = contracts.Amount32Eth()
		if _, err := testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature, depositRoots[0]); err != nil {
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

	if web3Service.chainStartData.Chainstarted {
		t.Error("Genesis has been triggered despite being 8 duplicate keys")
	}

	testutil.AssertLogsDoNotContain(t, hook, "Minimum number of validators reached for beacon-chain to start")
	hook.Reset()
}

func TestProcessETH2GenesisLog(t *testing.T) {
	config := &featureconfig.Flags{
		CustomGenesisDelay: 0,
	}
	featureconfig.Init(config)
	hook := logTest.NewGlobal()
	testAcc, err := contracts.Setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	beaconDB := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, beaconDB)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		ETH1Endpoint:    endpoint,
		DepositContract: testAcc.ContractAddr,
		BeaconDB:        beaconDB,
		DepositCache:    depositcache.NewDepositCache(),
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}
	web3Service = setDefaultMocks(web3Service)
	web3Service.depositContractCaller, err = contracts.NewDepositContractCaller(testAcc.ContractAddr, testAcc.Backend)
	if err != nil {
		t.Fatal(err)
	}
	bConfig := params.MinimalSpecConfig()
	bConfig.MinGenesisTime = 0
	params.OverrideBeaconConfig(bConfig)

	testAcc.Backend.Commit()
	testAcc.Backend.AdjustTime(time.Duration(int64(time.Now().Nanosecond())))

	deposits, _, _ := testutil.DeterministicDepositsAndKeys(uint64(depositsReqForChainStart))

	_, roots, err := testutil.DeterministicDepositTrie(len(deposits))
	if err != nil {
		t.Fatal(err)
	}

	// 64 Validators are used as size required for beacon-chain to start. This number
	// is defined in the deposit contract as the number required for the testnet. The actual number
	// is 2**14
	for i := 0; i < depositsReqForChainStart; i++ {
		data := deposits[i].Data
		testAcc.TxOpts.Value = contracts.Amount32Eth()
		testAcc.TxOpts.GasLimit = 1000000
		if _, err := testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature, roots[i]); err != nil {
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

	if len(logs) != depositsReqForChainStart {
		t.Fatalf(
			"Did not receive enough logs, received %d, wanted %d",
			len(logs),
			depositsReqForChainStart,
		)
	}

	// Set up our subscriber now to listen for the chain started event.
	stateChannel := make(chan *feed.Event, 1)
	stateSub := web3Service.stateNotifier.StateFeed().Subscribe(stateChannel)
	defer stateSub.Unsubscribe()

	for _, log := range logs {
		web3Service.ProcessLog(context.Background(), log)
	}

	err = web3Service.ProcessETH1Block(context.Background(), big.NewInt(int64(logs[len(logs)-1].BlockNumber)))
	if err != nil {
		t.Fatal(err)
	}

	cachedDeposits := web3Service.ChainStartDeposits()
	if len(cachedDeposits) != depositsReqForChainStart {
		t.Fatalf(
			"Did not cache the chain start deposits correctly, received %d, wanted %d",
			len(cachedDeposits),
			depositsReqForChainStart,
		)
	}

	// Receive the chain started event.
	for started := false; !started; {
		select {
		case event := <-stateChannel:
			if event.Type == statefeed.ChainStarted {
				started = true
			}
		}
	}

	testutil.AssertLogsDoNotContain(t, hook, "Unable to unpack ChainStart log data")
	testutil.AssertLogsDoNotContain(t, hook, "Receipt root from log doesn't match the root saved in memory")
	testutil.AssertLogsDoNotContain(t, hook, "Invalid timestamp from log")
	testutil.AssertLogsContain(t, hook, "Minimum number of validators reached for beacon-chain to start")

	hook.Reset()
}

func TestProcessETH2GenesisLog_CorrectNumOfDeposits(t *testing.T) {
	hook := logTest.NewGlobal()
	testAcc, err := contracts.Setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	kvStore := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, kvStore)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		ETH1Endpoint:    endpoint,
		DepositContract: testAcc.ContractAddr,
		BeaconDB:        kvStore,
		DepositCache:    depositcache.NewDepositCache(),
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}
	web3Service = setDefaultMocks(web3Service)
	web3Service.depositContractCaller, err = contracts.NewDepositContractCaller(testAcc.ContractAddr, testAcc.Backend)
	if err != nil {
		t.Fatal(err)
	}
	web3Service.rpcClient = &mockPOW.RPCClient{Backend: testAcc.Backend}
	web3Service.httpLogger = testAcc.Backend
	web3Service.latestEth1Data.LastRequestedBlock = 0
	web3Service.latestEth1Data.BlockHeight = 0
	bConfig := params.MinimalSpecConfig()
	bConfig.MinGenesisTime = 0
	params.OverrideBeaconConfig(bConfig)
	flags.Get().DeploymentBlock = 0

	testAcc.Backend.Commit()
	testAcc.Backend.AdjustTime(time.Duration(int64(time.Now().Nanosecond())))

	totalNumOfDeposits := depositsReqForChainStart + 30

	deposits, _, _ := testutil.DeterministicDepositsAndKeys(uint64(totalNumOfDeposits))
	_, depositRoots, err := testutil.DeterministicDepositTrie(len(deposits))
	if err != nil {
		t.Fatal(err)
	}
	depositOffset := 5

	// 64 Validators are used as size required for beacon-chain to start. This number
	// is defined in the deposit contract as the number required for the testnet. The actual number
	// is 2**14
	for i := 0; i < totalNumOfDeposits; i++ {
		data := deposits[i].Data
		testAcc.TxOpts.Value = contracts.Amount32Eth()
		testAcc.TxOpts.GasLimit = 1000000
		if _, err := testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature, depositRoots[i]); err != nil {
			t.Fatalf("Could not deposit to deposit contract %v", err)
		}
		// pack 8 deposits into a block with an offset of
		// 5
		if (i+1)%8 == depositOffset {
			testAcc.Backend.Commit()
		}
	}
	web3Service.latestEth1Data.BlockHeight = testAcc.Backend.Blockchain().CurrentBlock().NumberU64()

	// Set up our subscriber now to listen for the chain started event.
	stateChannel := make(chan *feed.Event, 1)
	stateSub := web3Service.stateNotifier.StateFeed().Subscribe(stateChannel)
	defer stateSub.Unsubscribe()

	err = web3Service.processPastLogs(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	cachedDeposits := web3Service.ChainStartDeposits()
	requiredDepsForChainstart := depositsReqForChainStart + depositOffset
	if len(cachedDeposits) != requiredDepsForChainstart {
		t.Fatalf(
			"Did not cache the chain start deposits correctly, received %d, wanted %d",
			len(cachedDeposits),
			requiredDepsForChainstart,
		)
	}

	// Receive the chain started event.
	for started := false; !started; {
		select {
		case event := <-stateChannel:
			if event.Type == statefeed.ChainStarted {
				started = true
			}
		}
	}

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
	beaconDB := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, beaconDB)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		ETH1Endpoint:    endpoint,
		DepositContract: testAcc.ContractAddr,
		BeaconDB:        beaconDB,
		DepositCache:    depositcache.NewDepositCache(),
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}
	web3Service = setDefaultMocks(web3Service)
	web3Service.depositContractCaller, err = contracts.NewDepositContractCaller(testAcc.ContractAddr, testAcc.Backend)
	if err != nil {
		t.Fatal(err)
	}
	web3Service.httpLogger = testAcc.Backend
	bConfig := params.MinimalSpecConfig()
	bConfig.MinGenesisTime = 0
	params.OverrideBeaconConfig(bConfig)

	testAcc.Backend.Commit()
	testAcc.Backend.AdjustTime(time.Duration(int64(time.Now().Nanosecond())))
	depositsWanted := 10
	deposits, _, _ := testutil.DeterministicDepositsAndKeys(uint64(depositsWanted))
	_, depositRoots, err := testutil.DeterministicDepositTrie(len(deposits))
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < depositsWanted; i++ {
		data := deposits[i].Data
		testAcc.TxOpts.Value = contracts.Amount32Eth()
		testAcc.TxOpts.GasLimit = 1000000
		if _, err := testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature, depositRoots[i]); err != nil {
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

	if len(logs) != depositsWanted {
		t.Fatalf(
			"Did not receive enough logs, received %d, wanted %d",
			len(logs),
			depositsReqForChainStart,
		)
	}

	logsToBeProcessed := append(logs[:depositsWanted-3], logs[depositsWanted-2:]...)
	// we purposely miss processing the middle two logs so that the service, re-requests them
	for _, log := range logsToBeProcessed {
		if err := web3Service.ProcessLog(context.Background(), log); err != nil {
			t.Fatal(err)
		}
		web3Service.latestEth1Data.LastRequestedBlock = log.BlockNumber
	}

	if web3Service.lastReceivedMerkleIndex != int64(depositsWanted-1) {
		t.Errorf("missing logs were not re-requested. Wanted Index %d but got %d", depositsWanted-1, web3Service.lastReceivedMerkleIndex)
	}

	web3Service.lastReceivedMerkleIndex = -1
	web3Service.latestEth1Data.LastRequestedBlock = 0
	genSt, err := state.EmptyGenesisState()
	if err != nil {
		t.Fatal(err)
	}
	web3Service.preGenesisState = genSt
	if err := web3Service.preGenesisState.SetEth1Data(&ethpb.Eth1Data{}); err != nil {
		t.Fatal(err)
	}
	web3Service.chainStartData.ChainstartDeposits = []*ethpb.Deposit{}
	web3Service.depositTrie, err = trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatal(err)
	}

	logsToBeProcessed = append(logs[:depositsWanted-8], logs[depositsWanted-2:]...)
	// We purposely miss processing the middle 7 logs so that the service, re-requests them.
	for _, log := range logsToBeProcessed {
		if err := web3Service.ProcessLog(context.Background(), log); err != nil {
			t.Fatal(err)
		}
		web3Service.latestEth1Data.LastRequestedBlock = log.BlockNumber
	}

	if web3Service.lastReceivedMerkleIndex != int64(depositsWanted-1) {
		t.Errorf("Missing logs were not re-requested want = %d but got = %d", depositsWanted-1, web3Service.lastReceivedMerkleIndex)
	}

	hook.Reset()
}

func TestConsistentGenesisState(t *testing.T) {
	t.Skip("Incorrect test setup")
	testAcc, err := contracts.Setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	beaconDB := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, beaconDB)
	web3Service := newPowchainService(t, testAcc, beaconDB)

	testAcc.Backend.Commit()
	testAcc.Backend.AdjustTime(time.Duration(int64(time.Now().Nanosecond())))

	deposits, _, _ := testutil.DeterministicDepositsAndKeys(uint64(depositsReqForChainStart))

	_, roots, err := testutil.DeterministicDepositTrie(len(deposits))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go web3Service.run(ctx.Done())

	// 64 Validators are used as size required for beacon-chain to start. This number
	// is defined in the deposit contract as the number required for the testnet. The actual number
	// is 2**14.
	for i := 0; i < depositsReqForChainStart; i++ {
		data := deposits[i].Data
		testAcc.TxOpts.Value = contracts.Amount32Eth()
		testAcc.TxOpts.GasLimit = 1000000
		if _, err := testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature, roots[i]); err != nil {
			t.Fatalf("Could not deposit to deposit contract %v", err)
		}

		testAcc.Backend.Commit()
	}

	for i := 0; i < int(params.BeaconConfig().LogBlockDelay); i++ {
		testAcc.Backend.Commit()
	}

	time.Sleep(2 * time.Second)
	if !web3Service.chainStartData.Chainstarted {
		t.Fatalf("Service hasn't chainstarted yet with a block height of %d", web3Service.latestEth1Data.BlockHeight)
	}

	// Advance 10 blocks.
	for i := 0; i < 10; i++ {
		testAcc.Backend.Commit()
	}

	// Tearing down to prevent registration error.
	testDB.TeardownDB(t, beaconDB)

	newBeaconDB := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, newBeaconDB)

	newWeb3Service := newPowchainService(t, testAcc, newBeaconDB)
	go newWeb3Service.run(ctx.Done())

	time.Sleep(2 * time.Second)
	if !newWeb3Service.chainStartData.Chainstarted {
		t.Fatal("Service hasn't chainstarted yet")
	}

	diff, _ := messagediff.PrettyDiff(web3Service.chainStartData.Eth1Data, newWeb3Service.chainStartData.Eth1Data)
	if diff != "" {
		t.Errorf("Two services have different eth1data: %s", diff)
	}
	cancel()
}

func newPowchainService(t *testing.T, eth1Backend *contracts.TestAccount, beaconDB db.Database) *Service {
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		ETH1Endpoint:    endpoint,
		DepositContract: eth1Backend.ContractAddr,
		BeaconDB:        beaconDB,
		DepositCache:    depositcache.NewDepositCache(),
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}
	web3Service = setDefaultMocks(web3Service)
	web3Service.depositContractCaller, err = contracts.NewDepositContractCaller(eth1Backend.ContractAddr, eth1Backend.Backend)
	if err != nil {
		t.Fatal(err)
	}

	web3Service.rpcClient = &mockPOW.RPCClient{Backend: eth1Backend.Backend}
	web3Service.reader = &goodReader{backend: eth1Backend.Backend}
	web3Service.blockFetcher = &goodFetcher{backend: eth1Backend.Backend}
	web3Service.httpLogger = &goodLogger{backend: eth1Backend.Backend}
	web3Service.logger = &goodLogger{backend: eth1Backend.Backend}
	bConfig := params.MinimalSpecConfig()
	bConfig.MinGenesisTime = 0
	params.OverrideBeaconConfig(bConfig)
	web3Service.headerChan = make(chan *gethTypes.Header)
	return web3Service
}
