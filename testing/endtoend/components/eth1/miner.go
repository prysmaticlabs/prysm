package eth1

import (
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
	cmd          *exec.Cmd
}

// NewMiner creates and returns an ETH1 node miner.
func NewMiner() *Miner {
	return &Miner{
		started: make(chan struct{}, 1),
	}
}

// ENR returns the miner's enode.
func (m *Miner) ENR() string {
	return m.enr
}

// SetBootstrapENR sets the bootstrap record.
func (m *Miner) SetBootstrapENR(bootstrapEnr string) {
	m.bootstrapEnr = bootstrapEnr
}

func (m *Miner) DataDir(sub ...string) string {
	parts := append([]string{e2e.TestParams.TestPath, "eth1data/miner"}, sub...)
	return path.Join(parts...)
}

func (m *Miner) Password() string {
	return KeystorePassword
}

func (m *Miner) initDataDir() error {
	eth1Path := m.DataDir()
	// Clear out potentially existing dir to prevent issues.
	if _, err := os.Stat(eth1Path); !os.IsNotExist(err) {
		if err = os.RemoveAll(eth1Path); err != nil {
			return err
		}
	}
	return nil
}

// Start runs a mining ETH1 node.
// The miner is responsible for moving the ETH1 chain forward and for deploying the deposit contract.
func (m *Miner) Start(ctx context.Context) error {
	if err := m.initDataDir(); err != nil {
		return err
	}

	// find geth so we can run it.
	binaryPath, found := bazel.FindBinary("cmd/geth", "geth")
	if !found {
		return errors.New("go-ethereum binary not found")
	}

	staticGenesis, err := e2e.TestParams.Paths.Eth1Runfile("genesis.json")
	if err != nil {
		return err
	}
	genesisPath := path.Join(path.Dir(binaryPath), "genesis.json")
	if err := io.CopyFile(staticGenesis, genesisPath); err != nil {
		return errors.Wrapf(err, "error copying %s to %s", staticGenesis, genesisPath)
	}

	initCmd := exec.CommandContext(
		ctx,
		binaryPath,
		"init",
		fmt.Sprintf("--datadir=%s", m.DataDir()),
		genesisPath) // #nosec G204 -- Safe

	// redirect stderr to a log file
	initFile, err := helpers.DeleteAndCreatePath(e2e.TestParams.Logfile("eth1-init_miner.log"))
	if err != nil {
		return err
	}
	initCmd.Stderr = initFile

	// run init command and wait until it exits. this will initialize the geth node (required before starting).
	if err = initCmd.Start(); err != nil {
		return err
	}
	if err = initCmd.Wait(); err != nil {
		return err
	}

	pwFile := m.DataDir("keystore", minerPasswordFile)
	args := []string{
		fmt.Sprintf("--datadir=%s", m.DataDir()),
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
		fmt.Sprintf("--password=%s", pwFile),
	}

	keystorePath, err := e2e.TestParams.Paths.MinerKeyPath()
	if err != nil {
		return err
	}
	if err = io.CopyFile(keystorePath, m.DataDir("keystore", minerFile)); err != nil {
		return errors.Wrapf(err, "error copying %s to %s", keystorePath, m.DataDir("keystore", minerFile))
	}
	err = io.WriteFile(pwFile, []byte(KeystorePassword))
	if err != nil {
		return err
	}

	// give the miner start a couple of tries, since the p2p networking check is flaky
	var minerLog *os.File
	var retryErr error
	for retries := 0; retries < 3; retries++ {
		runCmd := exec.CommandContext(ctx, binaryPath, args...) // #nosec G204 -- Safe
		// redirect miner stderr to a log file
		minerLog, err = helpers.DeleteAndCreatePath(e2e.TestParams.Logfile("eth1_miner.log"))
		if err != nil {
			return err
		}
		runCmd.Stderr = minerLog
		retryErr = nil
		log.Infof("Starting eth1 miner, attempt %d, with flags: %s", retries, strings.Join(args[2:], " "))
		if err = runCmd.Start(); err != nil {
			return fmt.Errorf("failed to start eth1 chain: %w", err)
		}
		// check logs for common issues that prevent the EL miner from starting up.
		if err = helpers.WaitForTextInFile(minerLog, "Commit new sealing work"); err != nil {
			return fmt.Errorf("mining log not found, this means the eth1 chain had issues starting: %w", err)
		}
		if err = helpers.WaitForTextInFile(minerLog, "Started P2P networking"); err != nil {
			retryErr = fmt.Errorf("P2P log not found, this means the eth1 chain had issues starting: %w", err)
			continue
		}
		m.cmd = runCmd
		log.Infof("miner started after %d retries", retries)
		break
	}
	if retryErr != nil {
		return retryErr
	}

	enode, err := enodeFromLogFile(minerLog.Name())
	if err != nil {
		return err
	}
	enode = "enode://" + enode + "@127.0.0.1:" + fmt.Sprintf("%d", e2e.TestParams.Ports.Eth1Port)
	m.enr = enode
	log.Infof("Communicated enode. Enode is %s", enode)

	// Connect to the started geth dev chain.
	client, err := rpc.DialHTTP(e2e.TestParams.Eth1RPCURL(e2e.MinerComponentOffset).String())
	if err != nil {
		return fmt.Errorf("failed to connect to ipc: %w", err)
	}
	web3 := ethclient.NewClient(client)
	// this is the key for the miner account. miner account balance is pre-mined in genesis.json.
	key, err := helpers.KeyFromPath(keystorePath, KeystorePassword)
	if err != nil {
		return err
	}
	// Waiting for the blocks to advance by eth1follow to prevent issues reading the chain.
	// Note that WaitForBlocks spams transfer transactions (to and from the miner's address) in order to advance.
	if err = WaitForBlocks(web3, key, params.BeaconConfig().Eth1FollowDistance); err != nil {
		return fmt.Errorf("unable to advance chain: %w", err)
	}

	// Time to deploy the contract using the miner's key.
	txOpts, err := bind.NewKeyedTransactorWithChainID(key.PrivateKey, big.NewInt(NetworkId))
	if err != nil {
		return err
	}
	nonce, err := web3.PendingNonceAt(ctx, key.Address)
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
	if err = WaitForBlocks(web3, key, params.BeaconConfig().Eth1FollowDistance); err != nil {
		return fmt.Errorf("unable to advance chain: %w", err)
	}

	// Mark node as ready.
	close(m.started)

	return m.cmd.Wait()
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
