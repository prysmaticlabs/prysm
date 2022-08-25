package eth1

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	contracts "github.com/prysmaticlabs/prysm/v3/contracts/deposit/mock"
	io "github.com/prysmaticlabs/prysm/v3/io/file"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/v3/testing/endtoend/params"
	e2etypes "github.com/prysmaticlabs/prysm/v3/testing/endtoend/types"
	log "github.com/sirupsen/logrus"
)

const (
	EthAddress = "0x878705ba3f8bc32fcf7f4caa1a35e72af65cf766"
)

// Miner represents an ETH1 node which mines blocks.
type Miner struct {
	e2etypes.ComponentRunner
	started      chan struct{}
	bootstrapEnr string
	enr          string
	keystorePath string
	cmd          *exec.Cmd
}

// NewMiner creates and returns an ETH1 node miner.
func NewMiner() *Miner {
	return &Miner{
		started: make(chan struct{}, 1),
	}
}

// KeystorePath returns the path of the keystore file.
func (m *Miner) KeystorePath() string {
	return m.keystorePath
}

// ENR returns the miner's enode.
func (m *Miner) ENR() string {
	return m.enr
}

// SetBootstrapENR sets the bootstrap record.
func (m *Miner) SetBootstrapENR(bootstrapEnr string) {
	m.bootstrapEnr = bootstrapEnr
}

