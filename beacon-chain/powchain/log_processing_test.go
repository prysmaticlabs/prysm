package powchain

import (
	"context"
	"encoding/binary"
	"fmt"
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
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
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
	require.NoError(t, err, "Unable to set up simulated backend")

	beaconDB, _ := testDB.SetupDB(t)
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint:    endpoint,
		DepositContract: testAcc.ContractAddr,
		BeaconDB:        beaconDB,
		DepositCache:    depositCache,
	})
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")
	web3Service = setDefaultMocks(web3Service)
	web3Service.depositContractCaller, err = contracts.NewDepositContractCaller(testAcc.ContractAddr, testAcc.Backend)
	require.NoError(t, err)

	testAcc.Backend.Commit()
	testutil.ResetCache()
	deposits, _, err := testutil.DeterministicDepositsAndKeys(1)
	require.NoError(t, err)

	_, depositRoots, err := testutil.DeterministicDepositTrie(len(deposits))
	require.NoError(t, err)
	data := deposits[0].Data

	testAcc.TxOpts.Value = contracts.Amount32Eth()
	testAcc.TxOpts.GasLimit = 1000000
	_, err = testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature, depositRoots[0])
	require.NoError(t, err, "Could not deposit to deposit contract")

	testAcc.Backend.Commit()

	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			web3Service.depositContractAddress,
		},
	}

	logs, err := testAcc.Backend.FilterLogs(web3Service.ctx, query)
	require.NoError(t, err, "Unable to retrieve logs")

	if len(logs) == 0 {
		t.Fatal("no logs")
	}

	err = web3Service.ProcessLog(context.Background(), logs[0])
	require.NoError(t, err)

	require.LogsDoNotContain(t, hook, "Could not unpack log")
	require.LogsDoNotContain(t, hook, "Could not save in trie")
	require.LogsDoNotContain(t, hook, "could not deserialize validator public key")
	require.LogsDoNotContain(t, hook, "could not convert bytes to signature")
	require.LogsDoNotContain(t, hook, "could not sign root for deposit data")
	require.LogsDoNotContain(t, hook, "deposit signature did not verify")
	require.LogsDoNotContain(t, hook, "could not tree hash deposit data")
	require.LogsDoNotContain(t, hook, "deposit merkle branch of deposit root did not verify for root")
	require.LogsContain(t, hook, "Deposit registered from deposit contract")

	hook.Reset()
}

func TestProcessDepositLog_InsertsPendingDeposit(t *testing.T) {
	hook := logTest.NewGlobal()
	testAcc, err := contracts.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	beaconDB, _ := testDB.SetupDB(t)
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint:    endpoint,
		DepositContract: testAcc.ContractAddr,
		BeaconDB:        beaconDB,
		DepositCache:    depositCache,
	})
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")
	web3Service = setDefaultMocks(web3Service)
	web3Service.depositContractCaller, err = contracts.NewDepositContractCaller(testAcc.ContractAddr, testAcc.Backend)
	require.NoError(t, err)

	testAcc.Backend.Commit()

	testutil.ResetCache()
	deposits, _, err := testutil.DeterministicDepositsAndKeys(1)
	require.NoError(t, err)
	_, depositRoots, err := testutil.DeterministicDepositTrie(len(deposits))
	require.NoError(t, err)
	data := deposits[0].Data

	testAcc.TxOpts.Value = contracts.Amount32Eth()
	testAcc.TxOpts.GasLimit = 1000000

	_, err = testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature, depositRoots[0])
	require.NoError(t, err, "Could not deposit to deposit contract")

	_, err = testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature, depositRoots[0])
	require.NoError(t, err, "Could not deposit to deposit contract")

	testAcc.Backend.Commit()

	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			web3Service.depositContractAddress,
		},
	}

	logs, err := testAcc.Backend.FilterLogs(web3Service.ctx, query)
	require.NoError(t, err, "Unable to retrieve logs")

	web3Service.chainStartData.Chainstarted = true

	err = web3Service.ProcessDepositLog(context.Background(), logs[0])
	require.NoError(t, err)
	err = web3Service.ProcessDepositLog(context.Background(), logs[1])
	require.NoError(t, err)

	pendingDeposits := web3Service.depositCache.PendingDeposits(context.Background(), nil /*blockNum*/)
	require.Equal(t, int(2), len(pendingDeposits), "Unexpected number of deposits")

	hook.Reset()
}

