package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	"github.com/prysmaticlabs/prysm/shared/interop"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

func sendDeposits(t *testing.T, testAcc *contracts.TestAccount,
	numberOfDeposits, numberOfValidators uint64) []*ethpb.Deposit {

	deposits := make([]*ethpb.Deposit, 0, numberOfValidators)
	depositDelay := int64(1)
	depositContractAddrStr := testAcc.ContractAddr.Hex()

	privKeys, pubKeys, err := interop.DeterministicallyGenerateKeys(0, numberOfValidators)
	if err != nil {
		t.Fatalf("Unable to generate keys: %v", err)
	}

	depositData, depositDataRoots, err := interop.DepositDataFromKeys(privKeys, pubKeys)
	if err != nil {
		t.Fatalf("Unable to generate deposit data from keys: %v", err)
	}

	for i, data := range depositData {
		dataRoot := [32]byte{}
		copy(dataRoot[:], depositDataRoots[i])

		pubKey := pubKeys[i]

		deposits = append(deposits, &ethpb.Deposit{
			Data: data,
		})

		for j := uint64(0); j < numberOfDeposits; j++ {
			tx, err := testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature, dataRoot)
			if err != nil {
				t.Fatalf("unable to send transaction to contract: %v", err)
			}

			testAcc.Backend.Commit()

			log.WithFields(logrus.Fields{
				"Transaction Hash": fmt.Sprintf("%#x", tx.Hash()),
			}).Infof("Deposit %d sent to contract address %v for validator with a public key %#x", j, depositContractAddrStr, pubKey.Marshal())

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

	testAcc.TxOpts.Value = contracts.Amount32Eth()
	testAcc.TxOpts.GasLimit = 1000000

	numberOfValidators := uint64(2)
	numberOfDeposits := uint64(5)
	deposits := sendDeposits(t, testAcc, numberOfDeposits, numberOfValidators)

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

	if len(logs) != int(numberOfDeposits*numberOfValidators) {
		t.Fatal("No sufficient number of logs")
	}

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
		if ok := trieutil.VerifyMerkleBranch(
			root[:],
			encodedDeposit,
			i,
			proof,
			int(params.BeaconConfig().DepositContractTreeDepth),
		); !ok {
			t.Fatalf(
				"Unable verify deposit merkle branch of deposit root for root: %#x, encodedDeposit: %#x, i : %d",
				root[:], encodedDeposit, i)
		}
	}
}
