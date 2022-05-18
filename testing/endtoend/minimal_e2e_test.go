package endtoend

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/prysmaticlabs/prysm/config/params"
	ev "github.com/prysmaticlabs/prysm/testing/endtoend/evaluators"
	e2eParams "github.com/prysmaticlabs/prysm/testing/endtoend/params"
	"github.com/prysmaticlabs/prysm/testing/endtoend/types"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestEndToEnd_MinimalConfig(t *testing.T) {
	e2eMinimal(t, false, 3)
}

func TestEndToEnd_MinimalConfig_Web3Signer(t *testing.T) {
	e2eMinimal(t, true, 0)
}

func e2eMinimal(t *testing.T, useWeb3RemoteSigner bool, extraEpochs uint64) {
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.E2ETestConfig().Copy())

	require.NoError(t, e2eParams.Init(e2eParams.StandardBeaconCount))

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
		//ev.TransactionsPresent, TODO: Renable Transaction evaluator once it tx pool issues are fixed.
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
		UseWeb3RemoteSigner: useWeb3RemoteSigner,
		TracingSinkEndpoint: tracingEndpoint,
		Evaluators:          evals,
		Seed:                int64(seed),
		ExtraEpochs:         extraEpochs,
	}

	newTestRunner(t, testConfig).run()
}