func TestUnpackDepositLogData_OK(t *testing.T) {
	testAcc, err := contracts.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	beaconDB, _ := testDB.SetupDB(t)
	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint:    endpoint,
		BeaconDB:        beaconDB,
		DepositContract: testAcc.ContractAddr,
	})
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")
	web3Service = setDefaultMocks(web3Service)
	web3Service.depositContractCaller, err = contracts.NewDepositContractCaller(testAcc.ContractAddr, testAcc.Backend)
	require.NoError(t, err)

	testAcc.Backend.Commit()

	if err := web3Service.initDataFromContract(); err != nil {
		t.Fatalf("Could not init from contract: %v", err)
	}

	testutil.ResetCache()
	deposits, _, err := testutil.DeterministicDepositsAndKeys(1)
	require.NoError(t, err)
	_, depositRoots, err := testutil.DeterministicDepositTrie(len(deposits))
	require.NoError(t, err)
	data := deposits[0].Data

	testAcc.TxOpts.Value = contracts.Amount32Eth()
	testAcc.TxOpts.GasLimit = 1000000
	_, err = testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature, depositRoots[0])
	require.NoError(t, err, "Could not deposit to deposit contract")
	testAcc.Backend.Commit()

	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			web3Service.depositContractAddress,
		},
	}

	logz, err := testAcc.Backend.FilterLogs(web3Service.ctx, query)
	require.NoError(t, err, "Unable to retrieve logs")

	loggedPubkey, withCreds, _, loggedSig, index, err := contracts.UnpackDepositLogData(logz[0].Data)
	require.NoError(t, err, "Unable to unpack logs")

	require.Equal(t, uint64(0), binary.LittleEndian.Uint64(index), "Retrieved merkle tree index is incorrect")
	require.DeepEqual(t, data.PublicKey, loggedPubkey, "Pubkey is not the same as the data that was put in")
	require.DeepEqual(t, data.Signature, loggedSig, "Proof of Possession is not the same as the data that was put in")
	require.DeepEqual(t, data.WithdrawalCredentials, withCreds, "Withdrawal Credentials is not the same as the data that was put in")
}

func TestProcessETH2GenesisLog_8DuplicatePubkeys(t *testing.T) {
	hook := logTest.NewGlobal()
	testAcc, err := contracts.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	beaconDB, _ := testDB.SetupDB(t)
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint:    endpoint,
		DepositContract: testAcc.ContractAddr,
		BeaconDB:        beaconDB,
		DepositCache:    depositCache,
	})
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")
	web3Service = setDefaultMocks(web3Service)
	web3Service.depositContractCaller, err = contracts.NewDepositContractCaller(testAcc.ContractAddr, testAcc.Backend)
	require.NoError(t, err)

	params.SetupTestConfigCleanup(t)
	bConfig := params.MinimalSpecConfig()
	bConfig.MinGenesisTime = 0
	params.OverrideBeaconConfig(bConfig)

	testAcc.Backend.Commit()
	if err := testAcc.Backend.AdjustTime(time.Duration(int64(time.Now().Nanosecond()))); err != nil {
		t.Fatal(err)
	}

	testutil.ResetCache()
	deposits, _, err := testutil.DeterministicDepositsAndKeys(1)
	require.NoError(t, err)
	_, depositRoots, err := testutil.DeterministicDepositTrie(len(deposits))
	require.NoError(t, err)
	data := deposits[0].Data

	testAcc.TxOpts.Value = contracts.Amount32Eth()
	testAcc.TxOpts.GasLimit = 1000000

	// 64 Validators are used as size required for beacon-chain to start. This number
	// is defined in the deposit contract as the number required for the testnet. The actual number
	// is 2**14
	for i := 0; i < depositsReqForChainStart; i++ {
		testAcc.TxOpts.Value = contracts.Amount32Eth()
		_, err = testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature, depositRoots[0])
		require.NoError(t, err, "Could not deposit to deposit contract")

		testAcc.Backend.Commit()
	}

	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			web3Service.depositContractAddress,
		},
	}

	logs, err := testAcc.Backend.FilterLogs(web3Service.ctx, query)
	require.NoError(t, err, "Unable to retrieve logs")

	for _, log := range logs {
		err = web3Service.ProcessLog(context.Background(), log)
		require.NoError(t, err)
	}
	assert.Equal(t, false, web3Service.chainStartData.Chainstarted, "Genesis has been triggered despite being 8 duplicate keys")

	require.LogsDoNotContain(t, hook, "Minimum number of validators reached for beacon-chain to start")
	hook.Reset()
}

