package spectest

import (
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestMain(m *testing.M) {
	run := func() int {
		prevConfig := params.BeaconConfig().Copy()
		defer params.OverrideBeaconConfig(prevConfig)
		c := params.BeaconConfig()
		c.MinGenesisActiveValidatorCount = 16384
		params.OverrideBeaconConfig(c)

		return m.Run()
	}
	os.Exit(run())
}
