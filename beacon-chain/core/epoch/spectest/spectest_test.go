package spectest

import (
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestMain(m *testing.M) {
	// TODO(2312): remove this and use the mainnet count.
	c := params.BeaconConfig().Copy()
	c.MinGenesisActiveValidatorCount = 16384
	reset := params.OverrideBeaconConfigWithReset(c)
	retVal := m.Run()
	reset()
	os.Exit(retVal)
}