func TestProcessETH2GenesisLog(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig()
	cfg.GenesisDelay = 0
	params.OverrideBeaconConfig(cfg)
	hook := logTest.NewGlobal()
	testAcc, err := contracts.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	beaconDB, _ := testDB.SetupDB(t)
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint:    endpoint,
		DepositContract: testAcc.ContractAddr,
		BeaconDB:        beaconDB,
		DepositCache:    depositCache,
	})
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")
	web3Service = setDefaultMocks(web3Service)
	web3Service.depositContractCaller, err = contracts.NewDepositContractCaller(testAcc.ContractAddr, testAcc.Backend)
	require.NoError(t, err)
	params.SetupTestConfigCleanup(t)
	bConfig := params.MinimalSpecConfig()
	bConfig.MinGenesisTime = 0
	params.OverrideBeaconConfig(bConfig)

	testAcc.Backend.Commit()
	if err := testAcc.Backend.AdjustTime(time.Duration(int64(time.Now().Nanosecond()))); err != nil {
		t.Fatal(err)
	}

	testutil.ResetCache()
	deposits, _, err := testutil.DeterministicDepositsAndKeys(uint64(depositsReqForChainStart))
	require.NoError(t, err)
	_, roots, err := testutil.DeterministicDepositTrie(len(deposits))
	require.NoError(t, err)

	// 64 Validators are used as size required for beacon-chain to start. This number
	// is defined in the deposit contract as the number required for the testnet. The actual number
	// is 2**14
	for i := 0; i < depositsReqForChainStart; i++ {
		data := deposits[i].Data
		testAcc.TxOpts.Value = contracts.Amount32Eth()
		testAcc.TxOpts.GasLimit = 1000000
		_, err = testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature, roots[i])
		require.NoError(t, err, "Could not deposit to deposit contract")

		testAcc.Backend.Commit()
	}

	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			web3Service.depositContractAddress,
		},
	}

	logs, err := testAcc.Backend.FilterLogs(web3Service.ctx, query)
	require.NoError(t, err, "Unable to retrieve logs")
	require.Equal(t, depositsReqForChainStart, len(logs))

	// Set up our subscriber now to listen for the chain started event.
	stateChannel := make(chan *feed.Event, 1)
	stateSub := web3Service.stateNotifier.StateFeed().Subscribe(stateChannel)
	defer stateSub.Unsubscribe()

	for _, log := range logs {
		err = web3Service.ProcessLog(context.Background(), log)
		require.NoError(t, err)
	}

	err = web3Service.ProcessETH1Block(context.Background(), big.NewInt(int64(logs[len(logs)-1].BlockNumber)))
	require.NoError(t, err)

	cachedDeposits := web3Service.ChainStartDeposits()
	require.Equal(t, depositsReqForChainStart, len(cachedDeposits))

	// Receive the chain started event.
	for started := false; !started; {
		select {
		case event := <-stateChannel:
			if event.Type == statefeed.ChainStarted {
				started = true
			}
		}
	}

	require.LogsDoNotContain(t, hook, "Unable to unpack ChainStart log data")
	require.LogsDoNotContain(t, hook, "Receipt root from log doesn't match the root saved in memory")
	require.LogsDoNotContain(t, hook, "Invalid timestamp from log")
	require.LogsContain(t, hook, "Minimum number of validators reached for beacon-chain to start")

	hook.Reset()
}

