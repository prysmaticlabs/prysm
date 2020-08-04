// Package components defines utilities to spin up actual
// beacon node and validator processes as needed by end to end tests.
package components

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/prysmaticlabs/prysm/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/endtoend/params"
	"github.com/prysmaticlabs/prysm/endtoend/types"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// StartBeaconNodes starts the requested amount of beacon nodes, passing in the deposit contract given.
func StartBeaconNodes(t *testing.T, config *types.E2EConfig, enr string) []int {
	var pIDs []int
	for i := 0; i < e2e.TestParams.BeaconNodeCount; i++ {
		pIDs = append(pIDs, StartNewBeaconNode(t, config, i, enr))
	}
	return pIDs
}

// StartNewBeaconNode starts a fresh beacon node, connecting to all passed in beacon nodes.
func StartNewBeaconNode(t *testing.T, config *types.E2EConfig, index int, enr string) int {
	binaryPath, found := bazel.FindBinary("beacon-chain", "beacon-chain")
	if !found {
		t.Log(binaryPath)
		t.Fatal("beacon chain binary not found")
	}

	stdOutFile, err := helpers.DeleteAndCreateFile(e2e.TestParams.LogPath, fmt.Sprintf(e2e.BeaconNodeLogFileName, index))
	if err != nil {
		t.Fatal(err)
	}
	profileFile, err := helpers.DeleteAndCreateFile(e2e.TestParams.LogPath, fmt.Sprintf(e2e.BeaconNodeCPUProfileFileName, index))
	if err != nil {
		t.Fatal(err)
	}

	args := []string{
		fmt.Sprintf("--datadir=%s/eth2-beacon-node-%d", e2e.TestParams.TestPath, index),
		fmt.Sprintf("--log-file=%s", stdOutFile.Name()),
		fmt.Sprintf("--cpuprofile=%s", profileFile.Name()),
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
		fmt.Sprintf("--pprofport=%d", e2e.TestParams.BeaconNodeRPCPort+index+50),
		"--pprof",
		"--verbosity=trace",
		"--force-clear-db",
		"--e2e-config",
	}
	args = append(args, featureconfig.E2EBeaconChainFlags...)
	args = append(args, config.BeaconFlags...)

	cmd := exec.Command(binaryPath, args...)
	t.Logf("Starting beacon chain %d with flags: %s", index, strings.Join(args[2:], " "))
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start beacon node: %v", err)
	}

	if err = helpers.WaitForTextInFile(stdOutFile, "RPC-API listening on port"); err != nil {
		t.Fatalf("could not find multiaddr for node %d, this means the node had issues starting: %v", index, err)
	}
	return cmd.Process.Pid
}

// StartBootnode starts a bootnode and returns its ENR and process ID.
func StartBootnode(t *testing.T) string {
	binaryPath, found := bazel.FindBinary("tools/bootnode", "bootnode")
	if !found {
		t.Log(binaryPath)
		t.Fatal("boot node binary not found")
	}

	stdOutFile, err := helpers.DeleteAndCreateFile(e2e.TestParams.LogPath, e2e.BootNodeLogFileName)
	if err != nil {
		t.Fatal(err)
	}

	args := []string{
		fmt.Sprintf("--log-file=%s", stdOutFile.Name()),
		fmt.Sprintf("--discv5-port=%d", e2e.TestParams.BootNodePort),
		fmt.Sprintf("--metrics-port=%d", e2e.TestParams.BootNodePort+20),
		"--debug",
	}

	cmd := exec.Command(binaryPath, args...)
	cmd.Stdout = stdOutFile
	cmd.Stderr = stdOutFile
	t.Logf("Starting boot node with flags: %s", strings.Join(args[1:], " "))
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start beacon node: %v", err)
	}

	if err = helpers.WaitForTextInFile(stdOutFile, "Running bootnode"); err != nil {
		t.Fatalf("could not find enr for bootnode, this means the bootnode had issues starting: %v", err)
	}

	enr, err := getENRFromLogFile(stdOutFile.Name())
	if err != nil {
		t.Fatalf("could not get enr for bootnode: %v", err)
	}

	return enr
}

func getENRFromLogFile(name string) (string, error) {
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
