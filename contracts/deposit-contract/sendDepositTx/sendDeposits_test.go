package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/go-ssz"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	prysmKeyStore "github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
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

func sendDeposits(t *testing.T, testAcc *contracts.TestAccount, validatorKeys map[string]*prysmKeyStore.Key,
	numDeposits uint64) []*ethpb.Deposit {

	deposits := make([]*ethpb.Deposit, 0, numDeposits)
	depositAmountInGwei := testAcc.TxOpts.Value.Uint64()

	depositDelay := int64(1)
	depositContractAddrStr := testAcc.ContractAddr.Hex()
	i := 0
	for _, validatorKey := range validatorKeys {
		 
		data, err := prysmKeyStore.DepositInput(validatorKey, validatorKey, depositAmountInGwei)
		if err != nil {
			t.Fatalf("Could not generate deposit input data: %v", err)
		}

		tx, err := testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature)
		if err != nil {
			t.Fatalf("unable to send transaction to contract: %v", err)
		}

		testAcc.Backend.Commit()
		
		//lgos do not show up in console 
		log.WithFields(logrus.Fields{
			"Transaction Hash": fmt.Sprintf("%#x", tx.Hash()),
		}).Infof("Deposit %d sent to contract address %v for validator with a public key %#x", i, depositContractAddrStr, validatorKey.PublicKey.Marshal())

		i++

		deposits = append(deposits, &ethpb.Deposit{
			Data: data,
		})

		time.Sleep(time.Duration(depositDelay) * time.Second)
	}
	return deposits
}

func TestEndtoEndDeposits(t *testing.T) {
	log = logrus.WithField("prefix", "main")
	testutil.ResetCache()
	testAcc, err := contracts.Setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}

	testAcc.Backend.Commit()

	// 256 validators - each validator makes one deposit for simplicity
	numDeposits := uint64(256)
	validatorKeys := generateValidators(numDeposits)

	testAcc.TxOpts.Value = contracts.Amount32Eth()
	testAcc.TxOpts.GasLimit = 1000000

	deposits := sendDeposits(t, testAcc, validatorKeys, numDeposits)

	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			testAcc.ContractAddr,
		},
	}
	//fetch logs
	logs, err := testAcc.Backend.FilterLogs(context.Background(), query)
	if err != nil {
		t.Fatalf("Unable to retrieve logs %v", err)
	}
	if len(logs) == 0 {
		t.Fatal("no logs")
	}

	if len(logs) != int(numDeposits) {
		t.Fatal("No sufficient number of logs")
	}

	if len(logs) != len(deposits) {
		t.Fatal("Number of logs does not match the number of deposits")
	}

	//validate deposit data
	for i, log := range logs {
		loggedPubkey, withCreds, _, loggedSig, index, err := contracts.UnpackDepositLogData(log.Data)
		if err != nil {
			t.Fatalf("Unable to unpack logs %v", err)
		}

		if binary.LittleEndian.Uint64(index) != uint64(i + 1) {
			t.Errorf("Retrieved merkle tree index is incorrect %d", index)
		}

		if !bytes.Equal(loggedPubkey, deposits[i].Data.PublicKey) {
			t.Errorf("Pubkey is not the same as the data that was put in %v", loggedPubkey)
		}

		if !bytes.Equal(loggedSig, deposits[i].Data.Signature) {
			t.Errorf("Proof of Possession is not the same as the data that was put in %v", loggedSig)
		}

		if !bytes.Equal(withCreds, deposits[i].Data.WithdrawalCredentials) {
			t.Errorf("Withdrawal Credentials is not the same as the data that was put in %v", withCreds)
		}
	}
	//generate deposit proof for every deposit
	var root [32]byte
	deposits, root = testutil.GenerateDepositProof(t, deposits[0:numDeposits])

	//Generate Eth1Data
	eth1Data := &ethpb.Eth1Data{
		BlockHash:    root[:],
		DepositRoot:  root[:],
		DepositCount: uint64(len(deposits)),
	}

	//verify merkle proof for each deposit
	receiptRoot := eth1Data.DepositRoot
	for _, deposit := range deposits {
		leaf, err := ssz.HashTreeRoot(deposit.Data)
		if err != nil {
			t.Fatalf("Unable tree hash deposit data: %v", err)
		}
		if ok := trieutil.VerifyMerkleProof(
			receiptRoot,
			leaf[:],
			int(eth1Data.DepositCount-1),
			deposit.Proof,
		); !ok {
			t.Fatalf(
				"Unable verify deposit merkle branch of deposit root for root: %#x",
				receiptRoot)
		}
	}
}