func TestProcessETH2GenesisLog_CorrectNumOfDeposits(t *testing.T) {
	hook := logTest.NewGlobal()
	testAcc, err := contracts.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	kvStore, _ := testDB.SetupDB(t)
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint:    endpoint,
		DepositContract: testAcc.ContractAddr,
		BeaconDB:        kvStore,
		DepositCache:    depositCache,
	})
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")
	web3Service = setDefaultMocks(web3Service)
	web3Service.depositContractCaller, err = contracts.NewDepositContractCaller(testAcc.ContractAddr, testAcc.Backend)
	require.NoError(t, err)
	web3Service.rpcClient = &mockPOW.RPCClient{Backend: testAcc.Backend}
	web3Service.httpLogger = testAcc.Backend
	web3Service.eth1DataFetcher = &goodFetcher{backend: testAcc.Backend}
	web3Service.latestEth1Data.LastRequestedBlock = 0
	web3Service.latestEth1Data.BlockHeight = testAcc.Backend.Blockchain().CurrentBlock().NumberU64()
	web3Service.latestEth1Data.BlockTime = testAcc.Backend.Blockchain().CurrentBlock().Time()
	params.SetupTestConfigCleanup(t)
	bConfig := params.MinimalSpecConfig()
	bConfig.MinGenesisTime = 0
	bConfig.SecondsPerETH1Block = 10
	params.OverrideBeaconConfig(bConfig)
	nConfig := params.BeaconNetworkConfig()
	nConfig.ContractDeploymentBlock = 0
	params.OverrideBeaconNetworkConfig(nConfig)

	testAcc.Backend.Commit()

	totalNumOfDeposits := depositsReqForChainStart + 30

	deposits, _, err := testutil.DeterministicDepositsAndKeys(uint64(totalNumOfDeposits))
	require.NoError(t, err)
	_, depositRoots, err := testutil.DeterministicDepositTrie(len(deposits))
	require.NoError(t, err)
	depositOffset := 5

	// 64 Validators are used as size required for beacon-chain to start. This number
	// is defined in the deposit contract as the number required for the testnet. The actual number
	// is 2**14
	for i := 0; i < totalNumOfDeposits; i++ {
		data := deposits[i].Data
		testAcc.TxOpts.Value = contracts.Amount32Eth()
		testAcc.TxOpts.GasLimit = 1000000
		_, err = testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature, depositRoots[i])
		require.NoError(t, err, "Could not deposit to deposit contract")
		// pack 8 deposits into a block with an offset of
		// 5
		if (i+1)%8 == depositOffset {
			testAcc.Backend.Commit()
		}
	}
	// Forward the chain to account for the follow distance
	for i := uint64(0); i < params.BeaconConfig().Eth1FollowDistance; i++ {
		testAcc.Backend.Commit()
	}
	web3Service.latestEth1Data.BlockHeight = testAcc.Backend.Blockchain().CurrentBlock().NumberU64()
	web3Service.latestEth1Data.BlockTime = testAcc.Backend.Blockchain().CurrentBlock().Time()

	// Set up our subscriber now to listen for the chain started event.
	stateChannel := make(chan *feed.Event, 1)
	stateSub := web3Service.stateNotifier.StateFeed().Subscribe(stateChannel)
	defer stateSub.Unsubscribe()

	err = web3Service.processPastLogs(context.Background())
	require.NoError(t, err)

	cachedDeposits := web3Service.ChainStartDeposits()
	requiredDepsForChainstart := depositsReqForChainStart + depositOffset
	require.Equal(t, requiredDepsForChainstart, len(cachedDeposits), "Did not cache the chain start deposits correctly")

	// Receive the chain started event.
	for started := false; !started; {
		select {
		case event := <-stateChannel:
			if event.Type == statefeed.ChainStarted {
				started = true
			}
		}
	}

	require.LogsDoNotContain(t, hook, "Unable to unpack ChainStart log data")
	require.LogsDoNotContain(t, hook, "Receipt root from log doesn't match the root saved in memory")
	require.LogsDoNotContain(t, hook, "Invalid timestamp from log")
	require.LogsContain(t, hook, "Minimum number of validators reached for beacon-chain to start")

	hook.Reset()
}