// Start runs a mining ETH1 node.
// The miner is responsible for moving the ETH1 chain forward and for deploying the deposit contract.
func (m *Miner) Start(ctx context.Context) error {
	binaryPath, found := bazel.FindBinary("cmd/geth", "geth")
	if !found {
		return errors.New("go-ethereum binary not found")
	}

	eth1Path := path.Join(e2e.TestParams.TestPath, "eth1data/miner/")
	// Clear out potentially existing dir to prevent issues.
	if _, err := os.Stat(eth1Path); !os.IsNotExist(err) {
		if err = os.RemoveAll(eth1Path); err != nil {
			return err
		}
	}

	genesisSrcPath, err := bazel.Runfile(path.Join(staticFilesPath, "genesis.json"))
	if err != nil {
		return err
	}
	genesisDstPath := binaryPath[:strings.LastIndex(binaryPath, "/")]
	cpCmd := exec.CommandContext(ctx, "cp", genesisSrcPath, genesisDstPath) // #nosec G204 -- Safe
	if err = cpCmd.Start(); err != nil {
		return err
	}
	if err = cpCmd.Wait(); err != nil {
		return err
	}

	initCmd := exec.CommandContext(
		ctx,
		binaryPath,
		"init",
		fmt.Sprintf("--datadir=%s", eth1Path),
		genesisDstPath+"/genesis.json") // #nosec G204 -- Safe
	initFile, err := helpers.DeleteAndCreateFile(e2e.TestParams.LogPath, "eth1-init_miner.log")
	if err != nil {
		return err
	}
	initCmd.Stderr = initFile
	if err = initCmd.Start(); err != nil {
		return err
	}
	if err = initCmd.Wait(); err != nil {
		return err
	}

	args := []string{
		fmt.Sprintf("--datadir=%s", eth1Path),
		fmt.Sprintf("--http.port=%d", e2e.TestParams.Ports.Eth1RPCPort),
		fmt.Sprintf("--ws.port=%d", e2e.TestParams.Ports.Eth1WSPort),
		fmt.Sprintf("--authrpc.port=%d", e2e.TestParams.Ports.Eth1AuthRPCPort),
		fmt.Sprintf("--bootnodes=%s", m.bootstrapEnr),
		fmt.Sprintf("--port=%d", e2e.TestParams.Ports.Eth1Port),
		fmt.Sprintf("--networkid=%d", NetworkId),
		"--http",
		"--http.api=engine,net,eth",
		"--http.addr=127.0.0.1",
		"--http.corsdomain=\"*\"",
		"--http.vhosts=\"*\"",
		"--rpc.allow-unprotected-txs",
		"--ws",
		"--ws.api=net,eth,engine",
		"--ws.addr=127.0.0.1",
		"--ws.origins=\"*\"",
		"--ipcdisable",
		"--verbosity=4",
		"--mine",
		fmt.Sprintf("--unlock=%s", EthAddress),
		"--allow-insecure-unlock",
		"--syncmode=full",
		fmt.Sprintf("--txpool.locals=%s", EthAddress),
		fmt.Sprintf("--password=%s", eth1Path+"/keystore/"+minerPasswordFile),
	}

	keystorePath, err := bazel.Runfile(path.Join(staticFilesPath, minerFile))
	if err != nil {
		return err
	}
	jsonBytes, err := os.ReadFile(keystorePath) // #nosec G304 -- ReadFile is safe
	if err != nil {
		return err
	}
	err = io.WriteFile(eth1Path+"/keystore/"+minerFile, jsonBytes)
	if err != nil {
		return err
	}
	err = io.WriteFile(eth1Path+"/keystore/"+minerPasswordFile, []byte(KeystorePassword))
	if err != nil {
		return err
	}

	runCmd := exec.CommandContext(ctx, binaryPath, args...) // #nosec G204 -- Safe
	file, err := os.Create(path.Join(e2e.TestParams.LogPath, "eth1_miner.log"))
	if err != nil {
		return err
	}
	runCmd.Stderr = file
	log.Infof("Starting eth1 miner with flags: %s", strings.Join(args[2:], " "))

	if err = runCmd.Start(); err != nil {
		return fmt.Errorf("failed to start eth1 chain: %w", err)
	}

	if err = helpers.WaitForTextInFile(file, "Commit new sealing work"); err != nil {
		return fmt.Errorf("mining log not found, this means the eth1 chain had issues starting: %w", err)
	}
	if err = helpers.WaitForTextInFile(file, "Started P2P networking"); err != nil {
		return fmt.Errorf("P2P log not found, this means the eth1 chain had issues starting: %w", err)
	}

	enode, err := enodeFromLogFile(file.Name())
	if err != nil {
		return err
	}
	enode = "enode://" + enode + "@127.0.0.1:" + fmt.Sprintf("%d", e2e.TestParams.Ports.Eth1Port)
	m.enr = enode
	log.Infof("Communicated enode. Enode is %s", enode)

	// Connect to the started geth dev chain.
	client, err := rpc.DialHTTP(fmt.Sprintf("http://127.0.0.1:%d", e2e.TestParams.Ports.Eth1RPCPort))
	if err != nil {
		return fmt.Errorf("failed to connect to ipc: %w", err)
	}
	web3 := ethclient.NewClient(client)

	// Deploy the contract.
	store, err := keystore.DecryptKey(jsonBytes, KeystorePassword)
	if err != nil {
		return err
	}
	// Advancing the blocks eth1follow distance to prevent issues reading the chain.
	if err = WaitForBlocks(web3, store, params.BeaconConfig().Eth1FollowDistance); err != nil {
		return fmt.Errorf("unable to advance chain: %w", err)
	}
	txOpts, err := bind.NewTransactorWithChainID(bytes.NewReader(jsonBytes), KeystorePassword, big.NewInt(NetworkId))
	if err != nil {
		return err
	}
	nonce, err := web3.PendingNonceAt(ctx, store.Address)
	if err != nil {
		return err
	}
	txOpts.Nonce = big.NewInt(0).SetUint64(nonce)
	txOpts.Context = ctx
	contractAddr, tx, _, err := contracts.DeployDepositContract(txOpts, web3)
	if err != nil {
		return fmt.Errorf("failed to deploy deposit contract: %w", err)
	}
	e2e.TestParams.ContractAddress = contractAddr

	// Wait for contract to mine.
	for pending := true; pending; _, pending, err = web3.TransactionByHash(ctx, tx.Hash()) {
		if err != nil {
			return err
		}
		time.Sleep(timeGapPerTX)
	}

	// Advancing the blocks another eth1follow distance to prevent issues reading the chain.
	if err = WaitForBlocks(web3, store, params.BeaconConfig().Eth1FollowDistance); err != nil {
		return fmt.Errorf("unable to advance chain: %w", err)
	}

	// Save keystore path (used for saving and mining deposits).
	m.keystorePath = keystorePath

	// Mark node as ready.
	close(m.started)

	m.cmd = runCmd
	return runCmd.Wait()
}

// Started checks whether ETH1 node is started and ready to be queried.
func (m *Miner) Started() <-chan struct{} {
	return m.started
}

// Pause pauses the component and its underlying process.
func (m *Miner) Pause() error {
	return m.cmd.Process.Signal(syscall.SIGSTOP)
}

// Resume resumes the component and its underlying process.
func (m *Miner) Resume() error {
	return m.cmd.Process.Signal(syscall.SIGCONT)
}

// Stop kills the component and its underlying process.
func (m *Miner) Stop() error {
	return m.cmd.Process.Kill()
}

func enodeFromLogFile(name string) (string, error) {
	byteContent, err := os.ReadFile(name) // #nosec G304
	if err != nil {
		return "", err
	}
	contents := string(byteContent)

	searchText := "self=enode://"
	startIdx := strings.Index(contents, searchText)
	if startIdx == -1 {
		return "", fmt.Errorf("did not find ENR text in %s", contents)
	}
	startIdx += len(searchText)
	endIdx := strings.Index(contents[startIdx:], "@")
	if endIdx == -1 {
		return "", fmt.Errorf("did not find ENR text in %s", contents)
	}
	enode := contents[startIdx : startIdx+endIdx]
	return strings.TrimPrefix(enode, "-"), nil
}
