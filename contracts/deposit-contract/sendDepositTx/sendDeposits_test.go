package main

import (
	"crypto/rand"
	"io/ioutil"
	"math/big"
	"testing"
	"time"
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	prysmKeyStore "github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/bazel-prysm/beacon-chain/powchain"
	"github.com/sirupsen/logrus"
)

func generateValidators(count int64) map[string]*prysmKeyStore.Key {
	validatorKeys := make(map[string]*prysmKeyStore.Key)
	for i := int64(0); i < count; i++ {
		validatorKey, err := prysmKeyStore.NewKey(rand.Reader)
		if err != nil {
			log.Errorf("Could not generate random key: %v", err)
		}
		validatorKeys[hex.EncodeToString(validatorKey.PublicKey.Marshal())] = validatorKey
	}
}

//should I explicitly return err here?
func sendDeposits(testAcc *contracts.TestAccount, validatorKeys map[string]*prysmKeyStore.Key,
	numberOfDeposits int64) {
	depositAmountInGwei := uint64(contracts.Amount32Eth.Int64())

	depositDelay := int64(1)
	depositContractAddrStr := testAcc.ContractAddr.Hex()

	for _, validatorKey := range validatorKeys {
		data, err := prysmKeyStore.DepositInput(validatorKey, validatorKey, depositAmountInGwei)
		if err != nil {
			log.Errorf("Could not generate deposit input data: %v", err)
			continue
		}

		for i := int64(0); i < numberOfDeposits; i++ {
			tx, err := testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature)
			if err != nil {
				log.Error("unable to send transaction to contract")
			}

			testAcc.Backend.Commit()

			log.WithFields(logrus.Fields{
				"Transaction Hash": fmt.Sprintf("%#x", tx.Hash()),
			}).Infof("Deposit %d sent to contract address %v for validator with a public key %#x", i, depositContractAddrStr, validatorKey.PublicKey.Marshal())

			time.Sleep(time.Duration(depositDelay) * time.Second)
		}
	}
}

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

func TestEndtoEndDeposits(t *testing.T) {
	log = logrus.WithField("prefix", "main")
	testutil.ResetCache()
	testAcc, err := contracts.Setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}

	web3Service, err := powchain.NewService(context.Background(), &Web3ServiceConfig{
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

	validatorsWanted := int64(16)
	validatorKeys := generateValidators(validatorsWanted)

	numberOfDeposits := int64(16)

	testAcc.TxOpts.Value = contracts.Amount32Eth()
	testAcc.TxOpts.GasLimit = 1000000

	sendDeposits(testAcc, validatorKeys, numberOfDeposits)

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

	web3Service.chainStarted = true

	for _, log := range logs {
		if err := web3Service.ProcessDepositLog(context.Background(), log); err != nil {
			t.Fatal("Unable to process deposit log %v", err)
		}
	}
	totalNumberOfDeposits := validatorsWanted * numberOfDeposits
	pendingDeposits := web3Service.depositCache.PendingDeposits(context.Background(), nil /*blockNum*/)
	if len(pendingDeposits) != int(totalNumberOfDeposits) {
		t.Errorf("Unexpected number of deposits. Wanted %v deposits, got %+v", int(totalNumberOfDeposits), pendingDeposits)
	}
}
