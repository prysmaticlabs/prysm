package beacon

import (
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestMain(m *testing.M) {
	// Use minimal config to reduce test setup time.
	reset := params.OverrideBeaconConfigWithReset(params.MinimalSpecConfig())
	flags.Init(&flags.GlobalFlags{
		MaxPageSize: 250,
	})
	retVal := m.Run()
	reset()
	os.Exit(retVal)
}
