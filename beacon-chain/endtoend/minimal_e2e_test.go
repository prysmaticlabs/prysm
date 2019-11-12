package endtoend

import (
	"testing"

	ev "github.com/prysmaticlabs/prysm/beacon-chain/endtoend/evaluators"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestEndToEnd_MinimalConfig(t *testing.T) {
	testutil.ResetCache()
	params.UseMinimalConfig()

	minimalConfig := &end2EndConfig{
		minimalConfig:  true,
		epochsToRun:    6,
		numBeaconNodes: 4,
		numValidators:  params.BeaconConfig().MinGenesisActiveValidatorCount,
		evaluators: []ev.Evaluator{
			ev.ValidatorsAreActive,
			ev.FinalizationOccurs,
		},
	}
	runEndToEndTest(t, minimalConfig)
}
