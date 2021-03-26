package endtoend

import (
	"testing"

	ev "github.com/prysmaticlabs/prysm/endtoend/evaluators"
	e2eParams "github.com/prysmaticlabs/prysm/endtoend/params"
	"github.com/prysmaticlabs/prysm/endtoend/types"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestEndToEnd_Connectivity(t *testing.T) {
	// This test isolates all the preliminary networking setup necessary for other e2e tests.
	// This allows easier checks for connectivity, discovery and peering issues.
	testutil.ResetCache()
	params.UseE2EConfig()
	require.NoError(t, e2eParams.Init(e2eParams.StandardBeaconCount))

	testConfig := &types.E2EConfig{
		BeaconFlags:    []string{},
		ValidatorFlags: []string{},
		EpochsToRun:    2,
		Evaluators: []types.Evaluator{
			ev.PeersConnect,
		},
	}

	newTestRunner(t, testConfig).run()
}
