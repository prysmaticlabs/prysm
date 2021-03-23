package components

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/prysmaticlabs/prysm/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/endtoend/params"
	e2etypes "github.com/prysmaticlabs/prysm/endtoend/types"
	"golang.org/x/sync/errgroup"
)

var _ e2etypes.ComponentRunner = (*SlasherNode)(nil)
var _ e2etypes.ComponentRunner = (*SlasherNodes)(nil)

// SlasherNodes represents set of slasher nodes.
type SlasherNodes struct {
	e2etypes.ComponentRunner
	config  *e2etypes.E2EConfig
	started chan struct{}
}

// NewSlasherNodes creates and returns a set of slasher nodes.
func NewSlasherNodes(config *e2etypes.E2EConfig) *SlasherNodes {
	return &SlasherNodes{
		config:  config,
		started: make(chan struct{}, 1),
	}
}

// Start starts all the slasher nodes in set.
func (s *SlasherNodes) Start(ctx context.Context) error {
	// Create slasher nodes.
	nodes := make([]*SlasherNode, e2e.TestParams.BeaconNodeCount)
	for i := 0; i < e2e.TestParams.BeaconNodeCount; i++ {
		nodes[i] = NewSlasherNode(s.config, i)
	}

	// Start all created slasher nodes.
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
func (s *SlasherNodes) Started() <-chan struct{} {
	return s.started
}

// SlasherNode represents a slasher node.
type SlasherNode struct {
	e2etypes.ComponentRunner
	index   int
	started chan struct{}
}

// NewSlasherNode creates and returns slasher node.
func NewSlasherNode(config *e2etypes.E2EConfig, index int) *SlasherNode {
	return &SlasherNode{
		index:   index,
		started: make(chan struct{}, 1),
	}
}

// Start starts slasher clients for use within E2E, connected to all beacon nodes.
func (node *SlasherNode) Start(ctx context.Context) error {
	binaryPath, found := bazel.FindBinary("cmd/slasher", "slasher")
	if !found {
		log.Info(binaryPath)
		return errors.New("slasher binary not found")
	}

	stdOutFile, err := helpers.DeleteAndCreateFile(e2e.TestParams.LogPath, fmt.Sprintf(e2e.SlasherLogFileName, node.index))
	if err != nil {
		return err
	}

	args := []string{
		fmt.Sprintf("--datadir=%s/slasher-data-%d/", e2e.TestParams.TestPath, node.index),
		fmt.Sprintf("--log-file=%s", stdOutFile.Name()),
		fmt.Sprintf("--rpc-port=%d", e2e.TestParams.SlasherRPCPort+node.index),
		fmt.Sprintf("--monitoring-port=%d", e2e.TestParams.SlasherMetricsPort+node.index),
		fmt.Sprintf("--beacon-rpc-provider=localhost:%d", e2e.TestParams.BeaconNodeRPCPort+node.index),
		"--force-clear-db",
		"--e2e-config",
		"--accept-terms-of-use",
	}

	log.Infof("Starting slasher %d with flags: %s", node.index, strings.Join(args[2:], " "))
	cmd := exec.CommandContext(ctx, binaryPath, args...)
	if err = cmd.Start(); err != nil {
		return fmt.Errorf("failed to start slasher client: %w", err)
	}

	if err = helpers.WaitForTextInFile(stdOutFile, "Starting slasher client"); err != nil {
		return fmt.Errorf("could not find starting logs for slasher %d, this means it had issues starting: %w", node.index, err)
	}

	// Mark node as ready.
	close(node.started)

	return cmd.Wait()
}

// Started checks whether slasher node is started and ready to be queried.
func (node *SlasherNode) Started() <-chan struct{} {
	return node.started
}
