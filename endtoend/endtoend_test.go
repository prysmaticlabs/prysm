package endtoend

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/endtoend/components"
	ev "github.com/prysmaticlabs/prysm/endtoend/evaluators"
	"github.com/prysmaticlabs/prysm/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/endtoend/params"
	"github.com/prysmaticlabs/prysm/endtoend/types"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc"
)

func init() {
	state.SkipSlotCache.Disable()
}

func runEndToEndTest(t *testing.T, config *types.E2EConfig) {
	t.Logf("Shard index: %d\n", e2e.TestParams.TestShardIndex)
	t.Logf("Starting time: %s\n", time.Now().String())
	t.Logf("Log Path: %s\n\n", e2e.TestParams.LogPath)

	keystorePath, eth1PID := components.StartEth1Node(t)
	multiAddrs, bProcessIDs := components.StartBeaconNodes(t, config)
	valProcessIDs := components.StartValidators(t, config, keystorePath)
	processIDs := append(valProcessIDs, bProcessIDs...)
	processIDs = append(processIDs, eth1PID)
	defer helpers.LogOutput(t, config)
	defer helpers.KillProcesses(t, processIDs)

	if config.TestSlasher {
		slasherPIDs := components.StartSlashers(t)
		defer helpers.KillProcesses(t, slasherPIDs)
	}

	beaconLogFile, err := os.Open(path.Join(e2e.TestParams.LogPath, fmt.Sprintf(e2e.BeaconNodeLogFileName, 0)))
	if err != nil {
		t.Fatal(err)
	}
	if err := helpers.WaitForTextInFile(beaconLogFile, "Chain started within the last epoch"); err != nil {
		t.Fatalf("failed to find chain start in logs, this means the chain did not start: %v", err)
	}

	// Failing early in case chain doesn't start.
	if t.Failed() {
		return
	}

	conns := make([]*grpc.ClientConn, e2e.TestParams.BeaconNodeCount)
	for i := 0; i < len(conns); i++ {
		conn, err := grpc.Dial(fmt.Sprintf("127.0.0.1:%d", e2e.TestParams.BeaconNodeRPCPort+i), grpc.WithInsecure())
		if err != nil {
			t.Fatalf("Failed to dial: %v", err)
		}
		conns[i] = conn
		defer func() {
			if err := conn.Close(); err != nil {
				t.Log(err)
			}
		}()
	}
	nodeClient := eth.NewNodeClient(conns[0])
	genesis, err := nodeClient.GetGenesis(context.Background(), &ptypes.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	// Small offset so evaluators perform in the middle of an epoch.
	epochSeconds := params.BeaconConfig().SecondsPerSlot * params.BeaconConfig().SlotsPerEpoch
	genesisTime := time.Unix(genesis.GenesisTime.Seconds+int64(epochSeconds/2), 0)

	ticker := helpers.GetEpochTicker(genesisTime, epochSeconds)
	for currentEpoch := range ticker.C() {
		for _, evaluator := range config.Evaluators {
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

		if t.Failed() || currentEpoch >= config.EpochsToRun-1 {
			ticker.Done()
			if t.Failed() {
				return
			}
			break
		}
	}

	if !config.TestSync {
		return
	}

	multiAddr, processID := components.StartNewBeaconNode(t, config, multiAddrs)
	multiAddrs = append(multiAddrs, multiAddr)
	index := e2e.TestParams.BeaconNodeCount
	syncConn, err := grpc.Dial(fmt.Sprintf("127.0.0.1:%d", e2e.TestParams.BeaconNodeRPCPort+index), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	conns = append(conns, syncConn)

	// Sleep until the next epoch to give time for the newly started node to sync.
	extraTimeToSync := (config.EpochsToRun+3)*epochSeconds + 60
	genesisTime.Add(time.Duration(extraTimeToSync) * time.Second)
	// Wait until middle of epoch to request to prevent conflicts.
	time.Sleep(time.Until(genesisTime))

	syncLogFile, err := os.Open(path.Join(e2e.TestParams.LogPath, fmt.Sprintf(e2e.BeaconNodeLogFileName, index)))
	if err != nil {
		t.Fatal(err)
	}
	defer helpers.LogErrorOutput(t, syncLogFile, "beacon chain node", index)
	defer helpers.KillProcesses(t, []int{processID})
	if err := helpers.WaitForTextInFile(syncLogFile, "Synced up to"); err != nil {
		t.Fatalf("Failed to sync: %v", err)
	}

	syncEvaluators := []types.Evaluator{ev.FinishedSyncing, ev.AllNodesHaveSameHead}
	for _, evaluator := range syncEvaluators {
		t.Run(evaluator.Name, func(t *testing.T) {
			if err := evaluator.Evaluation(conns...); err != nil {
				t.Errorf("evaluation failed for sync node: %v", err)
			}
		})
	}
}
