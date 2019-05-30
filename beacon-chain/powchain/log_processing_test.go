package powchain

import (
	"bytes"
	"context"
	"encoding/binary"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{
		CacheTreeHash: false,
	})

	logrus.SetLevel(logrus.DebugLevel)
}

func TestProcessDepositLog_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	testAcc, err := setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: testAcc.contractAddr,
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
		HTTPLogger:      &goodLogger{},
		ContractBackend: testAcc.backend,
		BeaconDB:        &db.BeaconDB{},
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	testAcc.backend.Commit()

	var pubkey [48]byte
	var withdrawalCreds [32]byte
	var sig [96]byte
	copy(pubkey[:], []byte("testing"))
	copy(sig[:], []byte("testing"))
	copy(withdrawalCreds[:], []byte("testing"))

	data := &pb.DepositData{
		Pubkey:                pubkey[:],
		Signature:             sig[:],
		WithdrawalCredentials: withdrawalCreds[:],
	}

	testAcc.txOpts.Value = amount32Eth
	testAcc.txOpts.GasLimit = 1000000
	if _, err := testAcc.contract.Deposit(testAcc.txOpts, data.Pubkey, data.WithdrawalCredentials, data.Signature); err != nil {
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
	testutil.AssertLogsContain(t, hook, "Deposit registered from deposit contract")

	hook.Reset()
}

