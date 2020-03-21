package endtoend

import (
	"testing"

	ev "github.com/prysmaticlabs/prysm/endtoend/evaluators"
	"github.com/prysmaticlabs/prysm/endtoend/types"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestEndToEnd_AntiFlake_MinimalConfig(t *testing.T) {
	testutil.ResetCache()
	params.UseMinimalConfig()

	minimalConfig := &types.E2EConfig{
		BeaconFlags:    []string{"--minimal-config", "--custom-genesis-delay=10"},
		ValidatorFlags: []string{"--minimal-config"},
		EpochsToRun:    3,
		NumBeaconNodes: 4,
		TestSync:       false,
		TestSlasher:    false,
		Evaluators: []types.Evaluator{
			ev.PeersConnect,
			ev.ValidatorsAreActive,
		},
	}
	// Running this test twice to test the quickest conditions (3 epochs) twice.
	runEndToEndTest(t, minimalConfig)
	runEndToEndTest(t, minimalConfig)
}
