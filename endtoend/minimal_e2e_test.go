package endtoend

import (
	"testing"

	e2eParams "github.com/prysmaticlabs/prysm/endtoend/params"

	ev "github.com/prysmaticlabs/prysm/endtoend/evaluators"
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

	minimalConfig := &types.E2EConfig{
		BeaconFlags:    []string{},
		ValidatorFlags: []string{},
		EpochsToRun:    10,
		TestSync:       true,
		TestSlasher:    true,
		TestDeposits:   false,
		Evaluators: []types.Evaluator{
			ev.PeersConnect,
			ev.HealthzCheck,
			ev.MetricsCheck,
			ev.ValidatorsAreActive,
			ev.ValidatorsParticipating,
			ev.FinalizationOccurs,
			ev.ProposeVoluntaryExit,
			ev.ValidatorHasExited,
		},
	}

	runEndToEndTest(t, minimalConfig)
}