func TestProcessDepositLog_InsertsPendingDeposit(t *testing.T) {
	hook := logTest.NewGlobal()
	testAcc, err := setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: testAcc.contractAddr,
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
		HTTPLogger:      &goodLogger{},
		ContractBackend: testAcc.backend,
		BeaconDB:        &db.BeaconDB{},
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	testAcc.backend.Commit()

	var pubkey [48]byte
	var withdrawalCreds [32]byte
	var sig [96]byte
	copy(pubkey[:], []byte("testing"))
	copy(sig[:], []byte("testing"))
	copy(withdrawalCreds[:], []byte("testing"))

	data := &pb.DepositData{
		Pubkey:                pubkey[:],
		Signature:             sig[:],
		WithdrawalCredentials: withdrawalCreds[:],
	}

	testAcc.txOpts.Value = amount32Eth
	testAcc.txOpts.GasLimit = 1000000

	if _, err := testAcc.contract.Deposit(testAcc.txOpts, data.Pubkey, data.WithdrawalCredentials, data.Signature); err != nil {
		t.Fatalf("Could not deposit to deposit contract %v", err)
	}

	if _, err := testAcc.contract.Deposit(testAcc.txOpts, data.Pubkey, data.WithdrawalCredentials, data.Signature); err != nil {
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
	web3Service.ProcessDepositLog(logs[1])
	pendingDeposits := web3Service.beaconDB.PendingDeposits(context.Background(), nil /*blockNum*/)
	if len(pendingDeposits) != 2 {
		t.Errorf("Unexpected number of deposits. Wanted 2 deposit, got %+v", pendingDeposits)
	}
	hook.Reset()
}

func TestUnpackDepositLogData_OK(t *testing.T) {
	testAcc, err := setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: testAcc.contractAddr,
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
		HTTPLogger:      &goodLogger{},
		ContractBackend: testAcc.backend,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	testAcc.backend.Commit()

	if err := web3Service.initDataFromContract(); err != nil {
		t.Fatalf("Could not init from contract: %v", err)
	}

	var pubkey [48]byte
	var withdrawalCreds [32]byte
	var sig [96]byte
	copy(pubkey[:], []byte("pubkey"))
	copy(sig[:], []byte("sig"))
	copy(withdrawalCreds[:], []byte("withdrawCreds"))

	data := &pb.DepositData{
		Pubkey:                pubkey[:],
		Signature:             sig[:],
		WithdrawalCredentials: withdrawalCreds[:],
	}

	testAcc.txOpts.Value = amount32Eth
	testAcc.txOpts.GasLimit = 1000000
	if _, err := testAcc.contract.Deposit(testAcc.txOpts, data.Pubkey, data.WithdrawalCredentials, data.Signature); err != nil {
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

	loggedPubkey, withCreds, _, loggedSig, index, err := contracts.UnpackDepositLogData(logz[0].Data)
	if err != nil {
		t.Fatalf("Unable to unpack logs %v", err)
	}

	if binary.LittleEndian.Uint64(index) != 0 {
		t.Errorf("Retrieved merkle tree index is incorrect %d", index)
	}

	if !bytes.Equal(loggedPubkey, data.Pubkey) {
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
	testAcc, err := setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: testAcc.contractAddr,
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
		HTTPLogger:      &goodLogger{},
		ContractBackend: testAcc.backend,
		BeaconDB:        &db.BeaconDB{},
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	testAcc.backend.Commit()
	testAcc.backend.AdjustTime(time.Duration(int64(time.Now().Nanosecond())))

	var pubkey [48]byte
	var withdrawalCreds [32]byte
	var sig [96]byte
	copy(pubkey[:], []byte("pubkey"))
	copy(sig[:], []byte("sig"))
	copy(withdrawalCreds[:], []byte("withdrawCreds"))

	data := &pb.DepositData{
		Pubkey:                pubkey[:],
		Signature:             sig[:],
		WithdrawalCredentials: withdrawalCreds[:],
	}

	testAcc.txOpts.Value = amount32Eth
	testAcc.txOpts.GasLimit = 1000000

	// 8 Validators are used as size required for beacon-chain to start. This number
	// is defined in the deposit contract as the number required for the testnet. The actual number
	// is 2**14
	for i := 0; i < depositsReqForChainStart; i++ {
		testAcc.txOpts.Value = amount32Eth
		if _, err := testAcc.contract.Deposit(testAcc.txOpts, data.Pubkey, data.WithdrawalCredentials, data.Signature); err != nil {
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

func TestProcessETH2GenesisLog_8UniquePubkeys(t *testing.T) {
	hook := logTest.NewGlobal()
	testAcc, err := setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: testAcc.contractAddr,
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
		HTTPLogger:      &goodLogger{},
		ContractBackend: testAcc.backend,
		BeaconDB:        &db.BeaconDB{},
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	testAcc.backend.Commit()
	testAcc.backend.AdjustTime(time.Duration(int64(time.Now().Nanosecond())))

	// 8 Validators are used as size required for beacon-chain to start. This number
	// is defined in the deposit contract as the number required for the testnet. The actual number
	// is 2**14
	for i := 0; i < depositsReqForChainStart; i++ {
		var pubkey [48]byte
		binary.LittleEndian.PutUint64(pubkey[:], uint64(i))

		var withdrawalCreds [32]byte
		var sig [96]byte
		copy(pubkey[:], []byte("pubkey"))
		copy(sig[:], []byte("sig"))
		copy(withdrawalCreds[:], []byte("withdrawCreds"))

		data := &pb.DepositData{
			Pubkey:                pubkey[:],
			Signature:             sig[:],
			WithdrawalCredentials: withdrawalCreds[:],
		}

		testAcc.txOpts.Value = amount32Eth
		testAcc.txOpts.GasLimit = 1000000
		if _, err := testAcc.contract.Deposit(testAcc.txOpts, data.Pubkey, data.WithdrawalCredentials, data.Signature); err != nil {
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

func TestUnpackETH2GenesisLogData_OK(t *testing.T) {
	testAcc, err := setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: testAcc.contractAddr,
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
		HTTPLogger:      &goodLogger{},
		ContractBackend: testAcc.backend,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	testAcc.backend.Commit()

	testAcc.backend.AdjustTime(time.Duration(int64(time.Now().Nanosecond())))

	var pubkey [48]byte
	var withdrawalCreds [32]byte
	var sig [96]byte
	copy(pubkey[:], []byte("pubkey"))
	copy(sig[:], []byte("sig"))
	copy(withdrawalCreds[:], []byte("withdrawCreds"))

	data := &pb.DepositData{
		Pubkey:                pubkey[:],
		Signature:             sig[:],
		WithdrawalCredentials: withdrawalCreds[:],
	}

	testAcc.txOpts.Value = amount32Eth
	testAcc.txOpts.GasLimit = 1000000

	// 8 Validators are used as size required for beacon-chain to start. This number
	// is defined in the deposit contract as the number required for the testnet.
	for i := 0; i < depositsReqForChainStart; i++ {
		testAcc.txOpts.Value = amount32Eth
		if _, err := testAcc.contract.Deposit(testAcc.txOpts, data.Pubkey, data.WithdrawalCredentials, data.Signature); err != nil {
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

	_, _, timestampData, err := contracts.UnpackChainStartLogData(logs[len(logs)-1].Data)
	if err != nil {
		t.Fatalf("Unable to unpack logs %v", err)
	}

	timestamp := binary.LittleEndian.Uint64(timestampData)

	if timestamp > uint64(time.Now().Unix()) {
		t.Errorf("Timestamp from log is higher than the current time %d > %d", timestamp, time.Now().Unix())
	}
}

func TestHasETH2GenesisLogOccurred_OK(t *testing.T) {
	testAcc, err := setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: testAcc.contractAddr,
		Reader:          &goodReader{},
		Logger:          testAcc.backend,
		HTTPLogger:      &goodLogger{},
		ContractBackend: testAcc.backend,
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	testAcc.backend.Commit()

	testAcc.backend.AdjustTime(time.Duration(int64(time.Now().Nanosecond())))

	var pubkey [48]byte
	var withdrawalCreds [32]byte
	var sig [96]byte
	copy(pubkey[:], []byte("pubkey"))
	copy(sig[:], []byte("sig"))
	copy(withdrawalCreds[:], []byte("withdrawCreds"))

	data := &pb.DepositData{
		Pubkey:                pubkey[:],
		Signature:             sig[:],
		WithdrawalCredentials: withdrawalCreds[:],
	}

	testAcc.txOpts.Value = amount32Eth
	testAcc.txOpts.GasLimit = 1000000

	ok, err := web3Service.HasChainStartLogOccurred()
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
		if _, err := testAcc.contract.Deposit(testAcc.txOpts, data.Pubkey, data.WithdrawalCredentials, data.Signature); err != nil {
			t.Fatalf("Could not deposit to deposit contract %v", err)
		}
		testAcc.backend.Commit()
	}
	ok, err = web3Service.HasChainStartLogOccurred()
	if err != nil {
		t.Fatalf("Could not check if chain start log occurred: %v", err)
	}
	if !ok {
		t.Error("Expected chain start log to have occurred")
	}
}

func TestETH1DataGenesis_OK(t *testing.T) {
	testAcc, err := setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: testAcc.contractAddr,
		Reader:          &goodReader{},
		Logger:          testAcc.backend,
		HTTPLogger:      &goodLogger{},
		ContractBackend: testAcc.backend,
		BeaconDB:        &db.BeaconDB{},
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	testAcc.backend.Commit()

	testAcc.backend.AdjustTime(time.Duration(int64(time.Now().Nanosecond())))

	var pubkey [48]byte
	var withdrawalCreds [32]byte
	var sig [96]byte
	copy(pubkey[:], []byte("pubkey"))
	copy(sig[:], []byte("sig"))
	copy(withdrawalCreds[:], []byte("withdrawCreds"))

	data := &pb.DepositData{
		Pubkey:                pubkey[:],
		Signature:             sig[:],
		WithdrawalCredentials: withdrawalCreds[:],
	}

	testAcc.txOpts.Value = amount32Eth
	testAcc.txOpts.GasLimit = 1000000

	ok, err := web3Service.HasChainStartLogOccurred()
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
		if _, err := testAcc.contract.Deposit(testAcc.txOpts, data.Pubkey, data.WithdrawalCredentials, data.Signature); err != nil {
			t.Fatalf("Could not deposit to deposit contract %v", err)
		}
		testAcc.backend.Commit()
	}
	ok, err = web3Service.HasChainStartLogOccurred()
	if err != nil {
		t.Fatalf("Could not check if chain start log occurred: %v", err)
	}
	if !ok {
		t.Error("Expected chain start log to have occurred")
	}

	eth2GenesisIterator, err := testAcc.contract.FilterEth2Genesis(nil)
	if err != nil {
		t.Fatalf("Could not create chainstart iterator: %v", err)
	}

	defer eth2GenesisIterator.Close()
	eth2GenesisIterator.Next()
	chainStartlog := eth2GenesisIterator.Event

	expectedETH1Data := &pb.Eth1Data{
		BlockRoot:   chainStartlog.Raw.BlockHash[:],
		DepositRoot: chainStartlog.DepositRoot[:],
	}

	// We add in another 8 deposits after chainstart.
	for i := 0; i < depositsReqForChainStart; i++ {
		testAcc.txOpts.Value = amount32Eth
		if _, err := testAcc.contract.Deposit(testAcc.txOpts, data.Pubkey, data.WithdrawalCredentials, data.Signature); err != nil {
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

	for _, log := range logs {
		web3Service.ProcessLog(log)
	}

	if !proto.Equal(expectedETH1Data, web3Service.ChainStartETH1Data()) {
		t.Errorf("Saved Chainstart eth1data not the expected chainstart eth1data, got: %v but expected: %v",
			web3Service.ChainStartETH1Data(), expectedETH1Data)
	}
}
