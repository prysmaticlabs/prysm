package endtoend

import (
	"testing"

	ev "github.com/prysmaticlabs/prysm/endtoend/evaluators"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestEndToEnd_DemoConfig(t *testing.T) {
	testutil.ResetCache()
	params.UseDemoBeaconConfig()

	demoConfig := &end2EndConfig{
		beaconFlags: []string{
			"--enable-ssz-cache",
			"--cache-proposer-indices",
			"--cache-filtered-block-tree",
		},
		epochsToRun:    5,
		numBeaconNodes: 1,
		numValidators:  params.BeaconConfig().MinGenesisActiveValidatorCount,
		evaluators: []ev.Evaluator{
			ev.ValidatorsAreActive,
			ev.ValidatorsParticipating,
			ev.FinalizationOccurs,
		},
	}
	runEndToEndTest(t, demoConfig)
}
