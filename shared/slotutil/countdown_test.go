package slotutil

import (
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestCountdownToGenesis(t *testing.T) {
	hook := logTest.NewGlobal()
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.GenesisCountdownInterval = time.Second
	params.OverrideBeaconConfig(config)

	firstStringResult := "1 minute(s) until chain genesis"
	CountdownToGenesis(roughtime.Now().Add(2*time.Second), params.BeaconConfig().MinGenesisActiveValidatorCount)
	testutil.AssertLogsContain(t, hook, firstStringResult)
}
