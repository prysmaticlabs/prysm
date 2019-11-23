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
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	"github.com/sirupsen/logrus"
)

func generateValidatorKeys(count uint64) map[string]*prysmKeyStore.Key {
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
	numberOfDeposits, numberOfValidators uint64) []*ethpb.Deposit {

	deposits := make([]*ethpb.Deposit, 0, numberOfValidators)
	depositAmountInGwei := testAcc.TxOpts.Value.Uint64()

	depositDelay := int64(1)
	depositContractAddrStr := testAcc.ContractAddr.Hex()

	for _, validatorKey := range validatorKeys {

		data, err := prysmKeyStore.DepositInput(validatorKey, validatorKey, depositAmountInGwei)
		if err != nil {
			t.Fatalf("Could not generate deposit input data: %v", err)
		}
		deposits = append(deposits, &ethpb.Deposit{
			Data: data,
		})

		for i := uint64(0); i < numberOfDeposits; i++ {
			tx, err := testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature)
			if err != nil {
				t.Fatalf("unable to send transaction to contract: %v", err)
			}

			testAcc.Backend.Commit()

			//lgos do not show up in console
			log.WithFields(logrus.Fields{
				"Transaction Hash": fmt.Sprintf("%#x", tx.Hash()),
			}).Infof("Deposit %d sent to contract address %v for validator with a public key %#x", i, depositContractAddrStr, validatorKey.PublicKey.Marshal())

			time.Sleep(time.Duration(depositDelay) * time.Second)
		}
	}
	return deposits
}

func TestEndtoEndDeposits(t *testing.T) {
	testutil.ResetCache()
	testAcc, err := contracts.Setup()
	if err != nil {
		t.Fatalf("Unable to set up simulated backend %v", err)
	}

	testAcc.Backend.Commit()

	// 256 validators - each validator makes one deposit for simplicity

	numberOfValidators := uint64(2)
	validatorKeys := generateValidatorKeys(numberOfValidators)

	testAcc.TxOpts.Value = contracts.Amount32Eth()
	testAcc.TxOpts.GasLimit = 1000000

	numberOfDeposits := uint64(5)
	deposits := sendDeposits(t, testAcc, validatorKeys, numberOfDeposits, numberOfValidators)

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

	if len(logs) != int((numberOfDeposits * numberOfValidators)) {
		t.Fatal("No sufficient number of logs")
	}

	//validate deposit data
	j := 0
	for i, log := range logs {
		loggedPubkey, withCreds, _, loggedSig, index, err := contracts.UnpackDepositLogData(log.Data)
		if err != nil {
			t.Fatalf("Unable to unpack logs %v", err)
		}

		if binary.LittleEndian.Uint64(index) != uint64(i) {
			t.Errorf("Retrieved merkle tree index is incorrect %d", index)
		}

		if !bytes.Equal(loggedPubkey, deposits[j].Data.PublicKey) {
			t.Errorf("Pubkey is not the same as the data that was put in %v, i: %d", loggedPubkey, i)
		}

		if !bytes.Equal(loggedSig, deposits[j].Data.Signature) {
			t.Errorf("Proof of Possession is not the same as the data that was put in %v, i: %d", loggedSig, i)
		}

		if !bytes.Equal(withCreds, deposits[j].Data.WithdrawalCredentials) {
			t.Errorf("Withdrawal Credentials is not the same as the data that was put in %v, i: %d", withCreds, i)
		}
		// if i == 4
		if i == int(numberOfDeposits)-1 {
			j++
		}
	}

	encodedDeposits := make([][]byte, numberOfValidators*numberOfDeposits)
	for i := 0; i < int(numberOfValidators); i++ {

		hashedDeposit, err := ssz.HashTreeRoot(deposits[i].Data)
		if err != nil {
			t.Fatalf("could not tree hash deposit data: %v", err)
		}
		for j := 0; j < int(numberOfDeposits); j++ {
			encodedDeposits[i*int(numberOfDeposits)+j] = hashedDeposit[:]
		}
	}

	depositTrie, err := trieutil.GenerateTrieFromItems(encodedDeposits, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("Could not generate trie: %v", err)
	}

	root := depositTrie.Root()

	for i, encodedDeposit := range encodedDeposits {
		proof, err := depositTrie.MerkleProof(i)
		if err != nil {
			t.Fatalf("Could not generate proof: %v", err)
		}
		if ok := trieutil.VerifyMerkleProof(
			root[:],
			encodedDeposit,
			i,
			proof,
		); !ok {
			t.Fatalf(
				"Unable verify deposit merkle branch of deposit root for root: %#x, encodedDeposit: %#x, i : %d",
				root[:], encodedDeposit, i)
		}

	}
}
