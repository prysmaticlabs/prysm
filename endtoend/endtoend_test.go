// Package endtoend performs full a end-to-end test for Prysm,
// including spinning up an ETH1 dev chain, sending deposits to the deposit
// contract, and making sure the beacon node and validators are running and
// performing properly for a few epochs.
package endtoend

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/eth2-types"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/endtoend/components"
	ev "github.com/prysmaticlabs/prysm/endtoend/evaluators"
	"github.com/prysmaticlabs/prysm/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/endtoend/params"
	e2etypes "github.com/prysmaticlabs/prysm/endtoend/types"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"google.golang.org/grpc"
)

func init() {
	state.SkipSlotCache.Disable()
}

func runEndToEndTest(t *testing.T, config *e2etypes.E2EConfig) {
	t.Logf("Shard index: %d\n", e2e.TestParams.TestShardIndex)
	t.Logf("Starting time: %s\n", time.Now().String())
	t.Logf("Log Path: %s\n", e2e.TestParams.LogPath)

	minGenesisActiveCount := int(params.BeaconConfig().MinGenesisActiveValidatorCount)

	keystorePath := components.StartEth1Node(t)
	go components.SendAndMineDeposits(t, keystorePath, minGenesisActiveCount, 0, true /* partial */)
	bootnodeENR := components.StartBootnode(t)
	components.StartBeaconNodes(t, config, bootnodeENR)
	components.StartValidatorClients(t, config)
	defer helpers.LogOutput(t, config)
	if config.UsePprof {
		defer func() {
			for i := 0; i < e2e.TestParams.BeaconNodeCount; i++ {
				assert.NoError(t, helpers.WritePprofFiles(e2e.TestParams.LogPath, i))
			}
		}()
	}

	// Sleep depending on the count of validators, as generating the genesis state could take some time.
	time.Sleep(time.Duration(params.BeaconConfig().GenesisDelay) * time.Second)
	beaconLogFile, err := os.Open(path.Join(e2e.TestParams.LogPath, fmt.Sprintf(e2e.BeaconNodeLogFileName, 0)))
	require.NoError(t, err)
	t.Run("chain started", func(t *testing.T) {
		require.NoError(t, helpers.WaitForTextInFile(beaconLogFile, "Chain started in sync service"), "Chain did not start")
	})

	// Failing early in case chain doesn't start.
	if t.Failed() {
		return
	}

	if config.TestSlasher {
		go components.StartSlashers(t)
	}
	if config.TestDeposits {
		go components.StartNewValidatorClient(t, config, int(e2e.DepositCount), e2e.TestParams.BeaconNodeCount, minGenesisActiveCount)
		go components.SendAndMineDeposits(t, keystorePath, int(e2e.DepositCount), minGenesisActiveCount, false /* partial */)
	}

	conns := make([]*grpc.ClientConn, e2e.TestParams.BeaconNodeCount)
	for i := 0; i < len(conns); i++ {
		conn, err := grpc.Dial(fmt.Sprintf("127.0.0.1:%d", e2e.TestParams.BeaconNodeRPCPort+i), grpc.WithInsecure())
		require.NoError(t, err, "Failed to dial")
		conns[i] = conn
		defer func() {
			if err = conn.Close(); err != nil {
				t.Log(err)
			}
		}()
	}
	nodeClient := eth.NewNodeClient(conns[0])
	genesis, err := nodeClient.GetGenesis(context.Background(), &ptypes.Empty{})
	require.NoError(t, err)

	epochSeconds := params.BeaconConfig().SecondsPerSlot * params.BeaconConfig().SlotsPerEpoch
	epochSecondsHalf := time.Duration(int64(epochSeconds*1000)/2) * time.Millisecond
	// Adding a half slot here to ensure the requests are in the middle of an epoch.
	middleOfEpoch := epochSecondsHalf + slotutil.DivideSlotBy(2 /* half a slot */)
	genesisTime := time.Unix(genesis.GenesisTime.Seconds, 0)
	// Offsetting the ticker from genesis so it ticks in the middle of an epoch, in order to keep results consistent.
	tickingStartTime := genesisTime.Add(middleOfEpoch)

	ticker := helpers.NewEpochTicker(tickingStartTime, epochSeconds)
	for currentEpoch := range ticker.C() {
		for _, evaluator := range config.Evaluators {
			// Only run if the policy says so.
			if !evaluator.Policy(types.Epoch(currentEpoch)) {
				continue
			}
			t.Run(fmt.Sprintf(evaluator.Name, currentEpoch), func(t *testing.T) {
				assert.NoError(t, evaluator.Evaluation(conns...), "Evaluation failed for epoch %d: %v", currentEpoch, err)
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

	index := e2e.TestParams.BeaconNodeCount
	components.StartNewBeaconNode(t, config, index, bootnodeENR)
	syncConn, err := grpc.Dial(fmt.Sprintf("127.0.0.1:%d", e2e.TestParams.BeaconNodeRPCPort+index), grpc.WithInsecure())
	require.NoError(t, err, "Failed to dial")
	conns = append(conns, syncConn)

	// Sleep a second for every 4 blocks that need to be synced for the newly started node.
	extraSecondsToSync := (config.EpochsToRun)*epochSeconds + (params.BeaconConfig().SlotsPerEpoch / 4 * config.EpochsToRun)
	waitForSync := tickingStartTime.Add(time.Duration(extraSecondsToSync) * time.Second)
	time.Sleep(time.Until(waitForSync))

	syncLogFile, err := os.Open(path.Join(e2e.TestParams.LogPath, fmt.Sprintf(e2e.BeaconNodeLogFileName, index)))
	require.NoError(t, err)
	defer helpers.LogErrorOutput(t, syncLogFile, "beacon chain node", index)
	t.Run("sync completed", func(t *testing.T) {
		assert.NoError(t, helpers.WaitForTextInFile(syncLogFile, "Synced up to"), "Failed to sync")
	})
	if t.Failed() {
		return
	}

	// Sleep a slot to make sure the synced state is made.
	time.Sleep(time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second)
	syncEvaluators := []e2etypes.Evaluator{ev.FinishedSyncing, ev.AllNodesHaveSameHead}
	for _, evaluator := range syncEvaluators {
		t.Run(evaluator.Name, func(t *testing.T) {
			assert.NoError(t, evaluator.Evaluation(conns...), "Evaluation failed for sync node")
		})
	}
}
