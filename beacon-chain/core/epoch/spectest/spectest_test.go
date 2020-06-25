package spectest

import (
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestMain(m *testing.M) {
	prevConfig := params.BeaconConfig().Copy()
	c := params.BeaconConfig()
	c.MinGenesisActiveValidatorCount = 16384
	params.OverrideBeaconConfig(c)

	retVal := m.Run()
	params.OverrideBeaconConfig(prevConfig)
	os.Exit(retVal)
}
