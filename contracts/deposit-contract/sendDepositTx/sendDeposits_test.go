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
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	prysmKeyStore "github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
)

func generateValidators(count uint64) map[string]*prysmKeyStore.Key {
	validatorKeys := make(map[string]*prysmKeyStore.Key)
	for i := uint64(0); i < count; i++ {
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

func sendDeposits(t testing.T, testAcc *contracts.TestAccount, validatorKeys map[string]*prysmKeyStore.Key,
	numberOfDeposits, numberofValidators uint64) ([]*ethpb.Deposit_Data depositData) {
	
	depositData := map([]*ethpb.Deposit_Data, 0, numberofValidators)
	depositAmountInGwei := contracts.Amount32Eth.Uint64()  //how to fix an error here?

	depositDelay := int64(1)
	depositContractAddrStr := testAcc.ContractAddr.Hex()

	for _, validatorKey := range validatorKeys {
		data, err := prysmKeyStore.DepositInput(validatorKey, validatorKey, depositAmountInGwei)
		if err != nil {
			t.Fatalf("Could not generate deposit input data: %v", err)
		}
		depositData = append(depositData, data)

		for i := uint64(0); i < numberOfDeposits; i++ {
			tx, err := testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature)
			if err != nil {
				t.Fatalf("unable to send transaction to contract", err)
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

	validatorsWanted := uint64(16)
	validatorKeys := generateValidators(validatorsWanted)

	numberOfDeposits := uint64(16)
	

	testAcc.TxOpts.Value = contracts.Amount32Eth()
	testAcc.TxOpts.GasLimit = 1000000

	depositData := sendDeposits(t, testAcc, validatorKeys, numberOfDeposits, numberofValidators)


	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			testAcc.ContractAddr,
		},
	}

	// should i have 256 logs here?
	logs, err := testAcc.Backend.FilterLogs(context.Background(), query) 
	if err != nil {
		t.Fatalf("Unable to retrieve logs %v", err)
	}
	if len(logs) == 0 {
		t.Fatal("no logs")
	}

	
	//Match the deposit count, 
	depositCount, err := testAcc.Contract.DepositContractCaller.DepositCount(bind.NewCallOpts()).Uint64()
	if err != nil {
		t.Fatalf("Unable to retrieve deposit count from deposit contract %v", err)
	}
	totalNumberOfDeposits := validatorsWanted * numberOfDeposits
	if depositCount != totalNumberDeposits {
		t.Errorf("Incorrect depositCount, expected %v, got %v", totalNumberOfDeposits, depositCount)
	}

	

	//TODO 
	
	//2.validate deposit data and ensure it is accurate and the same data you sent over in a transaction,
	//type DepositContractDepositEvent struct 

	//func (s *Service) ProcessDepositLog(ctx context.Context, depositLog gethTypes.Log) error {
	depositEventIterator, err := testAcc.Contract.DepositContractFilterer.FilterDepositEvent(bind.NewCallOpts())
	if err != nil {
		t.Fatalf("Unable to retrieve deposit event iterator %v", err)
	}
    //how rto iterate with Iterator with counter 
	for EventIterator.Next  {
		current := depositEventIterator.Event
		counter +16 
		data  := depositData[i]
	}




	// validate merkle root of trie in the deposit contract
	// deposits, depositDataRoots, _ := testutil.SetupInitialDeposits(t, 1)

	// trie, err := trieutil.GenerateTrieFromItems([][]byte{depositDataRoots[0][:]}, int(params.BeaconConfig().DepositContractTreeDepth))
	// if err != nil {
	// 	log.Error(err)
	// }
	
	root, err := testAcc.Contract.DepositContractCaller.GetHashTreeRoot(bind.NewCallOpts())
	if err != nil {
		t.Fatalf("Unable to retrieve hash tree root from deposit contract %v", err)
	}

}
