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

const depositGasLimit = 4000000

// StartValidatorClients starts the configured amount of validators, also sending and mining their validator deposits.
// Should only be used on initialization.
func StartValidatorClients(t *testing.T, config *types.E2EConfig, keystorePath string) {
	// Always using genesis count since using anything else would be difficult to test for.
	validatorNum := int(params.BeaconConfig().MinGenesisActiveValidatorCount)
	beaconNodeNum := e2e.TestParams.BeaconNodeCount
	if validatorNum%beaconNodeNum != 0 {
		t.Fatal("Validator count is not easily divisible by beacon node count.")
	}
	validatorsPerNode := validatorNum / beaconNodeNum
	for i := 0; i < beaconNodeNum; i++ {
		go StartNewValidatorClient(t, config, validatorsPerNode, i, validatorsPerNode*i)
	}
}

// StartNewValidatorClient starts a validator client with the passed in configuration.
func StartNewValidatorClient(t *testing.T, config *types.E2EConfig, validatorNum int, index int, offset int) {
	binaryPath, found := bazel.FindBinary("validator", "validator")
	if !found {
		t.Fatal("validator binary not found")
	}

	beaconRPCPort := e2e.TestParams.BeaconNodeRPCPort + index
	if beaconRPCPort >= e2e.TestParams.BeaconNodeRPCPort+e2e.TestParams.BeaconNodeCount {
		// Point any extra validator clients to a node we know is running.
		beaconRPCPort = e2e.TestParams.BeaconNodeRPCPort
	}

	file, err := helpers.DeleteAndCreateFile(e2e.TestParams.LogPath, fmt.Sprintf(e2e.ValidatorLogFileName, index))
	if err != nil {
		t.Fatal(err)
	}
	args := []string{
		fmt.Sprintf("--datadir=%s/eth2-val-%d", e2e.TestParams.TestPath, index),
		fmt.Sprintf("--log-file=%s", file.Name()),
		fmt.Sprintf("--interop-num-validators=%d", validatorNum),
		fmt.Sprintf("--interop-start-index=%d", offset),
		fmt.Sprintf("--monitoring-port=%d", e2e.TestParams.ValidatorMetricsPort+index),
		fmt.Sprintf("--beacon-rpc-provider=localhost:%d", beaconRPCPort),
		"--grpc-headers=dummy=value,foo=bar", // Sending random headers shouldn't break anything.
		"--verbosity=trace",
		"--force-clear-db",
		"--e2e-config",
	}
	args = append(args, featureconfig.E2EValidatorFlags...)
	args = append(args, config.ValidatorFlags...)

	cmd := exec.Command(binaryPath, args...)
	t.Logf("Starting validator client %d with flags: %s", index, strings.Join(args[2:], " "))
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
}

// SendAndMineDeposits sends the requested amount of deposits and mines the chain after to ensure the deposits are seen.
func SendAndMineDeposits(t *testing.T, keystorePath string, validatorNum int, offset int) {
	client, err := rpc.DialHTTP(fmt.Sprintf("http://127.0.0.1:%d", e2e.TestParams.Eth1RPCPort))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	web3 := ethclient.NewClient(client)

	keystoreBytes, err := ioutil.ReadFile(keystorePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := SendDeposits(web3, keystoreBytes, validatorNum, offset); err != nil {
		t.Fatal(err)
	}
	mineKey, err := keystore.DecryptKey(keystoreBytes, "" /*password*/)
	if err != nil {
		t.Fatal(err)
	}
	if err := mineBlocks(web3, mineKey, params.BeaconConfig().Eth1FollowDistance); err != nil {
		t.Fatalf("failed to mine blocks %v", err)
	}
}

// SendDeposits uses the passed in web3 and keystore bytes to send the requested deposits.
func SendDeposits(web3 *ethclient.Client, keystoreBytes []byte, num int, offset int) error {
	txOps, err := bind.NewTransactor(bytes.NewReader(keystoreBytes), "" /*password*/)
	if err != nil {
		return err
	}
	depositInGwei := big.NewInt(int64(params.BeaconConfig().MaxEffectiveBalance))
	txOps.Value = depositInGwei.Mul(depositInGwei, big.NewInt(int64(params.BeaconConfig().GweiPerEth)))
	txOps.GasLimit = depositGasLimit
	nonce, err := web3.PendingNonceAt(context.Background(), txOps.From)
	if err != nil {
		return err
	}
	txOps.Nonce = big.NewInt(int64(nonce))

	contract, err := contracts.NewDepositContract(e2e.TestParams.ContractAddress, web3)
	if err != nil {
		return err
	}

	deposits, _, err := testutil.DeterministicDepositsAndKeys(uint64(num + offset))
	if err != nil {
		return err
	}
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
