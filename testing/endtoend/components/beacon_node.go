// Package components defines utilities to spin up actual
// beacon node and validator processes as needed by end to end tests.
package components

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/pkg/errors"
	cmdshared "github.com/prysmaticlabs/prysm/cmd"
	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/config/features"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/testing/endtoend/e2ez"
	"github.com/prysmaticlabs/prysm/testing/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/testing/endtoend/params"
	e2etypes "github.com/prysmaticlabs/prysm/testing/endtoend/types"
)

var _ e2etypes.ComponentRunner = (*BeaconNode)(nil)
var _ e2etypes.ComponentRunner = (*BeaconNodeSet)(nil)

// BeaconNodeSet represents set of beacon nodes.
type BeaconNodeSet struct {
	e2etypes.ComponentRunner
	config  *e2etypes.E2EConfig
	enr     string
	ids     []string
	started chan struct{}
	nodes   []*BeaconNode
	flags   []string
	zp *e2ez.Server
}

// NewBeaconNodes creates and returns a set of beacon nodes.
func NewBeaconNodes(config *e2etypes.E2EConfig, enr string, flags []string, zp *e2ez.Server) *BeaconNodeSet {
	// Create beacon nodes.
	nodes := make([]*BeaconNode, e2e.TestParams.BeaconNodeCount)
	for i := 0; i < e2e.TestParams.BeaconNodeCount; i++ {
		nodes[i] = NewBeaconNode(i, enr, flags, config)
		zp.HandleZPages(nodes[i])
	}

	bns := &BeaconNodeSet{
		config:  config,
		started: make(chan struct{}, 1),
		nodes:   nodes,
		enr:     enr,
		flags:   flags,
		zp: zp,
	}
	zp.HandleZPages(bns)
	return bns
}

func (s *BeaconNodeSet) AddBeaconNode(index int, flags []string) *BeaconNode {
	bn := NewBeaconNode(index, s.enr, flags, s.config)
	s.nodes = append(s.nodes, bn)
	s.zp.HandleZPages(bn)
	return bn
}

// Start starts all the beacon nodes in set.
func (s *BeaconNodeSet) Start(ctx context.Context) error {
	if s.enr == "" {
		return errors.New("empty ENR")
	}

	nodes := make([]e2etypes.ComponentRunner, len(s.nodes))
	for i, n := range s.nodes {
		nodes[i] = n
	}

	// Wait for all nodes to finish their job (blocking).
	// Once nodes are ready passed in handler function will be called.
	return helpers.WaitOnNodes(ctx, nodes, func() {
		if s.config.UseFixedPeerIDs {
			for i := 0; i < len(nodes); i++ {
				s.ids = append(s.ids, nodes[i].(*BeaconNode).peerID)
			}
			s.config.PeerIDs = s.ids
		}
		// All nodes stated, close channel, so that all services waiting on a set, can proceed.
		close(s.started)
	})
}

func (s *BeaconNodeSet) ZPath() string {
	return "/beacon-nodes"
}

func (s *BeaconNodeSet) ZMarkdown() (string, error) {
	tmpl := `
%d beacon nodes
---------------

%s`
	nodeList := ""
	for _, node := range s.nodes {
		nodeList = nodeList + fmt.Sprintf("\n - [beacon node #%d](%s)", node.index, node.ZPath())
	}
	return fmt.Sprintf(tmpl, len(s.nodes), nodeList), nil
}

// Started checks whether beacon node set is started and all nodes are ready to be queried.
func (s *BeaconNodeSet) Started() <-chan struct{} {
	return s.started
}

// BeaconNode represents beacon node.
type BeaconNode struct {
	e2etypes.ComponentRunner
	config  *e2etypes.E2EConfig
	started chan struct{}
	index   int
	flags   []string
	enr     string
	peerID  string
}

func (node *BeaconNode) ZPath() string {
	return fmt.Sprintf("/beacon-node/%d", node.index)
}

var bnzm = template.Must(template.New("BeaconNode.ZMarkdown").Parse("" +
	"beacon node {{.Index}}\n" +
	"--------------\n\n" +
	"```\n" +
	"{{.StartCmd}}" +
	"```\n\n" +
	"http addr={{.HTTPAddr}}\n\n" +
	"grpc addr={{.GRPCAddr}}\n\n" +
	"db path={{.DBPath}}\n\n" +
	"log path={{.LogPath}}\n\n" +
	"stdout path={{.StdoutPath}}\n\n" +
	"stderr path={{.StderrPath}}\n\n"))

