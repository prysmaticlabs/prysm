package components

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prysmaticlabs/prysm/config/params"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit"
	"github.com/prysmaticlabs/prysm/testing/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/testing/endtoend/params"
	e2etypes "github.com/prysmaticlabs/prysm/testing/endtoend/types"
)

const timeGapPerTX = 100 * time.Millisecond
const timeGapPerMiningTX = 250 * time.Millisecond

var _ e2etypes.ComponentRunner = (*Eth1Node)(nil)

// Eth1Node represents ETH1 node.
type Eth1Node struct {
	e2etypes.ComponentRunner
	started      chan struct{}
	keystorePath string
}

// NewEth1Node creates and returns ETH1 node.
func NewEth1Node() *Eth1Node {
	return &Eth1Node{
		started: make(chan struct{}, 1),
	}
}

// KeystorePath exposes node's keystore path.
func (node *Eth1Node) KeystorePath() string {
	return node.keystorePath
}

// Start starts an ETH1 local dev chain and deploys a deposit contract.
func (node *Eth1Node) Start(ctx context.Context) error {
	binaryPath, found := bazel.FindBinary("cmd/geth", "geth")
	if !found {
		return errors.New("go-ethereum binary not found")
	}

	eth1Path := path.Join(e2e.TestParams.TestPath, "eth1data/")
	// Clear out ETH1 to prevent issues.
	if _, err := os.Stat(eth1Path); !os.IsNotExist(err) {
		if err = os.RemoveAll(eth1Path); err != nil {
			return err
		}
	}

	args := []string{
		fmt.Sprintf("--datadir=%s", eth1Path),
		fmt.Sprintf("--http.port=%d", e2e.TestParams.Eth1RPCPort),
		fmt.Sprintf("--ws.port=%d", e2e.TestParams.Eth1RPCPort+1),
		"--http",
		"--http.addr=127.0.0.1",
		"--http.corsdomain=\"*\"",
		"--http.vhosts=\"*\"",
		"--rpc.allow-unprotected-txs",
		"--ws",
		"--ws.addr=127.0.0.1",
		"--ws.origins=\"*\"",
		"--dev",
		"--dev.period=2",
		"--ipcdisable",
	}
	cmd := exec.CommandContext(ctx, binaryPath, args...) // #nosec G204 -- Safe
	file, err := helpers.DeleteAndCreateFile(e2e.TestParams.LogPath, "eth1.log")
	if err != nil {
		return err
	}
	cmd.Stdout = file
	cmd.Stderr = file
	if err = cmd.Start(); err != nil {
		return fmt.Errorf("failed to start eth1 chain: %w", err)
	}

	if err = helpers.WaitForTextInFile(file, "Commit new mining work"); err != nil {
		return fmt.Errorf("mining log not found, this means the eth1 chain had issues starting: %w", err)
	}

	// Connect to the started geth dev chain.
	client, err := rpc.DialHTTP(fmt.Sprintf("http://127.0.0.1:%d", e2e.TestParams.Eth1RPCPort))
	if err != nil {
		return fmt.Errorf("failed to connect to ipc: %w", err)
	}
	web3 := ethclient.NewClient(client)

	// Access the dev account keystore to deploy the contract.
	fileName, err := exec.Command("ls", path.Join(eth1Path, "keystore")).Output() // #nosec G204
	if err != nil {
		return err
	}
	keystorePath := path.Join(eth1Path, fmt.Sprintf("keystore/%s", strings.TrimSpace(string(fileName))))
	jsonBytes, err := ioutil.ReadFile(keystorePath) // #nosec G304 -- ReadFile is safe
	if err != nil {
		return err
	}
	store, err := keystore.DecryptKey(jsonBytes, "" /*password*/)
	if err != nil {
		return err
	}

	// Advancing the blocks eth1follow distance to prevent issues reading the chain.
	if err = mineBlocks(web3, store, params.BeaconConfig().Eth1FollowDistance); err != nil {
		return fmt.Errorf("unable to advance chain: %w", err)
	}

	txOpts, err := bind.NewTransactorWithChainID(bytes.NewReader(jsonBytes), "" /*password*/, big.NewInt(1337))
	if err != nil {
		return err
	}
	nonce, err := web3.PendingNonceAt(context.Background(), store.Address)
	if err != nil {
		return err
	}
	txOpts.Nonce = big.NewInt(int64(nonce))
	txOpts.Context = context.Background()
	contractAddr, tx, _, err := contracts.DeployDepositContract(txOpts, web3)
	if err != nil {
		return fmt.Errorf("failed to deploy deposit contract: %w", err)
	}
	e2e.TestParams.ContractAddress = contractAddr

	// Wait for contract to mine.
	for pending := true; pending; _, pending, err = web3.TransactionByHash(context.Background(), tx.Hash()) {
		if err != nil {
			return err
		}
		time.Sleep(timeGapPerTX)
	}

	// Advancing the blocks another eth1follow distance to prevent issues reading the chain.
	if err = mineBlocks(web3, store, params.BeaconConfig().Eth1FollowDistance); err != nil {
		return fmt.Errorf("unable to advance chain: %w", err)
	}

	// Save keystore path (used for saving and mining deposits).
	node.keystorePath = keystorePath

	// Mark node as ready.
	close(node.started)

	return cmd.Wait()
}

// Started checks whether ETH1 node is started and ready to be queried.
func (node *Eth1Node) Started() <-chan struct{} {
	return node.started
}

func mineBlocks(web3 *ethclient.Client, keystore *keystore.Key, blocksToMake uint64) error {
	nonce, err := web3.PendingNonceAt(context.Background(), keystore.Address)
	if err != nil {
		return err
	}
	chainID, err := web3.NetworkID(context.Background())
	if err != nil {
		return err
	}
	block, err := web3.BlockByNumber(context.Background(), nil)
	if err != nil {
		return err
	}
	finishBlock := block.NumberU64() + blocksToMake

	for block.NumberU64() <= finishBlock {
		spamTX := types.NewTransaction(nonce, keystore.Address, big.NewInt(0), 21000, big.NewInt(1e6), []byte{})
		signed, err := types.SignTx(spamTX, types.NewEIP155Signer(chainID), keystore.PrivateKey)
		if err != nil {
			return err
		}
		if err = web3.SendTransaction(context.Background(), signed); err != nil {
			return err
		}
		nonce++
		time.Sleep(timeGapPerMiningTX)
		block, err = web3.BlockByNumber(context.Background(), nil)
		if err != nil {
			return err
		}
	}
	return nil
}
