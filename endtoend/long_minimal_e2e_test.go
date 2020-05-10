package endtoend

import (
	"os"
	"strconv"
	"testing"

	ev "github.com/prysmaticlabs/prysm/endtoend/evaluators"
	e2eParams "github.com/prysmaticlabs/prysm/endtoend/params"
	"github.com/prysmaticlabs/prysm/endtoend/types"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestEndToEnd_Long_MinimalConfig(t *testing.T) {
	testutil.ResetCache()
	params.UseMinimalConfig()

	epochsToRun := 20
	var err error
	epochStr, ok := os.LookupEnv("E2E_EPOCHS")
	if ok {
		epochsToRun, err = strconv.Atoi(epochStr)
		if err != nil {
			t.Fatal(err)
		}
	} else {
		t.Skip("E2E_EPOCHS not set")
	}

	minimalConfig := &types.E2EConfig{
		BeaconFlags:    []string{"--minimal-config", "--custom-genesis-delay=25"},
		ValidatorFlags: []string{"--minimal-config"},
		EpochsToRun:    uint64(epochsToRun),
		TestSync:       false,
		TestDeposits:   true,
		TestSlasher:    true,
		Evaluators: []types.Evaluator{
			ev.PeersConnect,
			ev.HealthzCheck,
			ev.ValidatorsAreActive,
			ev.ValidatorsParticipating,
			ev.FinalizationOccurs,
			ev.MetricsCheck,
			ev.ProcessesDepositedValidators,
			ev.DepositedValidatorsAreActive,
		},
	}
	if err := e2eParams.Init(4); err != nil {
		t.Fatal(err)
	}

	runEndToEndTest(t, minimalConfig)
}
