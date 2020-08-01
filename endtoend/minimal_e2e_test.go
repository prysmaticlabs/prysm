package endtoend

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	ev "github.com/prysmaticlabs/prysm/endtoend/evaluators"
	e2eParams "github.com/prysmaticlabs/prysm/endtoend/params"
	"github.com/prysmaticlabs/prysm/endtoend/types"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestEndToEnd_MinimalConfig(t *testing.T) {
	testutil.ResetCache()
	params.UseE2EConfig()
	if err := e2eParams.Init(e2eParams.StandardBeaconCount); err != nil {
		t.Fatal(err)
	}

	// Run for 10 epochs if not in long-running to confirm long-running has no issues.
	epochsToRun := 10
	var err error
	epochStr, ok := os.LookupEnv("E2E_EPOCHS")
	if ok {
		epochsToRun, err = strconv.Atoi(epochStr)
		if err != nil {
			t.Fatal(err)
		}
	}

	minimalConfig := &types.E2EConfig{
		BeaconFlags: []string{
			fmt.Sprintf("--slots-per-archive-point=%d", params.BeaconConfig().SlotsPerEpoch*16),
		},
		ValidatorFlags: []string{},
		EpochsToRun:    uint64(epochsToRun),
		TestSync:       true,
		TestDeposits:   true,
		TestSlasher:    true,
		Evaluators: []types.Evaluator{
			ev.PeersConnect,
			ev.HealthzCheck,
			ev.MetricsCheck,
			ev.ValidatorsAreActive,
			ev.ValidatorsParticipating,
			ev.FinalizationOccurs,
			ev.ProcessesDepositsInBlocks,
			ev.ActivatesDepositedValidators,
			ev.DepositedValidatorsAreActive,
			ev.ProposeVoluntaryExit,
			ev.ValidatorHasExited,
			ev.ColdStateCheckpoint,
		},
	}

	runEndToEndTest(t, minimalConfig)
}