func TestWeb3ServiceProcessDepositLog_RequestMissedDeposits(t *testing.T) {
	hook := logTest.NewGlobal()
	testAcc, err := contracts.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	beaconDB, _ := testDB.SetupDB(t)
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint:    endpoint,
		DepositContract: testAcc.ContractAddr,
		BeaconDB:        beaconDB,
		DepositCache:    depositCache,
	})
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")
	web3Service = setDefaultMocks(web3Service)
	web3Service.depositContractCaller, err = contracts.NewDepositContractCaller(testAcc.ContractAddr, testAcc.Backend)
	require.NoError(t, err)
	web3Service.httpLogger = testAcc.Backend
	params.SetupTestConfigCleanup(t)
	bConfig := params.MinimalSpecConfig()
	bConfig.MinGenesisTime = 0
	params.OverrideBeaconConfig(bConfig)

	testAcc.Backend.Commit()
	if err := testAcc.Backend.AdjustTime(time.Duration(int64(time.Now().Nanosecond()))); err != nil {
		t.Fatal(err)
	}
	depositsWanted := 10
	testutil.ResetCache()
	deposits, _, err := testutil.DeterministicDepositsAndKeys(uint64(depositsWanted))
	require.NoError(t, err)
	_, depositRoots, err := testutil.DeterministicDepositTrie(len(deposits))
	require.NoError(t, err)

	for i := 0; i < depositsWanted; i++ {
		data := deposits[i].Data
		testAcc.TxOpts.Value = contracts.Amount32Eth()
		testAcc.TxOpts.GasLimit = 1000000
		_, err = testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature, depositRoots[i])
		require.NoError(t, err, "Could not deposit to deposit contract")

		testAcc.Backend.Commit()
	}

	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			web3Service.depositContractAddress,
		},
	}

	logs, err := testAcc.Backend.FilterLogs(web3Service.ctx, query)
	require.NoError(t, err, "Unable to retrieve logs")
	require.Equal(t, depositsWanted, len(logs), "Did not receive enough logs")

	logsToBeProcessed := append(logs[:depositsWanted-3], logs[depositsWanted-2:]...)
	// we purposely miss processing the middle two logs so that the service, re-requests them
	for _, log := range logsToBeProcessed {
		err = web3Service.ProcessLog(context.Background(), log)
		require.NoError(t, err)
		web3Service.latestEth1Data.LastRequestedBlock = log.BlockNumber
	}

	assert.Equal(t, int64(depositsWanted-1), web3Service.lastReceivedMerkleIndex, "missing logs were not re-requested")

	web3Service.lastReceivedMerkleIndex = -1
	web3Service.latestEth1Data.LastRequestedBlock = 0
	genSt, err := state.EmptyGenesisState()
	require.NoError(t, err)
	web3Service.preGenesisState = genSt
	if err := web3Service.preGenesisState.SetEth1Data(&ethpb.Eth1Data{}); err != nil {
		t.Fatal(err)
	}
	web3Service.chainStartData.ChainstartDeposits = []*ethpb.Deposit{}
	web3Service.depositTrie, err = trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	require.NoError(t, err)

	logsToBeProcessed = append(logs[:depositsWanted-8], logs[depositsWanted-2:]...)
	// We purposely miss processing the middle 7 logs so that the service, re-requests them.
	for _, log := range logsToBeProcessed {
		err = web3Service.ProcessLog(context.Background(), log)
		require.NoError(t, err)
		web3Service.latestEth1Data.LastRequestedBlock = log.BlockNumber
	}

	assert.Equal(t, int64(depositsWanted-1), web3Service.lastReceivedMerkleIndex, "missing logs were not re-requested")

	hook.Reset()
}

