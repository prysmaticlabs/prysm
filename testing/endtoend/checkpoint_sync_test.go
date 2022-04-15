package endtoend

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	e2types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	ev "github.com/prysmaticlabs/prysm/testing/endtoend/evaluators"
	e2eParams "github.com/prysmaticlabs/prysm/testing/endtoend/params"
	"github.com/prysmaticlabs/prysm/testing/endtoend/types"
	"github.com/prysmaticlabs/prysm/testing/require"
)

// This test customizes the minimal config in order to artificially shorten the weak subjectivity period
// so that the state used will not be genesis despite there only being 10 epochs of history.
func TestCheckpointSync_CustomConfig(t *testing.T) {
	// Run for 10 epochs if not in long-running to confirm long-running has no issues.
	var err error
	epochsToRun := 10
	epochStr, longRunning := os.LookupEnv("E2E_EPOCHS")
	if longRunning {
		epochsToRun, err = strconv.Atoi(epochStr)
		require.NoError(t, err)
	}

	cfg := params.E2ETestConfig()
	cfg.BellatrixForkEpoch = 10000
	// setting this to 1 should change the weak subjectivity computation,
	// so the computed weak subjectivity checkpoint will just be a few epochs before head
	cfg.MinValidatorWithdrawabilityDelay = e2types.Epoch(epochsToRun / 2)
	cfg.SlotsPerEpoch = 6
	cfg.SecondsPerSlot = 6
	params.OverrideBeaconConfig(cfg)
	require.NoError(t, e2eParams.Init(e2eParams.StandardBeaconCount))

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
		ev.PeersCheck,
		ev.ProcessesDepositsInBlocks,
		ev.VerifyBlockGraffiti,
		ev.ActivatesDepositedValidators,
		ev.DepositedValidatorsAreActive,
		ev.ProposeVoluntaryExit,
		ev.ValidatorHasExited,
		ev.ValidatorsVoteWithTheMajority,
		ev.ColdStateCheckpoint,
		ev.ForkTransition,
		ev.APIMiddlewareVerifyIntegrity,
		ev.APIGatewayV1Alpha1VerifyIntegrity,
		ev.FinishedSyncing,
		ev.AllNodesHaveSameHead,
		ev.ValidatorSyncParticipation,
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
		UseFixedPeerIDs:     true,
		UsePrysmShValidator: false,
		UsePprof:            !longRunning,
		TracingSinkEndpoint: tracingEndpoint,
		Evaluators:          evals,
		Seed:                int64(seed),
		BeaconChainConfig:   cfg,
		LeaveRunning: false,
	}

	newTestRunner(t, testConfig).run()
}
