package endtoend

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/prysmaticlabs/prysm/config/params"
	ev "github.com/prysmaticlabs/prysm/testing/endtoend/evaluators"
	"github.com/prysmaticlabs/prysm/testing/endtoend/helpers"
	e2eParams "github.com/prysmaticlabs/prysm/testing/endtoend/params"
	"github.com/prysmaticlabs/prysm/testing/endtoend/types"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestEndToEnd_MainnetConfig(t *testing.T) {
	e2eMainnet(t, false /*usePrysmSh*/)
}

func e2eMainnet(t *testing.T, usePrysmSh bool) {
	params.UseE2EMainnetConfig()
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
	tracingPort := e2eParams.TestParams.Ports.JaegerTracingPort
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
