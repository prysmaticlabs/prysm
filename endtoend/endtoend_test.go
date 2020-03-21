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
	defer logOutput(t, config)
	defer killProcesses(t, processIDs)

	if config.testSlasher {
		slasherPIDs := startSlashers(t, config)
		defer killProcesses(t, slasherPIDs)
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

	conns := make([]*grpc.ClientConn, len(beaconNodes))
	for i := 0; i < len(conns); i++ {
		conn, err := grpc.Dial(fmt.Sprintf("127.0.0.1:%d", beaconNodes[i].RPCPort), grpc.WithInsecure())
		if err != nil {
			t.Fatalf("Failed to dial: %v", err)
		}
		conns[i] = conn
		defer conn.Close()
	}
	nodeClient := eth.NewNodeClient(conns[0])
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
				if err := evaluator.Evaluation(conns...); err != nil {
					t.Errorf("evaluation failed for epoch %d: %v", currentEpoch, err)
				}
			})
		}

		if t.Failed() || currentEpoch >= config.epochsToRun {
			ticker.Done()
			if t.Failed() {
				return
			}
			break
		}
	}

	if !config.testSync {
		return
	}

	syncNodeInfo := startNewBeaconNode(t, config, beaconNodes)
	beaconNodes = append(beaconNodes, syncNodeInfo)
	syncConn, err := grpc.Dial(fmt.Sprintf("127.0.0.1:%d", syncNodeInfo.RPCPort), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	conns = append(conns, syncConn)
	index := uint64(len(beaconNodes) - 1)

	// Sleep until the next epoch to give time for the newly started node to sync.
	extraTimeToSync := (config.epochsToRun+3)*epochSeconds + 90
	genesisTime.Add(time.Duration(extraTimeToSync) * time.Second)
	// Wait until middle of epoch to request to prevent conflicts.
	time.Sleep(time.Until(genesisTime))

	syncLogFile, err := os.Open(path.Join(tmpPath, fmt.Sprintf(beaconNodeLogFileName, index)))
	if err != nil {
		t.Fatal(err)
	}
	defer logErrorOutput(t, syncLogFile, "beacon chain node", index)
	defer killProcesses(t, []int{syncNodeInfo.ProcessID})
	if err := waitForTextInFile(syncLogFile, "Synced up to"); err != nil {
		t.Fatalf("Failed to sync: %v", err)
	}

	syncEvaluators := []ev.Evaluator{ev.FinishedSyncing, ev.AllNodesHaveSameHead}
	for _, evaluator := range syncEvaluators {
		t.Run(evaluator.Name, func(t *testing.T) {
			if err := evaluator.Evaluation(conns...); err != nil {
				t.Errorf("evaluation failed for sync node: %v", err)
			}
		})
	}
}
