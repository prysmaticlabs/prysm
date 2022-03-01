package components

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/io/file"
	"github.com/prysmaticlabs/prysm/testing/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/testing/endtoend/params"
	e2etypes "github.com/prysmaticlabs/prysm/testing/endtoend/types"
)

var _ e2etypes.ComponentRunner = (*LighthouseBeaconNode)(nil)
var _ e2etypes.ComponentRunner = (*LighthouseBeaconNodeSet)(nil)
var _ e2etypes.BeaconNodeSet = (*LighthouseBeaconNodeSet)(nil)

// LighthouseBeaconNodeSet represents set of lighthouse beacon nodes.
type LighthouseBeaconNodeSet struct {
	e2etypes.ComponentRunner
	config  *e2etypes.E2EConfig
	enr     string
	started chan struct{}
}

// SetENR assigns ENR to the set of beacon nodes.
func (s *LighthouseBeaconNodeSet) SetENR(enr string) {
	s.enr = enr
}

// NewLighthouseBeaconNodes creates and returns a set of lighthouse beacon nodes.
func NewLighthouseBeaconNodes(config *e2etypes.E2EConfig) *LighthouseBeaconNodeSet {
	return &LighthouseBeaconNodeSet{
		config:  config,
		started: make(chan struct{}, 1),
	}
}

// Start starts all the beacon nodes in set.
func (s *LighthouseBeaconNodeSet) Start(ctx context.Context) error {
	if s.enr == "" {
		return errors.New("empty ENR")
	}

	// Create beacon nodes.
	nodes := make([]e2etypes.ComponentRunner, e2e.TestParams.LighthouseBeaconNodeCount)
	for i := 0; i < e2e.TestParams.LighthouseBeaconNodeCount; i++ {
		nodes[i] = NewLighthouseBeaconNode(s.config, i, s.enr)
	}

	// Wait for all nodes to finish their job (blocking).
	// Once nodes are ready passed in handler function will be called.
	return helpers.WaitOnNodes(ctx, nodes, func() {
		// All nodes stated, close channel, so that all services waiting on a set, can proceed.
		close(s.started)
	})
}

// Started checks whether beacon node set is started and all nodes are ready to be queried.
func (s *LighthouseBeaconNodeSet) Started() <-chan struct{} {
	return s.started
}

// LighthouseBeaconNode represents a lighthouse beacon node.
type LighthouseBeaconNode struct {
	e2etypes.ComponentRunner
	config  *e2etypes.E2EConfig
	started chan struct{}
	index   int
	enr     string
}

// NewBeaconNode creates and returns a beacon node.
func NewLighthouseBeaconNode(config *e2etypes.E2EConfig, index int, enr string) *LighthouseBeaconNode {
	return &LighthouseBeaconNode{
		config:  config,
		index:   index,
		enr:     enr,
		started: make(chan struct{}, 1),
	}
}

// Start starts a fresh beacon node, connecting to all passed in beacon nodes.
func (node *LighthouseBeaconNode) Start(ctx context.Context) error {
	binaryPath, found := bazel.FindBinary("external/lighthouse", "lighthouse")
	if !found {
		log.Info(binaryPath)
		log.Error("beacon chain binary not found")
	}

	_, index, _ := node.config, node.index, node.enr
	testDir, err := node.createTestnetDir(index)
	if err != nil {
		return err
	}

	args := []string{
		"beacon_node",
		fmt.Sprintf("--datadir=%s/lighthouse-beacon-node-%d", e2e.TestParams.TestPath, index),
		fmt.Sprintf("--testnet-dir=%s", testDir),
		"--staking",
		"--enr-address=127.0.0.1",
		fmt.Sprintf("--enr-udp-port=%d", e2e.TestParams.BeaconNodeRPCPort+index+e2e.LighthouseP2PPortOffset),
		fmt.Sprintf("--enr-tcp-port=%d", e2e.TestParams.BeaconNodeRPCPort+index+e2e.LighthouseP2PPortOffset),
		fmt.Sprintf("--port=%d", e2e.TestParams.BeaconNodeRPCPort+index+e2e.LighthouseP2PPortOffset),
		fmt.Sprintf("--http-port=%d", e2e.TestParams.BeaconNodeRPCPort+index+e2e.LighthouseHTTPPortOffset),
		fmt.Sprintf("--target-peers=%d", 10),
		fmt.Sprintf("--eth1-endpoints=http://127.0.0.1:%d", e2e.TestParams.Eth1RPCPort),
		fmt.Sprintf("--boot-nodes=%s", node.enr),
		fmt.Sprintf("--metrics-port=%d", e2e.TestParams.BeaconNodeMetricsPort+index+e2e.LighthouseMetricsPortOffset),
		"--metrics",
		"--http",
		"--debug-level=debug",
	}
	if node.config.UseFixedPeerIDs {
		flagVal := strings.Join(node.config.PeerIDs, ",")
		args = append(args,
			fmt.Sprintf("--trusted-peers=%s", flagVal))
	}
	cmd := exec.CommandContext(ctx, binaryPath, args...) /* #nosec G204 */
	// Write stdout and stderr to log files.
	stdout, err := os.Create(path.Join(e2e.TestParams.LogPath, fmt.Sprintf("lighthouse_beacon_node_%d_stdout.log", index)))
	if err != nil {
		return err
	}
	stderr, err := os.Create(path.Join(e2e.TestParams.LogPath, fmt.Sprintf("lighthouse_beacon_node_%d_stderr.log", index)))
	if err != nil {
		return err
	}
	defer func() {
		if err := stdout.Close(); err != nil {
			log.WithError(err).Error("Failed to close stdout file")
		}
		if err := stderr.Close(); err != nil {
			log.WithError(err).Error("Failed to close stderr file")
		}
	}()
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	log.Infof("Starting lighthouse beacon chain %d with flags: %s", index, strings.Join(args[2:], " "))
	if err = cmd.Start(); err != nil {
		return fmt.Errorf("failed to start beacon node: %w", err)
	}

	if err = helpers.WaitForTextInFile(stderr, "Configured for network"); err != nil {
		return fmt.Errorf("could not find initialization for node %d, this means the node had issues starting: %w", index, err)
	}

	// Mark node as ready.
	close(node.started)

	return cmd.Wait()
}

// Started checks whether beacon node is started and ready to be queried.
func (node *LighthouseBeaconNode) Started() <-chan struct{} {
	return node.started
}

func (node *LighthouseBeaconNode) createTestnetDir(index int) (string, error) {
	testNetDir := e2e.TestParams.TestPath + fmt.Sprintf("/lighthouse-testnet-%d", index)
	configPath := filepath.Join(testNetDir, "config.yaml")
	rawYaml := params.E2EMainnetConfigYaml()
	// Add in deposit contract in yaml
	depContractStr := fmt.Sprintf("\nDEPOSIT_CONTRACT_ADDRESS: %#x", e2e.TestParams.ContractAddress)
	rawYaml = append(rawYaml, []byte(depContractStr)...)

	if err := file.MkdirAll(testNetDir); err != nil {
		return "", err
	}
	if err := file.WriteFile(configPath, rawYaml); err != nil {
		return "", err
	}
	bootPath := filepath.Join(testNetDir, "boot_enr.yaml")
	enrYaml := []byte(fmt.Sprintf("[%s]", node.enr))
	if err := file.WriteFile(bootPath, enrYaml); err != nil {
		return "", err
	}
	deployPath := filepath.Join(testNetDir, "deploy_block.txt")
	deployYaml := []byte("0")
	return testNetDir, file.WriteFile(deployPath, deployYaml)
}
