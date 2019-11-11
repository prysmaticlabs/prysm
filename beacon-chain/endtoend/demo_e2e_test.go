package endtoend

import (
	"testing"

	ev "github.com/prysmaticlabs/prysm/beacon-chain/endtoend/evaluators"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestEndToEnd_DemoConfig(t *testing.T) {
	testutil.ResetCache()
	params.UseDemoBeaconConfig()
	demoConfig := &end2EndConfig{
		minimalConfig:  false,
		epochsToRun:    8,
		numBeaconNodes: 4,
		numValidators:  params.BeaconConfig().MinGenesisActiveValidatorCount,
		evaluators: []ev.Evaluator{
			{
				Name:       "validators_active",
				Evaluation: ev.ValidatorsAreActive,
			},
			{
				Name:       "checkpoint_finalizes",
				Evaluation: ev.FinalizationOccurs,
			},
		},
	}
	runEndToEndTest(t, demoConfig)
}
