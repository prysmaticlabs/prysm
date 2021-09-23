package endtoend

import (
	"testing"

	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	ev "github.com/prysmaticlabs/prysm/testing/endtoend/evaluators"
	e2eParams "github.com/prysmaticlabs/prysm/testing/endtoend/params"
	"github.com/prysmaticlabs/prysm/testing/endtoend/types"
)

func TestEndToEnd_Slashing_MinimalConfig(t *testing.T) {
	t.Skip("To be replaced with the new slasher implementation")

	params.UseE2EConfig()
	require.NoError(t, e2eParams.Init(e2eParams.StandardBeaconCount))

	testConfig := &types.E2EConfig{
		BeaconFlags:    []string{},
		ValidatorFlags: []string{},
		EpochsToRun:    4,
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

	newTestRunner(t, testConfig).run()
}
