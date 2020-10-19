package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	"github.com/prysmaticlabs/prysm/shared/interop"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	"github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)

	os.Exit(m.Run())
}

func sendDeposits(t *testing.T, testAcc *contracts.TestAccount,
	numberOfDeposits, numberOfValidators uint64) []*ethpb.Deposit {

	deposits := make([]*ethpb.Deposit, 0, numberOfValidators)
	depositDelay := int64(1)
	depositContractAddrStr := testAcc.ContractAddr.Hex()

	privKeys, pubKeys, err := interop.DeterministicallyGenerateKeys(0, numberOfValidators)
	require.NoError(t, err, "Unable to generate keys")

	depositData, depositDataRoots, err := interop.DepositDataFromKeys(privKeys, pubKeys)
	require.NoError(t, err, "Unable to generate deposit data from keys")

	for i, data := range depositData {
		dataRoot := [32]byte{}
		copy(dataRoot[:], depositDataRoots[i])

		pubKey := pubKeys[i]

		deposits = append(deposits, &ethpb.Deposit{
			Data: data,
		})

		for j := uint64(0); j < numberOfDeposits; j++ {
			tx, err := testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature, dataRoot)
			require.NoError(t, err, "Unable to send transaction to contract")

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
	require.NoError(t, err, "Unable to set up simulated backend")

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
	require.NoError(t, err, "Unable to retrieve logs")
	require.NotEqual(t, 0, len(logs), "No logs")
	require.Equal(t, int(numberOfDeposits*numberOfValidators), len(logs), "No sufficient number of logs")

	j := 0
	for i, log := range logs {
		loggedPubkey, withCreds, _, loggedSig, index, err := contracts.UnpackDepositLogData(log.Data)
		require.NoError(t, err, "Unable to unpack logs")
		assert.Equal(t, uint64(i), binary.LittleEndian.Uint64(index))
		assert.DeepEqual(t, deposits[j].Data.PublicKey, loggedPubkey, "Pubkey is not the same as the data that was put in %v, i: %d", loggedPubkey, i)
		assert.DeepEqual(t, deposits[j].Data.Signature, loggedSig, "Proof of Possession is not the same as the data that was put in %v, i: %d", loggedSig, i)
		assert.DeepEqual(t, deposits[j].Data.WithdrawalCredentials, withCreds, "Withdrawal Credentials is not the same as the data that was put in %v, i: %d", withCreds, i)

		if i == int(numberOfDeposits)-1 {
			j++
		}
	}

	encodedDeposits := make([][]byte, numberOfValidators*numberOfDeposits)
	for i := 0; i < int(numberOfValidators); i++ {
		hashedDeposit, err := deposits[i].Data.HashTreeRoot()
		require.NoError(t, err, "Could not tree hash deposit data")
		for j := 0; j < int(numberOfDeposits); j++ {
			encodedDeposits[i*int(numberOfDeposits)+j] = hashedDeposit[:]
		}
	}

	depositTrie, err := trieutil.GenerateTrieFromItems(encodedDeposits, params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not generate trie")

	root := depositTrie.Root()

	for i, encodedDeposit := range encodedDeposits {
		proof, err := depositTrie.MerkleProof(i)
		require.NoError(t, err, "Could not generate proof")
		require.Equal(t, true, trieutil.VerifyMerkleBranch(root[:], encodedDeposit, i, proof, params.BeaconConfig().DepositContractTreeDepth),
			"Unable verify deposit merkle branch of deposit root for root: %#x, encodedDeposit: %#x, i : %d", root[:], encodedDeposit, i)
	}
}
