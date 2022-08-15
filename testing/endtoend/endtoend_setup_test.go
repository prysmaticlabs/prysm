package endtoend

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/config/params"
	ev "github.com/prysmaticlabs/prysm/v3/testing/endtoend/evaluators"
	e2eParams "github.com/prysmaticlabs/prysm/v3/testing/endtoend/params"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/types"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func e2eMinimal(t *testing.T, cfgo ...types.E2EConfigOpt) *testRunner {
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.E2ETestConfig().Copy())
	require.NoError(t, e2eParams.Init(t, e2eParams.StandardBeaconCount))

	// Run for 12 epochs if not in long-running to confirm long-running has no issues.
	var err error
	epochsToRun := 10
	epochStr, longRunning := os.LookupEnv("E2E_EPOCHS")
	if longRunning {
		epochsToRun, err = strconv.Atoi(epochStr)
		require.NoError(t, err)
	}
	seed := 0
	seedStr, isValid := os.LookupEnv("E2E_SEED")
	if isValid {
		seed, err = strconv.Atoi(seedStr)
		require.NoError(t, err)
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
		ev.VerifyBlockGraffiti,
		ev.PeersCheck,
		ev.ProposeVoluntaryExit,
		ev.ValidatorHasExited,
		ev.ProcessesDepositsInBlocks,
		ev.ActivatesDepositedValidators,
		ev.DepositedValidatorsAreActive,
		ev.ValidatorsVoteWithTheMajority,
		ev.ColdStateCheckpoint,
		ev.AltairForkTransition,
		ev.BellatrixForkTransition,
		ev.APIMiddlewareVerifyIntegrity,
		ev.APIGatewayV1Alpha1VerifyIntegrity,
		ev.FinishedSyncing,
		ev.AllNodesHaveSameHead,
		ev.ValidatorSyncParticipation,
		ev.FeeRecipientIsPresent,
		//ev.TransactionsPresent, TODO: Re-enable Transaction evaluator once it tx pool issues are fixed.
	}
	testConfig := &types.E2EConfig{
		BeaconFlags: []string{
			fmt.Sprintf("--slots-per-archive-point=%d", params.BeaconConfig().SlotsPerEpoch*16),
			fmt.Sprintf("--tracing-endpoint=http://%s", tracingEndpoint),
			"--enable-tracing",
			"--trace-sample-fraction=1.0",
		},
		ValidatorFlags:      []string{},
		EpochsToRun:         uint64(epochsToRun),
		TestSync:            true,
		TestFeature:         true,
		TestDeposits:        true,
		UsePrysmShValidator: false,
		UsePprof:            !longRunning,
		TracingSinkEndpoint: tracingEndpoint,
		Evaluators:          evals,
		EvalInterceptor:     defaultInterceptor,
		Seed:                int64(seed),
	}
	for _, o := range cfgo {
		o(testConfig)
	}

	return newTestRunner(t, testConfig)
}

func e2eMainnet(t *testing.T, usePrysmSh, useMultiClient bool, cfgo ...types.E2EConfigOpt) *testRunner {
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.E2EMainnetTestConfig())
	if useMultiClient {
		require.NoError(t, e2eParams.InitMultiClient(t, e2eParams.StandardBeaconCount, e2eParams.StandardLighthouseNodeCount))
	} else {
		require.NoError(t, e2eParams.Init(t, e2eParams.StandardBeaconCount))
	}
	// Run for 10 epochs if not in long-running to confirm long-running has no issues.
	var err error
	epochsToRun := 10
	epochStr, longRunning := os.LookupEnv("E2E_EPOCHS")
	if longRunning {
		epochsToRun, err = strconv.Atoi(epochStr)
		require.NoError(t, err)
	}
	_, crossClient := os.LookupEnv("RUN_CROSS_CLIENT")
	seed := 0
	seedStr, isValid := os.LookupEnv("E2E_SEED")
	if isValid {
		seed, err = strconv.Atoi(seedStr)
		require.NoError(t, err)
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
		ev.AltairForkTransition,
		ev.BellatrixForkTransition,
		ev.APIMiddlewareVerifyIntegrity,
		ev.APIGatewayV1Alpha1VerifyIntegrity,
		ev.FinishedSyncing,
		ev.AllNodesHaveSameHead,
		ev.FeeRecipientIsPresent,
		//ev.TransactionsPresent, TODO: Re-enable Transaction evaluator once it tx pool issues are fixed.
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
		TestFeature:             true,
		TestDeposits:            true,
		UseFixedPeerIDs:         true,
		UseValidatorCrossClient: crossClient,
		UsePrysmShValidator:     usePrysmSh,
		UsePprof:                !longRunning,
		TracingSinkEndpoint:     tracingEndpoint,
		Evaluators:              evals,
		EvalInterceptor:         defaultInterceptor,
		Seed:                    int64(seed),
	}
	for _, o := range cfgo {
		o(testConfig)
	}
	return newTestRunner(t, testConfig)
}

func scenarioEvals() []types.Evaluator {
	return []types.Evaluator{
		ev.PeersConnect,
		ev.HealthzCheck,
		ev.MetricsCheck,
		ev.ValidatorsParticipatingAtEpoch(2),
		ev.FinalizationOccurs(3),
		ev.VerifyBlockGraffiti,
		ev.ProposeVoluntaryExit,
		ev.ValidatorHasExited,
		ev.ColdStateCheckpoint,
		ev.AltairForkTransition,
		ev.BellatrixForkTransition,
		ev.APIMiddlewareVerifyIntegrity,
		ev.APIGatewayV1Alpha1VerifyIntegrity,
		ev.FinishedSyncing,
		ev.AllNodesHaveSameHead,
		ev.ValidatorSyncParticipation,
	}
}

func scenarioEvalsMulti() []types.Evaluator {
	return []types.Evaluator{
		ev.PeersConnect,
		ev.HealthzCheck,
		ev.MetricsCheck,
		ev.ValidatorsParticipatingAtEpoch(2),
		ev.FinalizationOccurs(3),
		ev.ProposeVoluntaryExit,
		ev.ValidatorHasExited,
		ev.ColdStateCheckpoint,
		ev.AltairForkTransition,
		ev.BellatrixForkTransition,
		ev.APIMiddlewareVerifyIntegrity,
		ev.APIGatewayV1Alpha1VerifyIntegrity,
		ev.FinishedSyncing,
		ev.AllNodesHaveSameHead,
	}
}
