package endtoend

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"math/big"
	"os/exec"
	"strings"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

type validatorClientInfo struct {
	processID   int
	monitorPort uint64
}

var validatorLogFileName = "vals-%d.log"

// initializeValidators sends the deposits to the eth1 chain and starts the validator clients.
func initializeValidators(
	t *testing.T,
	config *end2EndConfig,
	keystorePath string,
) []*validatorClientInfo {
	binaryPath, found := bazel.FindBinary("validator", "validator")
	if !found {
		t.Fatal("validator binary not found")
	}

	tmpPath := config.tmpPath
	validatorNum := config.numValidators
	beaconNodeNum := config.numBeaconNodes
	if validatorNum%beaconNodeNum != 0 {
		t.Fatal("Validator count is not easily divisible by beacon node count.")
	}

	valClients := make([]*validatorClientInfo, beaconNodeNum)
	validatorsPerNode := validatorNum / beaconNodeNum
	for n := uint64(0); n < beaconNodeNum; n++ {
		file, err := deleteAndCreateFile(tmpPath, fmt.Sprintf(validatorLogFileName, n))
		if err != nil {
			t.Fatal(err)
		}
		args := []string{
			"--force-clear-db",
			fmt.Sprintf("--interop-num-validators=%d", validatorsPerNode),
			fmt.Sprintf("--interop-start-index=%d", validatorsPerNode*n),
			fmt.Sprintf("--monitoring-port=%d", 9280+n),
			fmt.Sprintf("--datadir=%s/eth2-val-%d", tmpPath, n),
			fmt.Sprintf("--beacon-rpc-provider=localhost:%d", 4200+n),
			fmt.Sprintf("--log-file=%s", file.Name()),
		}
		args = append(args, config.validatorFlags...)

		cmd := exec.Command(binaryPath, args...)
		t.Logf("Starting validator client %d with flags: %s", n, strings.Join(args, " "))
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		valClients[n] = &validatorClientInfo{
			processID:   cmd.Process.Pid,
			monitorPort: 9280 + n,
		}
	}

	client, err := rpc.DialHTTP("http://127.0.0.1:8745")
	if err != nil {
		t.Fatal(err)
	}
	web3 := ethclient.NewClient(client)

	jsonBytes, err := ioutil.ReadFile(keystorePath)
	if err != nil {
		t.Fatal(err)
	}
	txOps, err := bind.NewTransactor(bytes.NewReader(jsonBytes), "" /*password*/)
	if err != nil {
		t.Fatal(err)
	}
	depositInGwei := big.NewInt(int64(params.BeaconConfig().MaxEffectiveBalance))
	txOps.Value = depositInGwei.Mul(depositInGwei, big.NewInt(int64(params.BeaconConfig().GweiPerEth)))
	txOps.GasLimit = 4000000
	nonce, err := web3.PendingNonceAt(context.Background(), txOps.From)
	if err != nil {
		t.Fatal(err)
	}
	txOps.Nonce = big.NewInt(int64(nonce))

	contract, err := contracts.NewDepositContract(config.contractAddr, web3)
	if err != nil {
		t.Fatal(err)
	}

	deposits, _, _ := testutil.DeterministicDepositsAndKeys(validatorNum)
	_, roots, err := testutil.DeterministicDepositTrie(len(deposits))
	if err != nil {
		t.Fatal(err)
	}
	for index, dd := range deposits {
		_, err = contract.Deposit(txOps, dd.Data.PublicKey, dd.Data.WithdrawalCredentials, dd.Data.Signature, roots[index])
		if err != nil {
			t.Errorf("unable to send transaction to contract: %v", err)
		}
		txOps.Nonce = txOps.Nonce.Add(txOps.Nonce, big.NewInt(1))
	}

	keystore, err := keystore.DecryptKey(jsonBytes, "" /*password*/)
	if err != nil {
		t.Fatal(err)
	}

	if err := mineBlocks(web3, keystore, params.BeaconConfig().Eth1FollowDistance); err != nil {
		t.Fatalf("failed to mine blocks %v", err)
	}

	return valClients
}
