package endtoend

import (
	"testing"

	ev "github.com/prysmaticlabs/prysm/endtoend/evaluators"
	e2eParams "github.com/prysmaticlabs/prysm/endtoend/params"
	"github.com/prysmaticlabs/prysm/endtoend/types"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestEndToEnd_Slashing_MinimalConfig(t *testing.T) {
	testutil.ResetCache()
	params.UseE2EConfig()
	if err := e2eParams.Init(e2eParams.StandardBeaconCount); err != nil {
		t.Fatal(err)
	}

	minimalConfig := &types.E2EConfig{
		BeaconFlags:    []string{},
		ValidatorFlags: []string{},
		EpochsToRun:    3,
		TestSync:       false,
		TestSlasher:    true,
		TestDeposits:   false,
		Evaluators: []types.Evaluator{
			ev.PeersConnect,
			ev.HealthzCheck,
			ev.ValidatorsSlashed,
			ev.SlashedValidatorsLoseBalance,
			ev.InjectDoubleVote,
			ev.ProposeDoubleBlock,
		},
	}

	runEndToEndTest(t, minimalConfig)
}
