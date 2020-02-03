package endtoend

import (
	"testing"

	ev "github.com/prysmaticlabs/prysm/endtoend/evaluators"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestEndToEnd_DemoConfig(t *testing.T) {
	t.Skip("Demo is essentially mainnet and too much to run in e2e at the moment")
	testutil.ResetCache()
	params.UseDemoBeaconConfig()

	demoConfig := &end2EndConfig{
		beaconFlags:    append(featureconfig.E2EBeaconChainFlags, "--custom-genesis-delay=60"),
		validatorFlags: featureconfig.E2EValidatorFlags,
		epochsToRun:    5,
		numBeaconNodes: 2,
		numValidators:  params.BeaconConfig().MinGenesisActiveValidatorCount,
		evaluators: []ev.Evaluator{
			ev.ValidatorsAreActive,
			ev.ValidatorsParticipating,
			ev.FinalizationOccurs,
		},
	}
	runEndToEndTest(t, demoConfig)
}
