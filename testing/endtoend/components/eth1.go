package components

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit/mock"
	file2 "github.com/prysmaticlabs/prysm/io/file"
	"github.com/prysmaticlabs/prysm/testing/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/testing/endtoend/params"
	e2etypes "github.com/prysmaticlabs/prysm/testing/endtoend/types"
)

const timeGapPerTX = 100 * time.Millisecond
const networkId = 123456
const staticFilesPath = "/testing/endtoend/static-files/eth1"
const keystorePassword = "password"

var bootstrapNode = ""

var _ e2etypes.ComponentRunner = (*Eth1Node)(nil)
var _ e2etypes.ComponentRunner = (*Eth1NodeSet)(nil)

// Eth1NodeSet represents a set of Eth1 nodes.
type Eth1NodeSet struct {
	e2etypes.ComponentRunner
	started      chan struct{}
	keystorePath string
	enr          string
}

// NewEth1NodeSet creates and returns a set of Eth1 nodes.
func NewEth1NodeSet() *Eth1NodeSet {
	return &Eth1NodeSet{
		started: make(chan struct{}, 1),
	}
}

func (s *Eth1NodeSet) KeystorePath() string {
	return s.keystorePath
}

func (s *Eth1NodeSet) SetENR(enr string) {
	s.enr = enr
}

// Start starts all the beacon nodes in set.
func (s *Eth1NodeSet) Start(ctx context.Context) error {
	// Create Eth1 nodes. The number of nodes is the same as the number of beacon nodes
	// because we want each beacon node to connect to its own Eth1 node.
	nodes := make([]e2etypes.ComponentRunner, e2e.TestParams.BeaconNodeCount)
	for i := 0; i < e2e.TestParams.BeaconNodeCount; i++ {
		node := NewEth1Node(i, s.enr)
		nodes[i] = node
	}

	// Wait for all nodes to finish their job (blocking).
	// Once nodes are ready passed in handler function will be called.
	return helpers.WaitOnNodes(ctx, nodes, func() {
		s.keystorePath = nodes[0].(*Eth1Node).KeystorePath()
		// All nodes stated, close channel, so that all services waiting on a set, can proceed.
		close(s.started)
	})
}

// Started checks whether beacon node set is started and all nodes are ready to be queried.
func (s *Eth1NodeSet) Started() <-chan struct{} {
	return s.started
}

// Eth1Node represents ETH1 node.
type Eth1Node struct {
	e2etypes.ComponentRunner
	started      chan struct{}
	keystorePath string
	index        int
	enr          string
}

