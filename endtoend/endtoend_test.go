package endtoend

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	ptypes "github.com/gogo/protobuf/types"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	ev "github.com/prysmaticlabs/prysm/endtoend/evaluators"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc"
)

func runEndToEndTest(t *testing.T, config *end2EndConfig) {
	tmpPath := bazel.TestTmpDir()
	config.tmpPath = tmpPath
	t.Logf("Starting time: %s\n", time.Now().String())
	t.Logf("Test Path: %s\n\n", tmpPath)

	contractAddr, keystorePath, eth1PID := startEth1(t, tmpPath)
	config.contractAddr = contractAddr
	beaconNodes := startBeaconNodes(t, config)
	valClients := initializeValidators(t, config, keystorePath)
	processIDs := []int{eth1PID}
	for _, vv := range valClients {
		processIDs = append(processIDs, vv.processID)
	}
	for _, bb := range beaconNodes {
		processIDs = append(processIDs, bb.ProcessID)
	}
	defer logOutput(t, tmpPath, config)
	defer killProcesses(t, processIDs)

	if config.numBeaconNodes > 1 {
		t.Run("all_peers_connect", func(t *testing.T) {
			if err := ev.PeersConnect(beaconNodes); err != nil {
				t.Fatalf("Failed to connect to peers: %v", err)
			}
		})
	}

	beaconLogFile, err := os.Open(path.Join(tmpPath, fmt.Sprintf(beaconNodeLogFileName, 0)))
	if err != nil {
		t.Fatal(err)
	}
	if err := waitForTextInFile(beaconLogFile, "Chain started within the last epoch"); err != nil {
		t.Fatalf("failed to find chain start in logs, this means the chain did not start: %v", err)
	}

	// Failing early in case chain doesn't start.
	if t.Failed() {
		return
	}

	conn, err := grpc.Dial("127.0.0.1:4200", grpc.WithInsecure())
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

	ticker := GetEpochTicker(genesisTime, epochSeconds)
	for currentEpoch := range ticker.C() {
		for _, evaluator := range config.evaluators {
			// Only run if the policy says so.
			if !evaluator.Policy(currentEpoch) {
				continue
			}
			t.Run(fmt.Sprintf(evaluator.Name, currentEpoch), func(t *testing.T) {
				if err := evaluator.Evaluation(beaconClient); err != nil {
					t.Errorf("evaluation failed for epoch %d: %v", currentEpoch, err)
				}
			})
		}

		if t.Failed() || currentEpoch >= config.epochsToRun {
			if err := conn.Close(); err != nil {
				t.Fatal(err)
			}
			ticker.Done()
			if t.Failed() {
				return
			}
			break
		}
	}

	syncNodeInfo := startNewBeaconNode(t, config, beaconNodes)
	beaconNodes = append(beaconNodes, syncNodeInfo)
	index := uint64(len(beaconNodes) - 1)

	// Sleep until the next epoch to give time for the newly started node to sync.
	extraTimeToSync := (config.epochsToRun+3)*epochSeconds + 60
	genesisTime.Add(time.Duration(extraTimeToSync) * time.Second)
	// Wait until middle of epoch to request to prevent conflicts.
	time.Sleep(time.Until(genesisTime))

	syncLogFile, err := os.Open(path.Join(tmpPath, fmt.Sprintf(beaconNodeLogFileName, index)))
	if err != nil {
		t.Fatal(err)
	}
	if err := waitForTextInFile(syncLogFile, "Synced up to"); err != nil {
		t.Fatalf("Failed to sync: %v", err)
	}

	t.Run("node_finishes_sync", func(t *testing.T) {
		if err := ev.FinishedSyncing(syncNodeInfo.RPCPort); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("all_nodes_have_correct_head", func(t *testing.T) {
		if err := ev.AllChainsHaveSameHead(beaconNodes); err != nil {
			t.Fatal(err)
		}
	})

	defer logErrorOutput(t, syncLogFile, "beacon chain node", index)
	defer killProcesses(t, []int{syncNodeInfo.ProcessID})
}
