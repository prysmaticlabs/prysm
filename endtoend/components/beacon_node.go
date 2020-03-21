package components

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/prysmaticlabs/prysm/endtoend/helpers"
	"github.com/prysmaticlabs/prysm/endtoend/types"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var BeaconNodeLogFileName = "beacon-%d.log"

// StartBeaconNodes starts the requested amount of beacon nodes, passing in the deposit contract given.
func StartBeaconNodes(t *testing.T, config *types.E2EConfig) []*types.BeaconNodeInfo {
	var nodeInfo []*types.BeaconNodeInfo
	for i := uint64(0); i < config.NumBeaconNodes; i++ {
		newNode := StartNewBeaconNode(t, config, nodeInfo)
		nodeInfo = append(nodeInfo, newNode)
	}
	return nodeInfo
}

// StartNewBeaconNode starts a fresh beacon node, connecting to all passed in beacon nodes.
func StartNewBeaconNode(t *testing.T, config *types.E2EConfig, beaconNodes []*types.BeaconNodeInfo) *types.BeaconNodeInfo {
	testPath := config.TestPath
	index := len(beaconNodes)
	binaryPath, found := bazel.FindBinary("beacon-chain", "beacon-chain")
	if !found {
		t.Log(binaryPath)
		t.Fatal("beacon chain binary not found")
	}

	stdOutFile, err := helpers.DeleteAndCreateFile(testPath, fmt.Sprintf(BeaconNodeLogFileName, index))
	if err != nil {
		t.Fatal(err)
	}

	args := []string{
		fmt.Sprintf("--datadir=%s/eth2-beacon-node-%d", testPath, index),
		fmt.Sprintf("--log-file=%s", stdOutFile.Name()),
		"--force-clear-db",
		"--no-discovery",
		"--http-web3provider=http://127.0.0.1:8745",
		"--web3provider=ws://127.0.0.1:8746",
		fmt.Sprintf("--min-sync-peers=%d", config.NumBeaconNodes-1),
		fmt.Sprintf("--deposit-contract=%s", config.ContractAddress.Hex()),
		fmt.Sprintf("--rpc-port=%d", 4200+index),
		fmt.Sprintf("--p2p-udp-port=%d", 12200+index),
		fmt.Sprintf("--p2p-tcp-port=%d", 13200+index),
		fmt.Sprintf("--monitoring-port=%d", 8280+index),
		fmt.Sprintf("--grpc-gateway-port=%d", 3400+index),
		fmt.Sprintf("--contract-deployment-block=%d", 0),
		fmt.Sprintf("--rpc-max-page-size=%d", params.BeaconConfig().MinGenesisActiveValidatorCount),
	}
	args = append(args, featureconfig.E2EBeaconChainFlags...)
	args = append(args, config.BeaconFlags...)

	// After the first node is made, have all following nodes connect to all previously made nodes.
	if index >= 1 {
		for p := 0; p < index; p++ {
			args = append(args, fmt.Sprintf("--peer=%s", beaconNodes[p].MultiAddr))
		}
	}

	cmd := exec.Command(binaryPath, args...)
	t.Logf("Starting beacon chain %d with flags: %s", index, strings.Join(args[2:], " "))
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start beacon node: %v", err)
	}

	if err = helpers.WaitForTextInFile(stdOutFile, "Node started p2p server"); err != nil {
		t.Fatalf("could not find multiaddr for node %d, this means the node had issues starting: %v", index, err)
	}

	multiAddr, err := getMultiAddrFromLogFile(stdOutFile.Name())
	if err != nil {
		t.Fatalf("could not get multiaddr for node %d: %v", index, err)
	}

	return &types.BeaconNodeInfo{
		ProcessID:   cmd.Process.Pid,
		DataDir:     fmt.Sprintf("%s/eth2-beacon-node-%d", testPath, index),
		RPCPort:     4200 + uint64(index),
		MonitorPort: 8280 + uint64(index),
		GRPCPort:    3400 + uint64(index),
		MultiAddr:   multiAddr,
	}
}

func getMultiAddrFromLogFile(name string) (string, error) {
	byteContent, err := ioutil.ReadFile(name)
	if err != nil {
		return "", err
	}
	contents := string(byteContent)

	searchText := "\"Node started p2p server\" multiAddr=\""
	startIdx := strings.Index(contents, searchText)
	if startIdx == -1 {
		return "", fmt.Errorf("did not find peer text in %s", contents)
	}
	startIdx += len(searchText)
	endIdx := strings.Index(contents[startIdx:], "\"")
	if endIdx == -1 {
		return "", fmt.Errorf("did not find peer text in %s", contents)
	}
	return contents[startIdx : startIdx+endIdx], nil
}
