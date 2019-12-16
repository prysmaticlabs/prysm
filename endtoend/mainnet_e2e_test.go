package endtoend

import (
	"testing"

	ev "github.com/prysmaticlabs/prysm/endtoend/evaluators"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestEndToEnd_MainnetConfig(t *testing.T) {
	testutil.ResetCache()
	params.UseMainnetConfig()

	mainnetConfig := &end2EndConfig{
		beaconConfig:   "mainnet",
		epochsToRun:    5,
		numBeaconNodes: 1,
		numValidators:  params.BeaconConfig().MinGenesisActiveValidatorCount,
		evaluators: []ev.Evaluator{
			ev.ValidatorsAreActive,
			ev.ValidatorsParticipating,
			ev.FinalizationOccurs,
		},
	}
	runEndToEndTest(t, mainnetConfig)
}
