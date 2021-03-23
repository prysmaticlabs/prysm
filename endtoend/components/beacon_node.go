// Package components defines utilities to spin up actual
// beacon node and validator processes as needed by end to end tests.
package components

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/prysmaticlabs/prysm/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/endtoend/params"
	e2etypes "github.com/prysmaticlabs/prysm/endtoend/types"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"golang.org/x/sync/errgroup"
)

var _ e2etypes.ComponentRunner = (*BeaconNode)(nil)
var _ e2etypes.ComponentRunner = (*BeaconNodes)(nil)
var _ e2etypes.ComponentRunner = (*BootNode)(nil)

// BeaconNodes represents set of beacon nodes.
type BeaconNodes struct {
	e2etypes.ComponentRunner
	config  *e2etypes.E2EConfig
	enr     string
	started chan struct{}
}

// SetENR assigns ENR to the set of beacon nodes.
func (s *BeaconNodes) SetENR(enr string) {
	s.enr = enr
}

// NewBeaconNodes creates and returns a set of beacon nodes.
func NewBeaconNodes(config *e2etypes.E2EConfig) *BeaconNodes {
	return &BeaconNodes{
		config:  config,
		started: make(chan struct{}, 1),
	}
}

// Start starts all the beacon nodes in set.
func (s *BeaconNodes) Start(ctx context.Context) error {
	if s.enr == "" {
		return errors.New("empty ENR")
	}

	// Create beacon nodes.
	nodes := make([]*BeaconNode, e2e.TestParams.BeaconNodeCount)
	for i := 0; i < e2e.TestParams.BeaconNodeCount; i++ {
		nodes[i] = NewBeaconNode(s.config, i, s.enr)
	}

	// Start all created beacon nodes.
	g, ctx := errgroup.WithContext(ctx)
	for _, node := range nodes {
		node := node
		g.Go(func() error {
			return node.Start(ctx)
		})
	}

	// Mark set as ready (happens when all contained nodes report as started).
	go func() {
		for _, node := range nodes {
			select {
			case <-ctx.Done():
				return
			case <-node.Started():
				continue
			}
		}
		// Only close if all nodes are started. When context is cancelled, node is not considered as started.
		// Stalled nodes do not pose problem -- after certain period of time main routine times out.
		close(s.started)
	}()

	return g.Wait()
}

// Started checks whether beacon node set is started and all nodes are ready to be queried.
func (s *BeaconNodes) Started() <-chan struct{} {
	return s.started
}

// BeaconNode represents beacon node.
type BeaconNode struct {
	e2etypes.ComponentRunner
	config  *e2etypes.E2EConfig
	started chan struct{}
	index   int
	enr     string
}

// NewBeaconNode creates and returns a beacon node.
func NewBeaconNode(config *e2etypes.E2EConfig, index int, enr string) *BeaconNode {
	return &BeaconNode{
		config:  config,
		index:   index,
		enr:     enr,
		started: make(chan struct{}, 1),
	}
}

// Start starts a fresh beacon node, connecting to all passed in beacon nodes.
func (node *BeaconNode) Start(ctx context.Context) error {
	binaryPath, found := bazel.FindBinary("cmd/beacon-chain", "beacon-chain")
	if !found {
		log.Info(binaryPath)
		return errors.New("beacon chain binary not found")
	}

	config, index, enr := node.config, node.index, node.enr
	stdOutFile, err := helpers.DeleteAndCreateFile(e2e.TestParams.LogPath, fmt.Sprintf(e2e.BeaconNodeLogFileName, index))
	if err != nil {
		return err
	}

	args := []string{
		fmt.Sprintf("--datadir=%s/eth2-beacon-node-%d", e2e.TestParams.TestPath, index),
		fmt.Sprintf("--log-file=%s", stdOutFile.Name()),
		fmt.Sprintf("--deposit-contract=%s", e2e.TestParams.ContractAddress.Hex()),
		fmt.Sprintf("--rpc-port=%d", e2e.TestParams.BeaconNodeRPCPort+index),
		fmt.Sprintf("--http-web3provider=http://127.0.0.1:%d", e2e.TestParams.Eth1RPCPort),
		fmt.Sprintf("--min-sync-peers=%d", e2e.TestParams.BeaconNodeCount-1),
		fmt.Sprintf("--p2p-udp-port=%d", e2e.TestParams.BeaconNodeRPCPort+index+10),
		fmt.Sprintf("--p2p-tcp-port=%d", e2e.TestParams.BeaconNodeRPCPort+index+20),
		fmt.Sprintf("--monitoring-port=%d", e2e.TestParams.BeaconNodeMetricsPort+index),
		fmt.Sprintf("--grpc-gateway-port=%d", e2e.TestParams.BeaconNodeRPCPort+index+40),
		fmt.Sprintf("--contract-deployment-block=%d", 0),
		fmt.Sprintf("--rpc-max-page-size=%d", params.BeaconConfig().MinGenesisActiveValidatorCount),
		fmt.Sprintf("--bootstrap-node=%s", enr),
		"--verbosity=debug",
		"--force-clear-db",
		"--e2e-config",
		"--accept-terms-of-use",
	}
	if config.UsePprof {
		args = append(args, "--pprof", fmt.Sprintf("--pprofport=%d", e2e.TestParams.BeaconNodeRPCPort+index+50))
	}
	args = append(args, featureconfig.E2EBeaconChainFlags...)
	args = append(args, config.BeaconFlags...)

	cmd := exec.CommandContext(ctx, binaryPath, args...)
	log.Infof("Starting beacon chain %d with flags: %s", index, strings.Join(args[2:], " "))
	if err = cmd.Start(); err != nil {
		return fmt.Errorf("failed to start beacon node: %w", err)
	}

	if err = helpers.WaitForTextInFile(stdOutFile, "gRPC server listening on port"); err != nil {
		return fmt.Errorf("could not find multiaddr for node %d, this means the node had issues starting: %w", index, err)
	}

	// Mark node as ready.
	close(node.started)

	return cmd.Wait()
}

