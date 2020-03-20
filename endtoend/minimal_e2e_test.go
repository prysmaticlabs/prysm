package endtoend

import (
	"testing"

	ev "github.com/prysmaticlabs/prysm/endtoend/evaluators"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestEndToEnd_MinimalConfig(t *testing.T) {
	t.Skip("To be resolved until 5119 gets in")
	testutil.ResetCache()
	params.UseMinimalConfig()

	minimalConfig := &end2EndConfig{
		beaconFlags:    append(featureconfig.E2EBeaconChainFlags, "--minimal-config", "--custom-genesis-delay=15"),
		validatorFlags: append(featureconfig.E2EValidatorFlags, "--minimal-config"),
		epochsToRun:    6,
		numBeaconNodes: 4,
		numValidators:  params.BeaconConfig().MinGenesisActiveValidatorCount,
		testSync:       true,
		evaluators: []ev.Evaluator{
			ev.PeersConnect,
			ev.ValidatorsAreActive,
			ev.ValidatorsParticipating,
			ev.FinalizationOccurs,
		},
	}
	runEndToEndTest(t, minimalConfig)
}
