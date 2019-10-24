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
	numberofValidators uint64) ([]*ethpb.Deposit_Data depositData) {
	
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

    // 256 validators - each deposits ones for simplicity
	validatorsWanted := uint64(256)
	validatorKeys := generateValidators(validatorsWanted)

	
	

	testAcc.TxOpts.Value = contracts.Amount32Eth()
	testAcc.TxOpts.GasLimit = 1000000

	depositData := sendDeposits(t, testAcc, validatorKeys, validatorsWanted )


	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			testAcc.ContractAddr,
		},
	}

	logs, err := testAcc.Backend.FilterLogs(context.Background(), query) 
	if err != nil {
		t.Fatalf("Unable to retrieve logs %v", err)
	}
	if len(logs) == 0 {
		t.Fatal("no logs")
	}

	if len(logs) != validatorsWanted  {
		t.Fatal("No sufficient number of logs")
	}
	
	//validate deposit data 
	for i, log := range logs {
		
		data := depositData[i]

		loggedPubkey, withCreds, _, loggedSig, index, err := contracts.UnpackDepositLogData(log.Data)
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

	//Match the deposit count, 
	depositCount, err := testAcc.Contract.DepositContractCaller.DepositCount(bind.NewCallOpts()).Uint64()
	if err != nil {
		t.Fatalf("Unable to retrieve deposit count from deposit contract %v", err)
	}
	if depositCount != validatorsWanted  {
		t.Errorf("Incorrect depositCount, expected %v, got %v", validatorsWanted, depositCount)
	}

	// validate merkle root of trie in the deposit contract
	lock.Lock()
	defer lock.Unlock()

    //make deposits
	deposits := map([]*ethpb.Deposit, 0, validatorsWanted)
	for _, data := range depositData {
		deposit := &ethpb.Deposit{
			Data: data,
		}
        deposits = append(deposits, deposit)
	}

	//generate deposit proof
	_, root:= helpers.GenerateDepositProof(t, deposits[0:validatorsWanted])
   
	//verify merkle proof for each deposit
	receiptRoot := eth1Data.DepositRoot
	leaf, err := ssz.HashTreeRoot(deposit.Data)
	if err != nil {
		return errors.Wrap(err, "could not tree hash deposit data")
	}
	if ok := trieutil.VerifyMerkleProof(
		receiptRoot,
		leaf[:],
		int(eth1Data.DepositCount-1),
		deposit.Proof,
	); !ok {
		return fmt.Errorf(
			"deposit merkle branch of deposit root did not verify for root: %#x",
			receiptRoot,
		)
	}

	

		
		// root, err := testAcc.Contract.DepositContractCaller.GetHashTreeRoot(bind.NewCallOpts())
		// if err != nil {
		// 	t.Fatalf("Unable to retrieve hash tree root from deposit contract %v", err)
		// }
	}

}
