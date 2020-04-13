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
	"github.com/pkg/errors"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	"github.com/prysmaticlabs/prysm/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/endtoend/params"
	"github.com/prysmaticlabs/prysm/endtoend/types"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func StartValidatorClients(t *testing.T, config *types.E2EConfig, keystorePath string) []int {
	// Always using genesis count since using anything else would be difficult to test for.
	validatorNum := int(params.BeaconConfig().MinGenesisActiveValidatorCount)
	beaconNodeNum := e2e.TestParams.BeaconNodeCount
	if validatorNum%beaconNodeNum != 0 {
		t.Fatal("Validator count is not easily divisible by beacon node count.")
	}
	processIDs := make([]int, beaconNodeNum)
	validatorsPerNode := validatorNum / beaconNodeNum
	for i := 0; i < beaconNodeNum; i++ {
		pID := StartNewValidatorClient(t, config, validatorsPerNode, i)
		processIDs[i] = pID
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
	if err := SendDeposits(web3, jsonBytes, validatorNum, 0); err != nil {
		t.Fatal(err)
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

// StartNewValidators sends the deposits to the eth1 chain and starts the validator clients.
func StartNewValidatorClient(t *testing.T, config *types.E2EConfig, validatorNum int, index int) int {
	validatorsPerClient := int(params.BeaconConfig().MinGenesisActiveValidatorCount) / e2e.TestParams.BeaconNodeCount
	// Only allow validatorsPerClient count for each validator client.
	if validatorNum != validatorsPerClient {
		return 0
	}
	binaryPath, found := bazel.FindBinary("validator", "validator")
	if !found {
		t.Fatal("validator binary not found")
	}

	beaconRPCPort := e2e.TestParams.BeaconNodeRPCPort + index
	if beaconRPCPort >= e2e.TestParams.BeaconNodeCount+e2e.TestParams.BeaconNodeCount {
		// Point any extra validator clients to a node we know is running.
		beaconRPCPort = e2e.TestParams.BeaconNodeCount
	}

	file, err := helpers.DeleteAndCreateFile(e2e.TestParams.LogPath, fmt.Sprintf(e2e.ValidatorLogFileName, n))
	if err != nil {
		t.Fatal(err)
	}
	args := []string{
		fmt.Sprintf("--datadir=%s/eth2-val-%d", e2e.TestParams.TestPath, index),
		fmt.Sprintf("--log-file=%s", file.Name()),
		fmt.Sprintf("--interop-num-validators=%d", validatorNum),
		fmt.Sprintf("--interop-start-index=%d", validatorNum*index),
		fmt.Sprintf("--monitoring-port=%d", e2e.TestParams.ValidatorMetricsPort+index),
		fmt.Sprintf("--beacon-rpc-provider=localhost:%d", beaconRPCPort),
		"--grpc-headers=dummy=value,foo=bar", // Sending random headers shouldn't break anything.
		"--force-clear-db",
	}
	args = append(args, featureconfig.E2EValidatorFlags...)
	args = append(args, config.ValidatorFlags...)

	cmd := exec.Command(binaryPath, args...)
	t.Logf("Starting validator client %d with flags: %s", n, strings.Join(args[2:], " "))
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	return cmd.Process.Pid
}

func SendDeposits(web3 *ethclient.Client, jsonBytes []byte, num int, offset int) error {
	txOps, err := bind.NewTransactor(bytes.NewReader(jsonBytes), "" /*password*/)
	if err != nil {
		return err
	}
	depositInGwei := big.NewInt(int64(params.BeaconConfig().MaxEffectiveBalance))
	txOps.Value = depositInGwei.Mul(depositInGwei, big.NewInt(int64(params.BeaconConfig().GweiPerEth)))
	txOps.GasLimit = 4000000
	nonce, err := web3.PendingNonceAt(context.Background(), txOps.From)
	if err != nil {
		return err
	}
	txOps.Nonce = big.NewInt(int64(nonce))

	contract, err := contracts.NewDepositContract(e2e.TestParams.ContractAddress, web3)
	if err != nil {
		return err
	}

	deposits, _, _ := testutil.DeterministicDepositsAndKeys(uint64(num + offset))
	_, roots, err := testutil.DeterministicDepositTrie(len(deposits))
	if err != nil {
		return err
	}
	for index, dd := range deposits {
		if index < offset {
			continue
		}
		_, err = contract.Deposit(txOps, dd.Data.PublicKey, dd.Data.WithdrawalCredentials, dd.Data.Signature, roots[index])
		if err != nil {
			return errors.Wrap(err, "unable to send transaction to contract")
		}
		txOps.Nonce = txOps.Nonce.Add(txOps.Nonce, big.NewInt(1))
	}
	return nil
}
