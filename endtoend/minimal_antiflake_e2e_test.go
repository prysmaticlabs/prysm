package endtoend

import (
	"testing"

	ev "github.com/prysmaticlabs/prysm/endtoend/evaluators"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestEndToEnd_AntiFlake_MinimalConfig(t *testing.T) {
	testutil.ResetCache()
	params.UseMinimalConfig()

	minimalConfig := &end2EndConfig{
		beaconFlags:    append(featureconfig.E2EBeaconChainFlags, "--minimal-config", "--custom-genesis-delay=10"),
		validatorFlags: append(featureconfig.E2EValidatorFlags, "--minimal-config"),
		epochsToRun:    2,
		numBeaconNodes: 4,
		numValidators:  params.BeaconConfig().MinGenesisActiveValidatorCount,
		testSync:       false,
		testSlasher:    false,
		evaluators: []ev.Evaluator{
			ev.PeersConnect,
			ev.ValidatorsAreActive,
		},
	}
	// Running this test twice to test the quickest conditions (3 epochs) twice.
	runEndToEndTest(t, minimalConfig)
	runEndToEndTest(t, minimalConfig)
}
