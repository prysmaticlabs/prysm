package endtoend

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	cmdshared "github.com/prysmaticlabs/prysm/cmd"
	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/config/params"
	ev "github.com/prysmaticlabs/prysm/testing/endtoend/evaluators"
	"github.com/prysmaticlabs/prysm/testing/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/testing/endtoend/params"
	e2eParams "github.com/prysmaticlabs/prysm/testing/endtoend/params"
	"github.com/prysmaticlabs/prysm/testing/endtoend/types"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestEndToEnd_MainnetConfig(t *testing.T) {
	e2eMainnet(t, false /*usePrysmSh*/)
}

func TestEndToeNd_MainnetConfig_Prater(t *testing.T) {
	// params.UseE2EMainnetConfig()
	require.NoError(t, e2eParams.Init(1))

	// Run for 10 epochs if not in long-running to confirm long-running has no issues.
	var err error
	epochsToRun := 10
	epochStr, longRunning := os.LookupEnv("E2E_EPOCHS")
	if longRunning {
		epochsToRun, err = strconv.Atoi(epochStr)
		require.NoError(t, err)
	}
	evals := []types.Evaluator{
		// ev.PeersConnect,
		// ev.HealthzCheck,
		// ev.MetricsCheck,
		// ev.ValidatorsAreActive,
		// ev.ValidatorsParticipatingAtEpoch(2),
		// ev.FinalizationOccurs(3),
		// ev.ProposeVoluntaryExit,
		// ev.ValidatorHasExited,
		// ev.ColdStateCheckpoint,
		// ev.ForkTransition,
		// ev.APIMiddlewareVerifyIntegrity,
		// ev.APIGatewayV1Alpha1VerifyIntegrity,
		ev.FinishedSyncing,
		// ev.AllNodesHaveSameHead,
	}
	index := 0
	testConfig := &types.E2EConfig{
		BeaconFlags: []string{
			"--" + cmdshared.DataDirFlag.Name + "=/tmp/beacon-chain-data",
			// fmt.Sprintf("--%s=%s/eth2-beacon-node-%d", cmdshared.DataDirFlag.Name, e2e.TestParams.TestPath, index),
			// fmt.Sprintf("--%s=%s", cmdshared.LogFileName.Name, stdOutFile.Name()),
			// fmt.Sprintf("--%s=%s", flags.DepositContractFlag.Name, e2e.TestParams.ContractAddress.Hex()),
			fmt.Sprintf("--%s=%d", flags.RPCPort.Name, e2e.TestParams.BeaconNodeRPCPort+index),
			fmt.Sprintf("--%s=https://goerli.infura.io/v3/c0c32ccb156047038de5b28f343313bd", flags.HTTPWeb3ProviderFlag.Name),
			fmt.Sprintf("--%s=%d", flags.MinSyncPeers.Name, e2e.TestParams.BeaconNodeCount-1),
			fmt.Sprintf("--%s=%d", cmdshared.P2PUDPPort.Name, e2e.TestParams.BeaconNodeRPCPort+index+e2e.PrysmBeaconUDPOffset),
			fmt.Sprintf("--%s=%d", cmdshared.P2PTCPPort.Name, e2e.TestParams.BeaconNodeRPCPort+index+e2e.PrysmBeaconTCPOffset),
			// fmt.Sprintf("--%s=%d", cmdshared.P2PMaxPeers.Name, expectedNumOfPeers),
			fmt.Sprintf("--%s=%d", flags.MonitoringPortFlag.Name, e2e.TestParams.BeaconNodeMetricsPort+index),
			fmt.Sprintf("--%s=%d", flags.GRPCGatewayPort.Name, e2e.TestParams.BeaconNodeRPCPort+index+e2e.PrysmBeaconGatewayOffset),
			// fmt.Sprintf("--%s=%d", flags.ContractDeploymentBlock.Name, 0),
			// fmt.Sprintf("--%s=%d", flags.MinPeersPerSubnet.Name, 0),
			fmt.Sprintf("--%s=%d", cmdshared.RPCMaxPageSizeFlag.Name, params.BeaconConfig().MinGenesisActiveValidatorCount),
			// fmt.Sprintf("--%s=%s", cmdshared.BootstrapNode.Name, enr),
			fmt.Sprintf("--%s=%s", cmdshared.VerbosityFlag.Name, "debug"),
			// "--" + cmdshared.ForceClearDB.Name,
			// "--" + cmdshared.E2EConfigFlag.Name,
			"--" + cmdshared.AcceptTosFlag.Name,
			"--" + flags.EnableDebugRPCEndpoints.Name,
			"--prater",
		},
		ValidatorFlags:          []string{},
		EpochsToRun:             uint64(epochsToRun),
		TestSync:                true,
		TestDeposits:            false,
		UseFixedPeerIDs:         false,
		UseValidatorCrossClient: false,
		UsePrysmShValidator:     false,
		UsePprof:                false,
		NoModifyBeaconFlags:     true,
		Evaluators:              evals,
	}

	newTestRunner(t, testConfig).run()
}

func e2eMainnet(t *testing.T, usePrysmSh bool) {
	// params.UseE2EMainnetConfig()
	require.NoError(t, e2eParams.InitMultiClient(e2eParams.StandardBeaconCount, e2eParams.StandardLighthouseNodeCount))

	// Run for 10 epochs if not in long-running to confirm long-running has no issues.
	var err error
	epochsToRun := 10
	epochStr, longRunning := os.LookupEnv("E2E_EPOCHS")
	if longRunning {
		epochsToRun, err = strconv.Atoi(epochStr)
		require.NoError(t, err)
	}
	_, crossClient := os.LookupEnv("RUN_CROSS_CLIENT")
	if usePrysmSh {
		// If using prysm.sh, run for only 6 epochs.
		// TODO(#9166): remove this block once v2 changes are live.
		epochsToRun = helpers.AltairE2EForkEpoch - 1
	}
	tracingPort := 9411 + e2eParams.TestParams.TestShardIndex
	tracingEndpoint := fmt.Sprintf("127.0.0.1:%d", tracingPort)
	evals := []types.Evaluator{
		ev.PeersConnect,
		ev.HealthzCheck,
		ev.MetricsCheck,
		ev.ValidatorsAreActive,
		ev.ValidatorsParticipatingAtEpoch(2),
		ev.FinalizationOccurs(3),
		ev.ProposeVoluntaryExit,
		ev.ValidatorHasExited,
		ev.ColdStateCheckpoint,
		ev.ForkTransition,
		ev.APIMiddlewareVerifyIntegrity,
		ev.APIGatewayV1Alpha1VerifyIntegrity,
		ev.FinishedSyncing,
		ev.AllNodesHaveSameHead,
	}
	testConfig := &types.E2EConfig{
		BeaconFlags: []string{
			fmt.Sprintf("--slots-per-archive-point=%d", params.BeaconConfig().SlotsPerEpoch*16),
			fmt.Sprintf("--tracing-endpoint=http://%s", tracingEndpoint),
			"--enable-tracing",
			"--trace-sample-fraction=1.0",
		},
		ValidatorFlags:          []string{},
		EpochsToRun:             uint64(epochsToRun),
		TestSync:                true,
		TestDeposits:            true,
		UseFixedPeerIDs:         true,
		UseValidatorCrossClient: crossClient,
		UsePrysmShValidator:     usePrysmSh,
		UsePprof:                !longRunning,
		TracingSinkEndpoint:     tracingEndpoint,
		Evaluators:              evals,
	}

	newTestRunner(t, testConfig).run()
}
