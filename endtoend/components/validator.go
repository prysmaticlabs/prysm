package components

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
	"github.com/prysmaticlabs/prysm/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/endtoend/params"
	"github.com/prysmaticlabs/prysm/endtoend/types"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

// StartValidators sends the deposits to the eth1 chain and starts the validator clients.
func StartValidators(
	t *testing.T,
	config *types.E2EConfig,
	keystorePath string,
) []int {
	binaryPath, found := bazel.FindBinary("validator", "validator")
	if !found {
		t.Fatal("validator binary not found")
	}

	// Always using genesis count since using anything else would be difficult to test for.
	validatorNum := int(params.BeaconConfig().MinGenesisActiveValidatorCount)
	beaconNodeNum := e2e.TestParams.BeaconNodeCount
	if validatorNum%beaconNodeNum != 0 {
		t.Fatal("Validator count is not easily divisible by beacon node count.")
	}

	processIDs := make([]int, beaconNodeNum)
	validatorsPerNode := validatorNum / beaconNodeNum
	for n := 0; n < beaconNodeNum; n++ {
		file, err := helpers.DeleteAndCreateFile(e2e.TestParams.LogPath, fmt.Sprintf(e2e.ValidatorLogFileName, n))
		if err != nil {
			t.Fatal(err)
		}
		args := []string{
			fmt.Sprintf("--datadir=%s/eth2-val-%d", e2e.TestParams.TestPath, n),
			fmt.Sprintf("--log-file=%s", file.Name()),
			fmt.Sprintf("--interop-num-validators=%d", validatorsPerNode),
			fmt.Sprintf("--interop-start-index=%d", validatorsPerNode*n),
			fmt.Sprintf("--monitoring-port=%d", 9280+n),
			fmt.Sprintf("--beacon-rpc-provider=localhost:%d", e2e.TestParams.BeaconNodeRPCPort+n),
			"--force-clear-db",
		}
		args = append(args, featureconfig.E2EValidatorFlags...)
		args = append(args, config.ValidatorFlags...)

		cmd := exec.Command(binaryPath, args...)
		t.Logf("Starting validator client %d with flags: %s", n, strings.Join(args[2:], " "))
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		processIDs[n] = cmd.Process.Pid
	}

	client, err := rpc.DialHTTP(fmt.Sprintf("http://127.0.0.1:%d", e2e.TestParams.Eth1RPCPort))
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

	contract, err := contracts.NewDepositContract(e2e.TestParams.ContractAddress, web3)
	if err != nil {
		t.Fatal(err)
	}

	deposits, _, _ := testutil.DeterministicDepositsAndKeys(uint64(validatorNum))
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

	return processIDs
}
