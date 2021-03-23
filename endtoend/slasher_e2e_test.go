package endtoend

import (
	"context"
	"os"
	"strconv"
	"testing"

	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	slashersimulator "github.com/prysmaticlabs/prysm/beacon-chain/slasher/simulator"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestEndToEnd_Slasher(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	testutil.ResetCache()
	params.UseE2EConfig()

	// Run for 10 epochs if not in long-running to confirm long-running has no issues.
	simulatorParams := slashersimulator.DefaultParams()
	var err error
	epochStr, longRunning := os.LookupEnv("E2E_EPOCHS")
	if longRunning {
		epochsToRun, err := strconv.Atoi(epochStr)
		require.NoError(t, err)
		simulatorParams.NumEpochs = uint64(epochsToRun)
	}

	beaconState, err := testutil.NewBeaconState()
	require.NoError(t, err)
	mockChain := &mock.ChainService{State: beaconState}

	beaconDB := dbtest.SetupDB(t)
	sim, err := slashersimulator.New(ctx, &slashersimulator.ServiceConfig{
		Params:        simulatorParams,
		Database:      beaconDB,
		StateNotifier: &mock.MockStateNotifier{},
		StateFetcher:  mockChain,
		StateGen:      stategen.New(beaconDB),
	})
	require.NoError(t, err)
	sim.Start()
	err = sim.Stop()
	require.NoError(t, err)
	require.LogsDoNotContain(t, hook, "ERROR")
	require.LogsDoNotContain(t, hook, "Did not detect")
	require.LogsContain(t, hook, "Correctly detected simulated proposer slashing")
	require.LogsContain(t, hook, "Correctly detected simulated attester slashing")
}
