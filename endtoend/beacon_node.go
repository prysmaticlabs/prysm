package endtoend

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ethereum/go-ethereum/common"
	ev "github.com/prysmaticlabs/prysm/endtoend/evaluators"
)

type beaconNodeInfo struct {
	processID   int
	datadir     string
	rpcPort     uint64
	monitorPort uint64
	grpcPort    uint64
	multiAddr   string
}

type end2EndConfig struct {
	minimalConfig  bool
	tmpPath        string
	epochsToRun    uint64
	numValidators  uint64
	numBeaconNodes uint64
	contractAddr   common.Address
	evaluators     []ev.Evaluator
}

// startBeaconNodes starts the requested amount of beacon nodes, passing in the deposit contract given.
func startBeaconNodes(t *testing.T, config *end2EndConfig) []*beaconNodeInfo {
	numNodes := config.numBeaconNodes

	nodeInfo := []*beaconNodeInfo{}
	for i := uint64(0); i < numNodes; i++ {
		newNode := startNewBeaconNode(t, config, nodeInfo)
		nodeInfo = append(nodeInfo, newNode)
	}

	return nodeInfo
}

func startNewBeaconNode(t *testing.T, config *end2EndConfig, beaconNodes []*beaconNodeInfo) *beaconNodeInfo {
	tmpPath := config.tmpPath
	index := len(beaconNodes)
	binaryPath, found := bazel.FindBinary("beacon-chain", "beacon-chain")
	if !found {
		t.Log(binaryPath)
		t.Fatal("beacon chain binary not found")
	}
	file, err := os.Create(path.Join(tmpPath, fmt.Sprintf("beacon-%d.log", index)))
	if err != nil {
		t.Fatal(err)
	}

	args := []string{
		"--no-discovery",
		"--http-web3provider=http://127.0.0.1:8545",
		"--web3provider=ws://127.0.0.1:8546",
		fmt.Sprintf("--datadir=%s/eth2-beacon-node-%d", tmpPath, index),
		fmt.Sprintf("--deposit-contract=%s", config.contractAddr.Hex()),
		fmt.Sprintf("--rpc-port=%d", 4000+index),
		fmt.Sprintf("--p2p-udp-port=%d", 12000+index),
		fmt.Sprintf("--p2p-tcp-port=%d", 13000+index),
		fmt.Sprintf("--monitoring-port=%d", 8080+index),
		fmt.Sprintf("--grpc-gateway-port=%d", 3200+index),
	}

	if config.minimalConfig {
		args = append(args, "--minimal-config")
	}
	// After the first node is made, have all following nodes connect to all previously made nodes.
	if index >= 1 {
		for p := 0; p < index; p++ {
			args = append(args, fmt.Sprintf("--peer=%s", beaconNodes[p].multiAddr))
		}
	}

	t.Logf("Starting beacon chain with flags %s", strings.Join(args, " "))
	cmd := exec.Command(binaryPath, args...)
	cmd.Stderr = file
	cmd.Stdout = file
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start beacon node: %v", err)
	}

	if err = waitForTextInFile(file, "Node started p2p server"); err != nil {
		t.Fatal(err)
	}

	multiAddr, err := getMultiAddrFromLogFile(file.Name())
	if err != nil {
		t.Fatal(err)
	}

	return &beaconNodeInfo{
		processID:   cmd.Process.Pid,
		datadir:     fmt.Sprintf("%s/eth2-beacon-node-%d", tmpPath, index),
		rpcPort:     4000 + uint64(index),
		monitorPort: 8080 + uint64(index),
		grpcPort:    3200 + uint64(index),
		multiAddr:   multiAddr,
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

func waitForTextInFile(file *os.File, text string) error {
	wait := 0
	// Putting the wait cap at 24 seconds.
	totalWait := 24
	for wait < totalWait {
		time.Sleep(2 * time.Second)
		// Rewind the file pointer to the start of the file so we can read it again.
		_, err := file.Seek(0, io.SeekStart)
		if err != nil {
			return fmt.Errorf("could not rewind file to start: %v", err)
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), text) {
				return nil
			}
		}
		if err := scanner.Err(); err != nil {
			return err
		}
		wait += 2
	}
	contents, err := ioutil.ReadFile(file.Name())
	if err != nil {
		return err
	}
	return fmt.Errorf("could not find requested text %s in logs:\n%s", text, string(contents))
}
