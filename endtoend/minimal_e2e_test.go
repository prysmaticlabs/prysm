package endtoend

import (
	"testing"

	ev "github.com/prysmaticlabs/prysm/endtoend/evaluators"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestEndToEnd_MinimalConfig(t *testing.T) {
	testutil.ResetCache()
	params.UseMinimalConfig()

	minimalConfig := &end2EndConfig{
		beaconFlags:  []string{
			"--minimal-config",
			"--enable-ssz-cache",
			"--cache-proposer-indices",
			"--cache-filtered-block-tree",
			"--enable-skip-slots-cache",
			//"--enable-attestation-cache",
		},
		validatorFlags:  []string{
			"--minimal-config",
		},
		epochsToRun:    5,
		numBeaconNodes: 4,
		numValidators:  params.BeaconConfig().MinGenesisActiveValidatorCount,
		evaluators: []ev.Evaluator{
			ev.ValidatorsAreActive,
			ev.ValidatorsParticipating,
			ev.FinalizationOccurs,
		},
	}
	runEndToEndTest(t, minimalConfig)
}
