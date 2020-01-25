package endtoend

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// startEth1 starts an eth1 local dev chain and deploys a deposit contract.
func startEth1(t *testing.T, tmpPath string) (common.Address, string, int) {
	binaryPath, found := bazel.FindBinary("cmd/geth", "geth")
	if !found {
		t.Fatal("go-ethereum binary not found")
	}

	eth1Path := path.Join(tmpPath, "eth1data/")
	// Clear out ETH1 to prevent issues.
	if _, err := os.Stat(eth1Path); !os.IsNotExist(err) {
		if err := os.RemoveAll(eth1Path); err != nil {
			t.Fatal(err)
		}
	}

	args := []string{
		fmt.Sprintf("--datadir=%s", eth1Path),
		"--rpc",
		"--rpcaddr=0.0.0.0",
		"--rpccorsdomain=\"*\"",
		"--rpcvhosts=\"*\"",
		"--rpcport=8745",
		"--ws",
		"--wsaddr=0.0.0.0",
		"--wsorigins=\"*\"",
		"--wsport=8746",
		"--dev",
		"--dev.period=0",
		"--ipcdisable",
	}
	cmd := exec.Command(binaryPath, args...)
	file, err := os.Create(path.Join(tmpPath, "eth1.log"))
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stdout = file
	cmd.Stderr = file
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start eth1 chain: %v", err)
	}

	if err = waitForTextInFile(file, "Commit new mining work"); err != nil {
		t.Fatalf("mining log not found, this means the eth1 chain had issues starting: %v", err)
	}

	// Connect to the started geth dev chain.
	client, err := rpc.DialHTTP("http://127.0.0.1:8745")
	if err != nil {
		t.Fatalf("Failed to connect to ipc: %v", err)
	}
	web3 := ethclient.NewClient(client)

	// Access the dev account keystore to deploy the contract.
	fileName, err := exec.Command("ls", path.Join(tmpPath, "eth1data/keystore")).Output()
	if err != nil {
		t.Fatal(err)
	}
	keystorePath := path.Join(tmpPath, fmt.Sprintf("eth1data/keystore/%s", strings.TrimSpace(string(fileName))))
	jsonBytes, err := ioutil.ReadFile(keystorePath)
	if err != nil {
		t.Fatal(err)
	}
	keystore, err := keystore.DecryptKey(jsonBytes, "" /*password*/)
	if err != nil {
		t.Fatal(err)
	}

	// Advancing the blocks eth1follow distance to prevent issues reading the chain.
	if err := mineBlocks(web3, keystore, params.BeaconConfig().Eth1FollowDistance); err != nil {
		t.Fatalf("Unable to advance chain: %v", err)
	}

	txOpts, err := bind.NewTransactor(bytes.NewReader(jsonBytes), "" /*password*/)
	if err != nil {
		t.Fatal(err)
	}
	nonce, err := web3.PendingNonceAt(context.Background(), keystore.Address)
	if err != nil {
		t.Fatal(err)
	}
	txOpts.Nonce = big.NewInt(int64(nonce))
	contractAddr, tx, _, err := contracts.DeployDepositContract(txOpts, web3, txOpts.From)
	if err != nil {
		t.Fatalf("Failed to deploy deposit contract: %v", err)
	}

	// Wait for contract to mine.
	for pending := true; pending; _, pending, err = web3.TransactionByHash(context.Background(), tx.Hash()) {
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	return contractAddr, keystorePath, cmd.Process.Pid
}