func TestConsistentGenesisState(t *testing.T) {
	t.Skip("Incorrect test setup")
	testAcc, err := contracts.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	beaconDB, _ := testDB.SetupDB(t)
	web3Service := newPowchainService(t, testAcc, beaconDB)

	testAcc.Backend.Commit()
	err = testAcc.Backend.AdjustTime(time.Duration(int64(time.Now().Nanosecond())))
	require.NoError(t, err)

	testutil.ResetCache()
	deposits, _, err := testutil.DeterministicDepositsAndKeys(uint64(depositsReqForChainStart))
	require.NoError(t, err)

	_, roots, err := testutil.DeterministicDepositTrie(len(deposits))
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	go web3Service.run(ctx.Done())

	// 64 Validators are used as size required for beacon-chain to start. This number
	// is defined in the deposit contract as the number required for the testnet. The actual number
	// is 2**14.
	for i := 0; i < depositsReqForChainStart; i++ {
		data := deposits[i].Data
		testAcc.TxOpts.Value = contracts.Amount32Eth()
		testAcc.TxOpts.GasLimit = 1000000
		_, err = testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature, roots[i])
		require.NoError(t, err, "Could not deposit to deposit contract")

		testAcc.Backend.Commit()
	}

	for i := uint64(0); i < params.BeaconConfig().Eth1FollowDistance; i++ {
		testAcc.Backend.Commit()
	}

	time.Sleep(2 * time.Second)
	require.Equal(t, true, web3Service.chainStartData.Chainstarted, fmt.Sprintf("Service hasn't chainstarted yet with a block height of %d", web3Service.latestEth1Data.BlockHeight))

	// Advance 10 blocks.
	for i := 0; i < 10; i++ {
		testAcc.Backend.Commit()
	}

	// New db to prevent registration error.
	newBeaconDB, _ := testDB.SetupDB(t)

	newWeb3Service := newPowchainService(t, testAcc, newBeaconDB)
	go newWeb3Service.run(ctx.Done())

	time.Sleep(2 * time.Second)
	require.Equal(t, true, newWeb3Service.chainStartData.Chainstarted, fmt.Sprintf("Service hasn't chainstarted yet with a block height of %d", newWeb3Service.latestEth1Data.BlockHeight))

	diff, _ := messagediff.PrettyDiff(web3Service.chainStartData.Eth1Data, newWeb3Service.chainStartData.Eth1Data)
	assert.Equal(t, "", diff, "Two services have different eth1data")

	cancel()
}

func TestCheckForChainstart_NoValidator(t *testing.T) {
	hook := logTest.NewGlobal()
	testAcc, err := contracts.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	beaconDB, _ := testDB.SetupDB(t)
	s := newPowchainService(t, testAcc, beaconDB)
	s.checkForChainstart([32]byte{}, nil, 0)
	require.LogsDoNotContain(t, hook, "Could not determine active validator count from pre genesis state")
}

func newPowchainService(t *testing.T, eth1Backend *contracts.TestAccount, beaconDB db.Database) *Service {
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	web3Service, err := NewService(context.Background(), &Web3ServiceConfig{
		HTTPEndPoint:    endpoint,
		DepositContract: eth1Backend.ContractAddr,
		BeaconDB:        beaconDB,
		DepositCache:    depositCache,
	})
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")
	web3Service = setDefaultMocks(web3Service)
	web3Service.depositContractCaller, err = contracts.NewDepositContractCaller(eth1Backend.ContractAddr, eth1Backend.Backend)
	require.NoError(t, err)

	web3Service.rpcClient = &mockPOW.RPCClient{Backend: eth1Backend.Backend}
	web3Service.eth1DataFetcher = &goodFetcher{backend: eth1Backend.Backend}
	web3Service.httpLogger = &goodLogger{backend: eth1Backend.Backend}
	params.SetupTestConfigCleanup(t)
	bConfig := params.MinimalSpecConfig()
	bConfig.MinGenesisTime = 0
	params.OverrideBeaconConfig(bConfig)
	web3Service.headerChan = make(chan *gethTypes.Header)
	return web3Service
}
