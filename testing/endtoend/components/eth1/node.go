package eth1

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/testing/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/testing/endtoend/params"
	e2etypes "github.com/prysmaticlabs/prysm/testing/endtoend/types"
	log "github.com/sirupsen/logrus"
)

// Node represents an ETH1 node.
type Node struct {
	e2etypes.ComponentRunner
	started chan struct{}
	index   int
	enr     string
}

// NewNode creates and returns ETH1 node.
func NewNode(index int, enr string) *Node {
	return &Node{
		started: make(chan struct{}, 1),
		index:   index,
		enr:     enr,
	}
}

// Start runs a non-mining ETH1 node.
// To connect to a miner and start working properly, this node should be a part of a NodeSet.
func (node *Node) Start(ctx context.Context) error {
	binaryPath, found := bazel.FindBinary("cmd/geth", "geth")
	if !found {
		return errors.New("go-ethereum binary not found")
	}

	eth1Path := path.Join(e2e.TestParams.TestPath, "eth1data/"+strconv.Itoa(node.index)+"/")
	// Clear out potentially existing dir to prevent issues.
	if _, err := os.Stat(eth1Path); !os.IsNotExist(err) {
		if err = os.RemoveAll(eth1Path); err != nil {
			return err
		}
	}

	initCmd := exec.CommandContext(
		ctx,
		binaryPath,
		"init",
		binaryPath[:strings.LastIndex(binaryPath, "/")]+"/genesis.json",
		fmt.Sprintf("--datadir=%s", eth1Path)) // #nosec G204 -- Safe
	initFile, err := helpers.DeleteAndCreateFile(e2e.TestParams.LogPath, "eth1-init_"+strconv.Itoa(node.index)+".log")
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
		fmt.Sprintf("--http.port=%d", e2e.TestParams.Ports.Eth1RPCPort+node.index),
		fmt.Sprintf("--ws.port=%d", e2e.TestParams.Ports.Eth1WSPort+node.index),
		fmt.Sprintf("--bootnodes=%s", node.enr),
		fmt.Sprintf("--port=%d", e2e.TestParams.Ports.Eth1Port+node.index),
		fmt.Sprintf("--networkid=%d", NetworkId),
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
	if err = helpers.WaitForTextInFile(file, "Started P2P networking"); err != nil {
		return fmt.Errorf("P2P log not found, this means the eth1 chain had issues starting: %w", err)
	}

	// Mark node as ready.
	close(node.started)

	return runCmd.Wait()
}

// Started checks whether ETH1 node is started and ready to be queried.
func (node *Node) Started() <-chan struct{} {
	return node.started
}