func (node *BeaconNode) ZMarkdown() (string, error) {
	bin, args, err := node.startCommand()
	if err != nil {
		return "", err
	}
	cmd := path.Join(bin, args[0])
	for _, a := range args {
		cmd += fmt.Sprintf("\n%s \\", a)
	}

	buf := bytes.NewBuffer(nil)
	err = bnzm.Execute(buf, struct {
		Index      int
		StartCmd   string
		DBPath     string
		LogPath    string
		StdoutPath string
		StderrPath string
		HTTPAddr   string
		GRPCAddr   string
	}{
		Index:      node.index,
		StartCmd:   cmd,
		DBPath:     node.dbPath(),
		LogPath:    node.logPath(),
		StdoutPath: node.stdoutPath(),
		StderrPath: node.stderrPath(),
		HTTPAddr:   node.httpAddr(),
		GRPCAddr:   node.grpcAddr(),
	})
	return buf.String(), err
}

var _ e2ez.ZPage = &BeaconNode{}

// NewBeaconNode creates and returns a beacon node.
func NewBeaconNode(index int, enr string, flags []string, config *e2etypes.E2EConfig) *BeaconNode {
	return &BeaconNode{
		config:  config,
		index:   index,
		enr:     enr,
		started: make(chan struct{}, 1),
		flags:   flags,
	}
}

func (node *BeaconNode) startCommand() (string, []string, error) {
	binaryPath, found := bazel.FindBinary("cmd/beacon-chain", "beacon-chain")
	if !found {
		log.Info(binaryPath)
		return "", []string{}, errors.New("beacon chain binary not found")
	}
	config, index, enr := node.config, node.index, node.enr
	expectedNumOfPeers := e2e.TestParams.BeaconNodeCount + e2e.TestParams.LighthouseBeaconNodeCount
	if node.config.TestSync {
		expectedNumOfPeers += 1
	}
	expectedNumOfPeers += 10
	jwtPath := path.Join(e2e.TestParams.TestPath, "eth1data/"+strconv.Itoa(node.index)+"/")
	if index == 0 {
		jwtPath = path.Join(e2e.TestParams.TestPath, "eth1data/miner/")
	}
	jwtPath = path.Join(jwtPath, "geth/jwtsecret")
	args := []string{
		fmt.Sprintf("--%s=%s", cmdshared.DataDirFlag.Name, node.dbPath()),
		fmt.Sprintf("--%s=%s", cmdshared.LogFileName.Name, node.logPath()),
		fmt.Sprintf("--%s=%s", flags.DepositContractFlag.Name, e2e.TestParams.ContractAddress.Hex()),
		fmt.Sprintf("--%s=%d", flags.RPCPort.Name, e2e.TestParams.Ports.PrysmBeaconNodeRPCPort+index),
		fmt.Sprintf("--%s=http://127.0.0.1:%d", flags.HTTPWeb3ProviderFlag.Name, e2e.TestParams.Ports.Eth1RPCPort+index),
		fmt.Sprintf("--%s=%s", flags.ExecutionJWTSecretFlag.Name, jwtPath),
		fmt.Sprintf("--%s=%d", flags.MinSyncPeers.Name, 1),
		fmt.Sprintf("--%s=%d", cmdshared.P2PUDPPort.Name, e2e.TestParams.Ports.PrysmBeaconNodeUDPPort+index),
		fmt.Sprintf("--%s=%d", cmdshared.P2PTCPPort.Name, e2e.TestParams.Ports.PrysmBeaconNodeTCPPort+index),
		fmt.Sprintf("--%s=%d", cmdshared.P2PMaxPeers.Name, expectedNumOfPeers),
		fmt.Sprintf("--%s=%d", flags.MonitoringPortFlag.Name, e2e.TestParams.Ports.PrysmBeaconNodeMetricsPort+index),
		fmt.Sprintf("--%s=%d", flags.GRPCGatewayPort.Name, e2e.TestParams.Ports.PrysmBeaconNodeGatewayPort+index),
		fmt.Sprintf("--%s=%d", flags.ContractDeploymentBlock.Name, 0),
		fmt.Sprintf("--%s=%d", flags.MinPeersPerSubnet.Name, 0),
		fmt.Sprintf("--%s=%d", cmdshared.RPCMaxPageSizeFlag.Name, params.BeaconConfig().MinGenesisActiveValidatorCount),
		fmt.Sprintf("--%s=%s", cmdshared.BootstrapNode.Name, enr),
		fmt.Sprintf("--%s=%s", cmdshared.VerbosityFlag.Name, "debug"),
		fmt.Sprintf("--%s=%s", cmdshared.ChainConfigFileFlag.Name, node.config.BeaconChainConfigPath()),
		"--slots-per-archive-point=1",
		"--" + cmdshared.ForceClearDB.Name,
		"--" + cmdshared.AcceptTosFlag.Name,
		"--" + flags.EnableDebugRPCEndpoints.Name,
	}
	if config.UsePprof {
		args = append(args, "--pprof", fmt.Sprintf("--pprofport=%d", e2e.TestParams.Ports.PrysmBeaconNodePprofPort+index))
	}
	// Only add in the feature flags if we either aren't performing a control test
	// on our features or the beacon index is a power of 2.
	if !config.TestFeature || index%2 == 0 {
		args = append(args, features.E2EBeaconChainFlags...)
	}
	args = append(args, config.BeaconFlags...)
	args = append(args, node.flags...)

	return binaryPath, args, nil
}

