package slotutil

import (
	"context"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestCountdownToGenesis(t *testing.T) {
	hook := logTest.NewGlobal()
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.GenesisCountdownInterval = time.Millisecond * 500
	params.OverrideBeaconConfig(config)

	firstStringResult := "1s until chain genesis"
	genesisReached := "Chain genesis time reached"
	CountdownToGenesis(
		context.Background(),
		roughtime.Now().Add(2*time.Second),
		params.BeaconConfig().MinGenesisActiveValidatorCount,
	)
	require.LogsContain(t, hook, firstStringResult)
	require.LogsContain(t, hook, genesisReached)
}
