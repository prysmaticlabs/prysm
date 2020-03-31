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
func StartBeaconNodes(t *testing.T, config *types.E2EConfig) ([]string, []int) {
	var multiAddrs []string
	var processIDs []int
	for i := 0; i < e2e.TestParams.BeaconNodeCount; i++ {
		multiAddr, pID := StartNewBeaconNode(t, config, multiAddrs)
		multiAddrs = append(multiAddrs, multiAddr)
		processIDs = append(processIDs, pID)
	}
	return multiAddrs, processIDs
}

// StartNewBeaconNode starts a fresh beacon node, connecting to all passed in beacon nodes.
func StartNewBeaconNode(t *testing.T, config *types.E2EConfig, multiAddrs []string) (string, int) {
	index := len(multiAddrs)
	binaryPath, found := bazel.FindBinary("beacon-chain", "beacon-chain")
	if !found {
		t.Fatal("beacon chain binary not found")
	}

	stdOutFile, err := helpers.DeleteAndCreateFile(e2e.TestParams.LogPath, fmt.Sprintf(e2e.BeaconNodeLogFileName, index))
	if err != nil {
		t.Fatal(err)
	}

	args := []string{
		fmt.Sprintf("--datadir=%s/eth2-beacon-node-%d", e2e.TestParams.TestPath, index),
		fmt.Sprintf("--log-file=%s", stdOutFile.Name()),
		fmt.Sprintf("--deposit-contract=%s", e2e.TestParams.ContractAddress.Hex()),
		fmt.Sprintf("--rpc-port=%d", e2e.TestParams.BeaconNodeRPCPort+index),
		fmt.Sprintf("--http-web3provider=http://127.0.0.1:%d", e2e.TestParams.Eth1RPCPort),
		fmt.Sprintf("--web3provider=ws://127.0.0.1:%d", e2e.TestParams.Eth1RPCPort+1),
		fmt.Sprintf("--min-sync-peers=%d", e2e.TestParams.BeaconNodeCount-1),
		fmt.Sprintf("--p2p-udp-port=%d", e2e.TestParams.BeaconNodeRPCPort+index+10),      //12200
		fmt.Sprintf("--p2p-tcp-port=%d", e2e.TestParams.BeaconNodeRPCPort+index+20),      //13200
		fmt.Sprintf("--monitoring-port=%d", e2e.TestParams.BeaconNodeRPCPort+index+30),   //8280
		fmt.Sprintf("--grpc-gateway-port=%d", e2e.TestParams.BeaconNodeRPCPort+index+40), // 3400
		fmt.Sprintf("--contract-deployment-block=%d", 0),
		fmt.Sprintf("--rpc-max-page-size=%d", params.BeaconConfig().MinGenesisActiveValidatorCount),
		"--force-clear-db",
		"--no-discovery",
	}
	args = append(args, featureconfig.E2EBeaconChainFlags...)
	args = append(args, config.BeaconFlags...)

	// After the first node is made, have all following nodes connect to all previously made nodes.
	if index >= 1 {
		for p := 0; p < index; p++ {
			args = append(args, fmt.Sprintf("--peer=%s", multiAddrs[p]))
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

	return multiAddr, cmd.Process.Pid
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
