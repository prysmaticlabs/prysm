package endtoend

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc"
)

func runEndToEndTest(t *testing.T, config *end2EndConfig) {
	tmpPath := bazel.TestTmpDir()
	config.tmpPath = tmpPath
	t.Logf("Test Path: %s\n", tmpPath)

	contractAddr, keystorePath, eth1PID := startEth1(t, tmpPath)
	config.contractAddr = contractAddr
	beaconNodes := startBeaconNodes(t, config)
	valClients := initializeValidators(t, config, keystorePath, beaconNodes)
	processIDs := []int{eth1PID}
	for _, vv := range valClients {
		processIDs = append(processIDs, vv.processID)
	}
	for _, bb := range beaconNodes {
		processIDs = append(processIDs, bb.processID)
	}
	defer logOutput(t, tmpPath)
	defer killProcesses(t, processIDs)

	if config.numBeaconNodes > 1 {
		t.Run("all_peers_connect", func(t *testing.T) {
			for _, bNode := range beaconNodes {
				if err := peersConnect(bNode.monitorPort, config.numBeaconNodes-1); err != nil {
					t.Fatalf("Failed to connect to peers: %v", err)
				}
			}
		})
	}

	beaconLogFile, err := os.Open(path.Join(tmpPath, "beacon-0.log"))
	if err != nil {
		t.Fatal(err)
	}
	if err := waitForTextInFile(beaconLogFile, "Sending genesis time notification"); err != nil {
		t.Fatalf("failed to find genesis in logs, this means the chain did not start: %v", err)
	}

	// Failing early in case chain doesn't start.
	if t.Failed() {
		return
	}

	conn, err := grpc.Dial("127.0.0.1:4000", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	beaconClient := eth.NewBeaconChainClient(conn)
	nodeClient := eth.NewNodeClient(conn)

	genesis, err := nodeClient.GetGenesis(context.Background(), &ptypes.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	// Small offset so evaluators perform in the middle of an epoch.
	epochSeconds := params.BeaconConfig().SecondsPerSlot * params.BeaconConfig().SlotsPerEpoch
	genesisTime := time.Unix(genesis.GenesisTime.Seconds+int64(epochSeconds/2), 0)
	currentEpoch := uint64(0)
	ticker := GetEpochTicker(genesisTime, epochSeconds)
	for c := range ticker.C() {
		if c >= config.epochsToRun || t.Failed() {
			ticker.Done()
			break
		}

		for _, evaluator := range config.evaluators {
			// Only run if the policy says so.
			if !evaluator.Policy(currentEpoch) {
				continue
			}
			t.Run(fmt.Sprintf(evaluator.Name, currentEpoch), func(t *testing.T) {
				if err := evaluator.Evaluation(beaconClient); err != nil {
					t.Fatalf("evaluation failed for epoch %d: %v", currentEpoch, err)
				}
			})
		}
		currentEpoch++
	}

	if currentEpoch < config.epochsToRun {
		t.Fatalf("Test ended prematurely, only reached epoch %d", currentEpoch)
	}
}

func peersConnect(port uint64, expectedPeers uint64) error {
	response, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/p2p", port))
	if err != nil {
		return errors.Wrap(err, "failed to reach p2p metrics page")
	}
	dataInBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	pageContent := string(dataInBytes)
	if err := response.Body.Close(); err != nil {
		return err
	}
	// Subtracting by 2 here since the libp2p page has "3 peers" as text.
	// With a starting index before the "p", going two characters back should give us
	// the number we need.
	startIdx := strings.Index(pageContent, "peers") - 2
	if startIdx == -3 {
		return fmt.Errorf("could not find needed text in %s", pageContent)
	}
	peerCount, err := strconv.Atoi(pageContent[startIdx : startIdx+1])
	if err != nil {
		return err
	}
	if expectedPeers != uint64(peerCount) {
		return fmt.Errorf("unexpected amount of peers, expected %d, received %d", expectedPeers, peerCount)
	}
	return nil
}

func killProcesses(t *testing.T, pIDs []int) {
	for _, id := range pIDs {
		process, err := os.FindProcess(id)
		if err != nil {
			t.Fatalf("Could not find process %d: %v", id, err)
		}
		if err := process.Kill(); err != nil {
			t.Fatal(err)
		}
	}
}

func logOutput(t *testing.T, tmpPath string) {
	if t.Failed() {
		beacon0LogFile, err := os.Open(path.Join(tmpPath, "beacon-0.log"))
		if err != nil {
			t.Fatal(err)
		}
		scanner := bufio.NewScanner(beacon0LogFile)
		t.Log("Beacon chain node output:")
		for scanner.Scan() {
			currentLine := scanner.Text()
			t.Log(currentLine)
		}
		t.Log("End of beacon chain node output")
	}
}
