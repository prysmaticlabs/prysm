package endtoend

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ethereum/go-ethereum/common"
	ev "github.com/prysmaticlabs/prysm/endtoend/evaluators"
	"github.com/prysmaticlabs/prysm/shared/params"
)

type end2EndConfig struct {
	beaconFlags    []string
	validatorFlags []string
	tmpPath        string
	epochsToRun    uint64
	numValidators  uint64
	numBeaconNodes uint64
	contractAddr   common.Address
	evaluators     []ev.Evaluator
}

var beaconNodeLogFileName = "beacon-%d.log"

// startBeaconNodes starts the requested amount of beacon nodes, passing in the deposit contract given.
func startBeaconNodes(t *testing.T, config *end2EndConfig) []*ev.BeaconNodeInfo {
	numNodes := config.numBeaconNodes

	nodeInfo := []*ev.BeaconNodeInfo{}
	for i := uint64(0); i < numNodes; i++ {
		newNode := startNewBeaconNode(t, config, nodeInfo)
		nodeInfo = append(nodeInfo, newNode)
	}

	return nodeInfo
}

func startNewBeaconNode(t *testing.T, config *end2EndConfig, beaconNodes []*ev.BeaconNodeInfo) *ev.BeaconNodeInfo {
	tmpPath := config.tmpPath
	index := len(beaconNodes)
	binaryPath, found := bazel.FindBinary("beacon-chain", "beacon-chain")
	if !found {
		t.Log(binaryPath)
		t.Fatal("beacon chain binary not found")
	}

	stdOutFile, err := deleteAndCreateFile(tmpPath, fmt.Sprintf(beaconNodeLogFileName, index))
	if err != nil {
		t.Fatal(err)
	}

	args := []string{
		"--force-clear-db",
		"--no-discovery",
		"--http-web3provider=http://127.0.0.1:8745",
		"--web3provider=ws://127.0.0.1:8746",
		fmt.Sprintf("--min-sync-peers=%d", config.numBeaconNodes),
		fmt.Sprintf("--datadir=%s/eth2-beacon-node-%d", tmpPath, index),
		fmt.Sprintf("--deposit-contract=%s", config.contractAddr.Hex()),
		fmt.Sprintf("--rpc-port=%d", 4200+index),
		fmt.Sprintf("--p2p-udp-port=%d", 12200+index),
		fmt.Sprintf("--p2p-tcp-port=%d", 13200+index),
		fmt.Sprintf("--monitoring-port=%d", 8280+index),
		fmt.Sprintf("--grpc-gateway-port=%d", 3400+index),
		fmt.Sprintf("--contract-deployment-block=%d", 0),
		fmt.Sprintf("--rpc-max-page-size=%d", params.BeaconConfig().MinGenesisActiveValidatorCount),
		fmt.Sprintf("--log-file=%s", stdOutFile.Name()),
	}
	args = append(args, config.beaconFlags...)

	// After the first node is made, have all following nodes connect to all previously made nodes.
	if index >= 1 {
		for p := 0; p < index; p++ {
			args = append(args, fmt.Sprintf("--peer=%s", beaconNodes[p].MultiAddr))
		}
	}

	t.Logf("Starting beacon chain %d with flags: %s", index, strings.Join(args, " "))
	cmd := exec.Command(binaryPath, args...)
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start beacon node: %v", err)
	}

	if err = waitForTextInFile(stdOutFile, "Node started p2p server"); err != nil {
		t.Fatalf("could not find multiaddr for node %d, this means the node had issues starting: %v", index, err)
	}

	multiAddr, err := getMultiAddrFromLogFile(stdOutFile.Name())
	if err != nil {
		t.Fatalf("could not get multiaddr for node %d: %v", index, err)
	}

	return &ev.BeaconNodeInfo{
		ProcessID:   cmd.Process.Pid,
		DataDir:     fmt.Sprintf("%s/eth2-beacon-node-%d", tmpPath, index),
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
