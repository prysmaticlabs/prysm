package endtoend

import (
	"testing"

	ev "github.com/prysmaticlabs/prysm/endtoend/evaluators"
	e2eParams "github.com/prysmaticlabs/prysm/endtoend/params"
	"github.com/prysmaticlabs/prysm/endtoend/types"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestEndToEnd_Slasher_MinimalConfig(t *testing.T) {
	params.UseE2EConfig()
	require.NoError(t, e2eParams.Init(e2eParams.StandardBeaconCount))

	testConfig := &types.E2EConfig{
		BeaconFlags: []string{
			"--slasher",
		},
		ValidatorFlags: []string{},
		EpochsToRun:    4,
		TestSync:       false,
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
