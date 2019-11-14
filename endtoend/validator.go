package endtoend

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"os/exec"
	"path"
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

// initializeValidators sends the deposits to the eth1 chain and starts the validator clients.
func initializeValidators(
	t *testing.T,
	config *end2EndConfig,
	keystorePath string,
	beaconNodes []*beaconNodeInfo,
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
		file, err := os.Create(path.Join(tmpPath, fmt.Sprintf("vals-%d.log", n)))
		if err != nil {
			t.Fatal(err)
		}
		args := []string{
			fmt.Sprintf("--interop-num-validators=%d", validatorsPerNode),
			fmt.Sprintf("--interop-start-index=%d", validatorsPerNode*n),
			fmt.Sprintf("--monitoring-port=%d", 9080+n),
			fmt.Sprintf("--beacon-rpc-provider=localhost:%d", 4000+n),
		}
		cmd := exec.Command(binaryPath, args...)
		cmd.Stdout = file
		cmd.Stderr = file
		t.Logf("Starting validator client with flags %s", strings.Join(args, " "))
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		valClients[n] = &validatorClientInfo{
			processID:   cmd.Process.Pid,
			monitorPort: 9080 + n,
		}
	}

	client, err := rpc.DialHTTP("http://127.0.0.1:8545")
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

	contract, err := contracts.NewDepositContract(config.contractAddr, web3)
	if err != nil {
		t.Fatal(err)
	}

	deposits, roots, _ := testutil.SetupInitialDeposits(t, validatorNum)
	for index, dd := range deposits {
		_, err = contract.Deposit(txOps, dd.Data.PublicKey, dd.Data.WithdrawalCredentials, dd.Data.Signature, roots[index])
		if err != nil {
			t.Error("unable to send transaction to contract")
		}
	}

	keystore, err := keystore.DecryptKey(jsonBytes, "" /*password*/)
	if err != nil {
		t.Fatal(err)
	}
	// Picked 20 for this as a "safe" number of blocks to mine so the deposits
	// are detected.
	if err := mineBlocks(web3, keystore, 20); err != nil {
		t.Fatal(err)
	}

	return valClients
}