func (node *BeaconNode) dbPath() string {
	return fmt.Sprintf("%s/eth2-beacon-node-%d", e2e.TestParams.TestPath, node.index)
}

func (node *BeaconNode) logPath() string {
	return filepath.Clean(path.Join(e2e.TestParams.LogPath, fmt.Sprintf(e2e.BeaconNodeLogFileName, node.index)))
}

func (node *BeaconNode) stdoutPath() string {
	return path.Join(e2e.TestParams.LogPath, fmt.Sprintf("beacon_node_%d_stdout.log", node.index))
}

func (node *BeaconNode) stderrPath() string {
	return path.Join(e2e.TestParams.LogPath, fmt.Sprintf("beacon_node_%d_stderr.log", node.index))
}

func (node *BeaconNode) httpAddr() string {
	port := e2e.TestParams.Ports.PrysmBeaconNodeGatewayPort + node.index
	return fmt.Sprintf("http://localhost:%d", port)
}

func (node *BeaconNode) grpcAddr() string {
	port := e2e.TestParams.Ports.PrysmBeaconNodeRPCPort + node.index
	return fmt.Sprintf("localhost:%d", port)
}

// Start starts a fresh beacon node, connecting to all passed in beacon nodes.
func (node *BeaconNode) Start(ctx context.Context) error {
	stdOutFile, err := helpers.DeleteAndCreateFile(node.logPath(), "")
	if err != nil {
		return err
	}

	bin, args, err := node.startCommand()
	if err != nil {
		return errors.Wrap(err, "filed to generate start command")
	}
	cmd := exec.CommandContext(ctx, bin, args...) // #nosec G204 -- Safe
	// Write stdout and stderr to log files.
	stdout, err := os.Create(node.stdoutPath())
	if err != nil {
		return err
	}
	stderr, err := os.Create(node.stderrPath())
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
	log.Infof("Starting beacon chain %d with flags: %s", node.index, strings.Join(args[2:], " "))
	if err = cmd.Start(); err != nil {
		return fmt.Errorf("failed to start beacon node: %w", err)
	}

	if err = helpers.WaitForTextInFile(stdOutFile, "gRPC server listening on port"); err != nil {
		return fmt.Errorf("could not find multiaddr for node %d, this means the node had issues starting: %w", node.index, err)
	}

	if node.config.UseFixedPeerIDs {
		peerId, err := helpers.FindFollowingTextInFile(stdOutFile, "Running node with peer id of ")
		if err != nil {
			return fmt.Errorf("could not find peer id: %w", err)
		}
		node.peerID = peerId
	}

	// Mark node as ready.
	close(node.started)

	return cmd.Wait()
}

// Started checks whether beacon node is started and ready to be queried.
func (node *BeaconNode) Started() <-chan struct{} {
	return node.started
}

func (node *BeaconNode) Index() int {
	return node.index
}