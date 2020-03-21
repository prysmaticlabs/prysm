package endtoend

import (
	"testing"

	ev "github.com/prysmaticlabs/prysm/endtoend/evaluators"
	"github.com/prysmaticlabs/prysm/endtoend/types"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestEndToEnd_Slashing_MinimalConfig(t *testing.T) {
	testutil.ResetCache()
	params.UseMinimalConfig()

	minimalConfig := &types.E2EConfig{
		BeaconFlags:    []string{"--minimal-config", "--custom-genesis-delay=10"},
		ValidatorFlags: []string{"--minimal-config"},
		EpochsToRun:    2,
		NumBeaconNodes: 2,
		TestSync:       false,
		TestSlasher:    true,
		Evaluators: []types.Evaluator{
			ev.PeersConnect,
			ev.ValidatorsSlashed,
			ev.SlashedValidatorsLoseBalance,
			ev.InjectDoubleVote,
		},
	}
	runEndToEndTest(t, minimalConfig)
}
