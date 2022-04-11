package components

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/prysmaticlabs/prysm/testing/endtoend/e2ez"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/io/file"
	"github.com/prysmaticlabs/prysm/testing/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/testing/endtoend/params"
	e2etypes "github.com/prysmaticlabs/prysm/testing/endtoend/types"
)

var _ e2etypes.ComponentRunner = (*LighthouseBeaconNode)(nil)
var _ e2etypes.ComponentRunner = (*LighthouseBeaconNodeSet)(nil)

// LighthouseBeaconNodeSet represents set of lighthouse beacon nodes.
type LighthouseBeaconNodeSet struct {
	e2etypes.ComponentRunner
	config  *e2etypes.E2EConfig
	enr     string
	started chan struct{}
	nodes []*LighthouseBeaconNode
}

// NewLighthouseBeaconNodes creates and returns a set of lighthouse beacon nodes.
func NewLighthouseBeaconNodes(config *e2etypes.E2EConfig, enr string) *LighthouseBeaconNodeSet {
	nodes := make([]*LighthouseBeaconNode, e2e.TestParams.LighthouseBeaconNodeCount)
	for i := 0; i < e2e.TestParams.LighthouseBeaconNodeCount; i++ {
		nodes[i] = NewLighthouseBeaconNode(config, i, enr)
	}
	return &LighthouseBeaconNodeSet{
		config:  config,
		started: make(chan struct{}, 1),
		enr: enr,
		nodes: nodes,
	}
}