// Started checks whether beacon node is started and ready to be queried.
func (node *BeaconNode) Started() <-chan struct{} {
	return node.started
}

// BootNode represents boot node.
type BootNode struct {
	e2etypes.ComponentRunner
	started chan struct{}
	enr     string
}

// NewBootNode creates and returns boot node.
func NewBootNode(config *e2etypes.E2EConfig) *BootNode {
	return &BootNode{
		started: make(chan struct{}, 1),
	}
}

// ENR exposes node's ENR.
func (node *BootNode) ENR() string {
	return node.enr
}

// StartBootnode starts a bootnode and returns its ENR.
func (node *BootNode) Start(ctx context.Context) error {
	binaryPath, found := bazel.FindBinary("tools/bootnode", "bootnode")
	if !found {
		log.Info(binaryPath)
		return errors.New("boot node binary not found")
	}

	stdOutFile, err := helpers.DeleteAndCreateFile(e2e.TestParams.LogPath, e2e.BootNodeLogFileName)
	if err != nil {
		return err
	}

	args := []string{
		fmt.Sprintf("--log-file=%s", stdOutFile.Name()),
		fmt.Sprintf("--discv5-port=%d", e2e.TestParams.BootNodePort),
		fmt.Sprintf("--metrics-port=%d", e2e.TestParams.BootNodePort+20),
		"--debug",
	}

	cmd := exec.CommandContext(ctx, binaryPath, args...)
	cmd.Stdout = stdOutFile
	cmd.Stderr = stdOutFile
	log.Infof("Starting boot node with flags: %s", strings.Join(args[1:], " "))
	if err = cmd.Start(); err != nil {
		return fmt.Errorf("failed to start beacon node: %w", err)
	}

	if err = helpers.WaitForTextInFile(stdOutFile, "Running bootnode"); err != nil {
		return fmt.Errorf("could not find enr for bootnode, this means the bootnode had issues starting: %w", err)
	}

	node.enr, err = enrFromLogFile(stdOutFile.Name())
	if err != nil {
		return fmt.Errorf("could not get enr for bootnode: %w", err)
	}

	// Mark node as ready.
	close(node.started)

	return cmd.Wait()
}

// Started checks whether boot node is started and ready to be queried.
func (node *BootNode) Started() <-chan struct{} {
	return node.started
}

func enrFromLogFile(name string) (string, error) {
	byteContent, err := ioutil.ReadFile(name)
	if err != nil {
		return "", err
	}
	contents := string(byteContent)

	searchText := "Running bootnode: "
	startIdx := strings.Index(contents, searchText)
	if startIdx == -1 {
		return "", fmt.Errorf("did not find ENR text in %s", contents)
	}
	startIdx += len(searchText)
	endIdx := strings.Index(contents[startIdx:], " prefix=bootnode")
	if endIdx == -1 {
		return "", fmt.Errorf("did not find ENR text in %s", contents)
	}
	return contents[startIdx : startIdx+endIdx-1], nil
}
