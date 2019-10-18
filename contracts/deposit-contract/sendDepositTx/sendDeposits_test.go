package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"testing"
	"time"
	"context"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	prysmKeyStore "github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/testutil"
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
	return validatorKeys 

}

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

func sendDeposits(testAcc *contracts.TestAccount, validatorKeys map[string]*prysmKeyStore.Key,
	numberOfDeposits int64) {
	depositAmountInGwei := contracts.Amount32Eth.Uint64()  //how to fix an error here?

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

func TestEndtoEndDeposits(t *testing.T) {
	log = logrus.WithField("prefix", "main")
	testutil.ResetCache()
	testAcc, err := contracts.Setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
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
			testAcc.ContractAddr,
		},
	}

	logs, err := testAcc.Backend.FilterLogs(context.Background(), query) //context backgrround ?
	if err != nil {
		t.Fatalf("Unable to retrieve logs %v", err)
	}
	if len(logs) == 0 {
		t.Fatal("no logs")
	}


    // TODO to check deposit count
	// func (_DepositContract *DepositContractCaller) DepositCount(opts *bind.CallOpts) (*big.Int, error) {
	// 	var (
	// 		ret0 = new(*big.Int)
	// 	)
	// 	out := ret0
	// 	err := _DepositContract.contract.Call(opts, out, "deposit_count")
	// 	return *ret0, err
	// }

}