// Start starts all the beacon nodes in set.
func (s *LighthouseBeaconNodeSet) Start(ctx context.Context) error {
	if s.enr == "" {
		return errors.New("empty ENR")
	}

	// Create beacon nodes.
	nodes := make([]e2etypes.ComponentRunner, len(s.nodes))
	for i := 0; i < e2e.TestParams.LighthouseBeaconNodeCount; i++ {
		nodes[i] = s.nodes[i]
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

func (s *LighthouseBeaconNodeSet) ZPath() string {
	return "/lh-beacon-nodes"
}

func (s *LighthouseBeaconNodeSet) ZMarkdown() (string, error) {
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

func (s *LighthouseBeaconNodeSet) ZChildren() []e2ez.ZPage {
	zps := make([]e2ez.ZPage, len(s.nodes))
	for i := 0; i < len(s.nodes); i++ {
		zps[i] = s.nodes[i]
	}
	return zps
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

func (node *LighthouseBeaconNode) dbPath() string {
	return fmt.Sprintf("%s/lighthouse-beacon-node-%d", e2e.TestParams.TestPath, node.index)
}

func (node *LighthouseBeaconNode) stdoutPath() string {
	return path.Join(e2e.TestParams.LogPath, fmt.Sprintf("lighthouse_beacon_node_%d_stdout.log", node.index))
}

func (node *LighthouseBeaconNode) stderrPath() string {
	return path.Join(e2e.TestParams.LogPath, fmt.Sprintf("lighthouse_beacon_node_%d_stderr.log", node.index))
}

func (node *LighthouseBeaconNode) httpPort() int {
	return e2e.TestParams.Ports.LighthouseBeaconNodeHTTPPort+node.index
}

func (node *LighthouseBeaconNode) startCommand() (string, []string, error) {
	binaryPath, found := bazel.FindBinary("external/lighthouse", "lighthouse")
	if !found {
		log.Info(binaryPath)
		log.Error("beacon chain binary not found")
	}
	testDir, err := node.createTestnetDir(node.index)
	if err != nil {
		return "", []string{}, err
	}
	prysmNodeCount := e2e.TestParams.BeaconNodeCount
	jwtPath := path.Join(e2e.TestParams.TestPath, "eth1data/"+strconv.Itoa(node.index+prysmNodeCount)+"/")
	jwtPath = path.Join(jwtPath, "geth/jwtsecret")
	args := []string{
		"beacon_node",
		fmt.Sprintf("--datadir=%s", node.dbPath()),
		fmt.Sprintf("--testnet-dir=%s", testDir),
		"--staking",
		"--enr-address=127.0.0.1",
		fmt.Sprintf("--enr-udp-port=%d", e2e.TestParams.Ports.LighthouseBeaconNodeP2PPort+node.index),
		fmt.Sprintf("--enr-tcp-port=%d", e2e.TestParams.Ports.LighthouseBeaconNodeP2PPort+node.index),
		fmt.Sprintf("--port=%d", e2e.TestParams.Ports.LighthouseBeaconNodeP2PPort+node.index),
		fmt.Sprintf("--http-port=%d", node.httpPort()),
		fmt.Sprintf("--target-peers=%d", 10),
		fmt.Sprintf("--eth1-endpoints=http://127.0.0.1:%d", e2e.TestParams.Ports.Eth1RPCPort+prysmNodeCount+node.index),
		fmt.Sprintf("--execution-endpoints=http://127.0.0.1:%d", e2e.TestParams.Ports.Eth1AuthRPCPort+prysmNodeCount+node.index),
		fmt.Sprintf("--jwt-secrets=%s", jwtPath),
		fmt.Sprintf("--boot-nodes=%s", node.enr),
		fmt.Sprintf("--metrics-port=%d", e2e.TestParams.Ports.LighthouseBeaconNodeMetricsPort+node.index),
		"--metrics",
		"--http",
		"--http-allow-sync-stalled",
		"--enable-private-discovery",
		"--debug-level=debug",
		"--merge",
	}
	if node.config.UseFixedPeerIDs {
		flagVal := strings.Join(node.config.PeerIDs, ",")
		args = append(args,
			fmt.Sprintf("--trusted-peers=%s", flagVal))
	}
	return binaryPath, args, nil
}

// Start starts a fresh beacon node, connecting to all passed in beacon nodes.
func (node *LighthouseBeaconNode) Start(ctx context.Context) error {
	binaryPath, args, err := node.startCommand()
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, binaryPath, args...) /* #nosec G204 */
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
	log.Infof("Starting lighthouse beacon chain %d with flags: %s", node.index, strings.Join(args[2:], " "))
	if err = cmd.Start(); err != nil {
		return fmt.Errorf("failed to start beacon node: %w", err)
	}

	if err = helpers.WaitForTextInFile(stderr, "Configured for network"); err != nil {
		return fmt.Errorf("could not find initialization for node %d, this means the node had issues starting: %w", node.index, err)
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

func (node *LighthouseBeaconNode) ZPath() string {
	return fmt.Sprintf("/lh-beacon-node/%d", node.index)
}

var lbnzm = template.Must(template.New("BeaconNode.ZMarkdown").Parse("" +
	"beacon node {{.Index}}\n" +
	"--------------\n\n" +
	"http addr={{.HTTPAddr}}\n\n" +
	"db path={{.DBPath}}\n\n" +
	"stdout path={{.StdoutPath}}\n\n" +
	"stderr path={{.StderrPath}}\n\n" +
	"```\n" +
	"{{.StartCmd}}" +
	"```\n\n"))

func (node *LighthouseBeaconNode) ZMarkdown() (string, error) {
	bin, args, err := node.startCommand()
	if err != nil {
		return "", err
	}
	cmd := path.Join(bin, args[0])
	for _, a := range args {
		cmd += fmt.Sprintf("\n%s \\", a)
	}

	buf := bytes.NewBuffer(nil)
	err = lbnzm.Execute(buf, struct{
		Index int
		StartCmd string
		DBPath string
		StdoutPath string
		StderrPath string
		HTTPAddr string
	}{
		Index: node.index,
		StartCmd: cmd,
		DBPath: node.dbPath(),
		StdoutPath: node.stdoutPath(),
		StderrPath: node.stderrPath(),
		HTTPAddr: fmt.Sprintf("http://localhost:%d", node.httpPort()),
	})
	return buf.String(), err
}

func (node *LighthouseBeaconNode) ZChildren() []e2ez.ZPage {
	return []e2ez.ZPage{}
}

var _ e2ez.ZPage = &LighthouseBeaconNode{}
