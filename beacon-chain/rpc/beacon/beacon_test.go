package beacon

import (
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestMain(m *testing.M) {
	// Use minimal config to reduce test setup time.
	prevConfig := params.BeaconConfig().Copy()
	params.OverrideBeaconConfig(params.MinimalSpecConfig())
	flags.Init(&flags.GlobalFlags{
		MinimumSyncPeers: 30,
	})

	retVal := m.Run()

	// Reset configuration.
	params.OverrideBeaconConfig(prevConfig)
	os.Exit(retVal)
}
