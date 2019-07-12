package powchain

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestRandom(t *testing.T) {
	hook := logTest.NewGlobal()
	testAcc, err := contracts.Setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: testAcc.ContractAddr,
		Reader:          &goodReader{},
		Logger:          &goodLogger{},
		HTTPLogger:      &goodLogger{},
		ContractBackend: testAcc.Backend,
		BeaconDB:        &db.BeaconDB{},
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

	deposits, _ := testutil.SetupInitialDeposits(t, uint64(depositsReqForChainStart), true)

	// 64 Validators are used as size required for beacon-chain to start. This number
	// is defined in the deposit contract as the number required for the testnet. The actual number
	// is 2**14
	for i := 0; i < depositsReqForChainStart; i++ {
		data := deposits[i].Data
		testAcc.TxOpts.Value = contracts.Amount32Eth()
		testAcc.TxOpts.GasLimit = 1000000
		if _, err := testAcc.Contract.Deposit(testAcc.TxOpts, data.Pubkey, data.WithdrawalCredentials, data.Signature); err != nil {
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