// NewEth1Node creates and returns ETH1 node.
func NewEth1Node(index int, enr string) *Eth1Node {
	return &Eth1Node{
		started: make(chan struct{}, 1),
		index:   index,
		enr:     enr,
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

	eth1Path := path.Join(e2e.TestParams.TestPath, "eth1data/"+strconv.Itoa(node.index)+"/")
	// Clear out ETH1 to prevent issues.
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

	if !common.FileExist(genesisDstPath + "/genesis.json") {
		cpCmd := exec.Command("cp", genesisSrcPath, genesisDstPath)
		if err = cpCmd.Start(); err != nil {
			return err
		}
		if err = cpCmd.Wait(); err != nil {
			return err
		}
	}

	initCmd := exec.CommandContext(
		ctx,
		binaryPath,
		"init",
		genesisDstPath+"/genesis.json",
		fmt.Sprintf("--datadir=%s", eth1Path)) // #nosec G204 -- Safe
	initFile, err := helpers.DeleteAndCreateFile(e2e.TestParams.LogPath, "eth1-init_"+strconv.Itoa(node.index)+".log")
	if err != nil {
		return err
	}
	//initCmd.Stdout = initFile
	initCmd.Stderr = initFile
	if err = initCmd.Start(); err != nil {
		return err
	}
	if err = initCmd.Wait(); err != nil {
		return err
	}
	// We want only one node to mine blocks.
	isMiner := node.index == 0
	if !isMiner {
		for bootstrapNode == "" {
			time.Sleep(2 * time.Second)
		}
	}
	bootNode := ""
	if !isMiner {
		bootNode = bootstrapNode
	} else {
		bootNode = node.enr
	}
	args := []string{
		fmt.Sprintf("--datadir=%s", eth1Path),
		fmt.Sprintf("--http.port=%d", e2e.TestParams.Eth1RPCPort+10*node.index),
		fmt.Sprintf("--ws.port=%d", e2e.TestParams.Eth1RPCPort+10*node.index+e2e.ETH1WSOffset),
		fmt.Sprintf("--bootnodes=%s", bootNode),
		fmt.Sprintf("--port=%d", 30303+node.index),
		fmt.Sprintf("--networkid=%d", networkId),
		"--http",
		"--http.addr=127.0.0.1",
		"--http.corsdomain=\"*\"",
		"--http.vhosts=\"*\"",
		"--rpc.allow-unprotected-txs",
		"--ws",
		"--ws.addr=127.0.0.1",
		"--ws.origins=\"*\"",
		"--ipcdisable",
		"--verbosity=4",
	}

	keystoreFileName := "UTC--2021-12-22T19-14-08.590377700Z--878705ba3f8bc32fcf7f4caa1a35e72af65cf766"
	keystorePass := "password.txt"
	keystorePath, err := bazel.Runfile(path.Join(staticFilesPath, keystoreFileName))
	if err != nil {
		return err
	}
	jsonBytes, err := ioutil.ReadFile(keystorePath) // #nosec G304 -- ReadFile is safe
	if err != nil {
		return err
	}
	err = file2.WriteFile(eth1Path+"/keystore/"+keystoreFileName, jsonBytes)
	if err != nil {
		return err
	}
	err = file2.WriteFile(eth1Path+"/keystore/"+keystorePass, []byte("password"))
	if err != nil {
		return err
	}

	if isMiner {
		args = append(args, "--mine", "--unlock=0x878705ba3f8bc32fcf7f4caa1a35e72af65cf766", "--allow-insecure-unlock", fmt.Sprintf("--password=%s", eth1Path+"/keystore/"+keystorePass))
	}

	runCmd := exec.CommandContext(ctx, binaryPath, args...) // #nosec G204 -- Safe
	file, err := helpers.DeleteAndCreateFile(e2e.TestParams.LogPath, "eth1_"+strconv.Itoa(node.index)+".log")
	if err != nil {
		return err
	}
	runCmd.Stdout = file
	runCmd.Stderr = file
	log.Infof("Starting eth1 node %d with flags: %s", node.index, strings.Join(args[2:], " "))

	if err = runCmd.Start(); err != nil {
		return fmt.Errorf("failed to start eth1 chain: %w", err)
	}

	if isMiner {
		if err = helpers.WaitForTextInFile(file, "Commit new mining work"); err != nil {
			return fmt.Errorf("mining log not found, this means the eth1 chain had issues starting: %w", err)
		}
		if err = helpers.WaitForTextInFile(file, "Started P2P networking"); err != nil {
			return fmt.Errorf("mining log not found, this means the eth1 chain had issues starting: %w", err)
		}
		enode, err := enodeFromLogFile(file.Name())
		if err != nil {
			return err
		}
		enode = "enode://" + enode + "@127.0.0.1:" + fmt.Sprintf("%d", 30303+node.index)
		bootstrapNode = enode
		log.Infof("enode is %s", enode)
	}

	// Connect to the started geth dev chain.
	client, err := rpc.DialHTTP(fmt.Sprintf("http://127.0.0.1:%d", e2e.TestParams.Eth1RPCPort+10*node.index))
	if err != nil {
		return fmt.Errorf("failed to connect to ipc: %w", err)
	}
	web3 := ethclient.NewClient(client)

	if isMiner {
		keystoreFileName := "UTC--2021-12-22T19-14-08.590377700Z--878705ba3f8bc32fcf7f4caa1a35e72af65cf766"
		keystorePath, err := bazel.Runfile(path.Join(staticFilesPath, keystoreFileName))
		if err != nil {
			return err
		}

		// Access the dev account keystore to deploy the contract.
		jsonBytes, err := ioutil.ReadFile(keystorePath) // #nosec G304 -- ReadFile is safe
		if err != nil {
			return err
		}
		store, err := keystore.DecryptKey(jsonBytes, keystorePassword)
		if err != nil {
			return err
		}

		txOpts, err := bind.NewTransactorWithChainID(bytes.NewReader(jsonBytes), keystorePassword, big.NewInt(networkId))
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

		// Save keystore path (used for saving and mining deposits).
		node.keystorePath = keystorePath
	}

	// Mark node as ready.
	close(node.started)

	return runCmd.Wait()
}

// Started checks whether ETH1 node is started and ready to be queried.
func (node *Eth1Node) Started() <-chan struct{} {
	return node.started
}

func enodeFromLogFile(name string) (string, error) {
	byteContent, err := ioutil.ReadFile(name) // #nosec G304
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
	if strings.HasPrefix(enode, "-") {
		enode = strings.TrimPrefix(enode, "-")
	}
	return enode, nil
}
